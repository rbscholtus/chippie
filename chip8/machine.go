package chip8

import "errors"

// Memory layout.
const (
	// ProgramStart is where ROMs are loaded and execution starts.
	ProgramStart = 0x200
	// numRegisters is the number of general registers V0..VF.
	numRegisters = 16
	// flagReg is VF, the flag register.
	flagReg = 0xF
	// stackDepth is the number of nested subroutine calls. The COSMAC VIP
	// had room for 12; later platforms have 16. 16 is used for all.
	stackDepth = 16
	// NumKeys is the number of keys on the hex keypad (0..F).
	NumKeys = 16
)

// Errors reported by the machine. Errors from running a ROM halt the machine
// and are reported in [CPUState.Err]; the machine never panics on bad input.
var (
	ErrROMTooLarge         = errors.New("chip8: ROM does not fit in memory")
	ErrUnknownOpcode       = errors.New("chip8: unknown opcode")
	ErrStackOverflow       = errors.New("chip8: stack overflow")
	ErrStackUnderflow      = errors.New("chip8: return with empty stack")
	ErrUnsupportedPlatform = errors.New("chip8: platform not implemented yet")
)

// noKey marks that FX0A is waiting for a key to be pressed.
const noKey = -1

// Machine is one emulated CHIP-8 computer.
type Machine struct {
	profile Profile
	mem     []byte // sized for the platform
	memMask uint16 // len(mem)-1, for wrapping addresses

	v     [numRegisters]byte
	i     uint16
	pc    uint16
	stack [stackDepth]uint16
	sp    int
	dt    byte // delay timer
	st    byte // sound timer

	keys [NumKeys]bool

	// keysRead records which keys the program has checked with EX9E/EXA1,
	// and anyKey whether it waited for any key with FX0A: the UI shows them
	// as the game's controls.
	keysRead uint16
	anyKey   bool

	// FX0A state: waiting is true while FX0A blocks. lastKeys holds the keys
	// at the previous check, so only a new press counts. waitKey is the key
	// that was pressed and must now be released, or noKey.
	waiting  bool
	waitKey  int
	lastKeys [NumKeys]bool

	disp display
	rng  rng

	frame     uint32
	stopFrame bool // set by an instruction to end the current frame early
	err       error

	trace *TraceBuffer
}

// New returns a machine set up for the profile, with memory and display
// sized for its platform.
func New(p Profile) *Machine {
	m := &Machine{}
	m.Reset(p)

	return m
}

// Reset clears the machine and applies the profile. Memory and display
// buffers are reused when the platform needs the same sizes, and reallocated
// otherwise. The trace buffer is kept and cleared.
func (m *Machine) Reset(p Profile) {
	info := p.Platform.info()

	m.profile = p
	if len(m.mem) == info.memSize {
		clear(m.mem)
	} else {
		m.mem = make([]byte, info.memSize)
	}

	m.memMask = uint16(info.memSize - 1)

	m.v = [numRegisters]byte{}
	m.i = 0
	m.pc = ProgramStart
	m.stack = [stackDepth]uint16{}
	m.sp = 0
	m.dt, m.st = 0, 0
	m.keys = [NumKeys]bool{}
	m.keysRead, m.anyKey = 0, false
	m.waiting, m.waitKey, m.lastKeys = false, noKey, [NumKeys]bool{}
	m.frame = 0
	m.stopFrame = false
	m.err = nil

	m.disp.reset(info.maxWidth, info.maxHeight, loresWidth, loresHeight)
	m.rng.seed(p.Seed)
	copy(m.mem[FontAddr:], smallFont[:])

	if m.trace != nil {
		m.trace.clear()
	}
}

// LoadROM resets the machine with its current profile and copies the ROM
// into memory at [ProgramStart].
func (m *Machine) LoadROM(rom []byte) error {
	if !m.profile.Platform.Implemented() {
		return ErrUnsupportedPlatform
	}

	if len(rom) > len(m.mem)-ProgramStart {
		return ErrROMTooLarge
	}

	m.Reset(m.profile)
	copy(m.mem[ProgramStart:], rom)

	return nil
}

// Poke writes one byte of memory. Tests use it, for example to set 0x1FF to
// skip a test ROM's menu. Addresses wrap at the memory size.
func (m *Machine) Poke(addr uint16, v byte) { m.mem[addr&m.memMask] = v }

// Peek reads one byte of memory. Addresses wrap at the memory size.
func (m *Machine) Peek(addr uint16) byte { return m.mem[addr&m.memMask] }

// SetKey sets the state of a hex key (0..F). Other values are ignored.
func (m *Machine) SetKey(key uint8, down bool) {
	if int(key) < NumKeys {
		m.keys[key] = down
	}
}

// KeyDown reports whether a hex key (0..F) is down.
func (m *Machine) KeyDown(key uint8) bool { return int(key) < NumKeys && m.keys[key] }

// SetTicksPerFrame changes the speed while running (the speed knob).
// Values below 1 mean the platform's default.
func (m *Machine) SetTicksPerFrame(n int) { m.profile.TicksPerFrame = n }

// SetQuirks changes the quirks while running. Changing the platform needs
// Reset, because memory and display sizes may change.
func (m *Machine) SetQuirks(q Quirks) { m.profile.Quirks = q }

// KeysRead reports which hex keys the program has checked since the last
// reset (bit k for key k), and whether it has waited for any key (FX0A).
// The UI shows them, so players can find a game's controls.
func (m *Machine) KeysRead() (uint16, bool) { return m.keysRead, m.anyKey }

// Profile returns the profile the machine is running with.
func (m *Machine) Profile() Profile { return m.profile }

// Display returns a read-only view of the display. It is valid until the
// next call that changes the machine.
func (m *Machine) Display() DisplayView { return m.disp.view() }

// SoundState is what the audio output needs to know.
type SoundState struct {
	Buzzer bool // the buzzer sounds while the sound timer is non-zero
}

// Sound returns the current sound state.
func (m *Machine) Sound() SoundState { return SoundState{Buzzer: m.st > 0} }

// CPUState is a snapshot of the CPU, for tracing, debugging and the UI.
type CPUState struct {
	PC, I      uint16
	V          [numRegisters]byte
	Stack      [stackDepth]uint16
	SP         int
	DelayTimer byte
	SoundTimer byte
	Frame      uint32 // frames run since the last reset
	WaitingKey bool   // FX0A is waiting for a key
	Err        error  // why the machine halted, or nil
}

// State returns a snapshot of the CPU.
func (m *Machine) State() CPUState {
	return CPUState{
		PC: m.pc, I: m.i, V: m.v, Stack: m.stack, SP: m.sp,
		DelayTimer: m.dt, SoundTimer: m.st, Frame: m.frame,
		WaitingKey: m.waiting, Err: m.err,
	}
}

// Halted reports whether the machine stopped because of an error.
func (m *Machine) Halted() bool { return m.err != nil }

// halt stops the machine with an error.
func (m *Machine) halt(err error) {
	m.err = err
	m.stopFrame = true
}
