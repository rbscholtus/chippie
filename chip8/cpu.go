package chip8

import "fmt"

// OpcodeError reports an opcode the platform doesn't have. It matches
// [ErrUnknownOpcode] with errors.Is.
type OpcodeError struct {
	PC uint16 // address of the opcode
	Op uint16
}

func (e *OpcodeError) Error() string {
	return fmt.Sprintf("chip8: unknown opcode %04X at %03X", e.Op, e.PC)
}

// Unwrap returns [ErrUnknownOpcode].
func (e *OpcodeError) Unwrap() error { return ErrUnknownOpcode }

// RunFrame runs one 60 Hz frame: the timers tick once, then up to
// TicksPerFrame instructions run. The frame ends early when the machine
// halts, when FX0A is waiting for a key, or after DXYN with the vblank quirk.
// A halted machine does nothing.
func (m *Machine) RunFrame() {
	if m.err != nil {
		return
	}

	m.frame++
	if m.dt > 0 {
		m.dt--
	}

	if m.st > 0 {
		m.st--
	}

	ticks := m.profile.TicksPerFrame
	if ticks <= 0 {
		ticks = m.profile.Platform.DefaultTicks()
	}

	m.stopFrame = false
	for range ticks {
		m.step()

		if m.stopFrame {
			break
		}
	}
}

// step fetches, records and executes one instruction.
func (m *Machine) step() {
	pc := m.pc
	op := uint16(m.mem[pc&m.memMask])<<8 | uint16(m.mem[(pc+1)&m.memMask])

	if m.trace != nil {
		m.trace.record(TraceEntry{PC: pc, Op: op, I: m.i, Frame: m.frame})
	}

	m.pc = (pc + 2) & m.memMask
	m.execute(pc, op)
}

// execute decodes one opcode and runs its handler.
func (m *Machine) execute(pc, op uint16) { //nolint:funlen // one case per opcode group
	x := int(op >> 8 & 0xF)
	y := int(op >> 4 & 0xF)
	n := op & 0xF
	nn := byte(op)
	nnn := op & 0xFFF

	switch op >> 12 {
	case 0x0:
		m.execute0(pc, op)
	case 0x1:
		m.op1NNN(nnn)
	case 0x2:
		m.op2NNN(nnn)
	case 0x3:
		m.op3XNN(x, nn)
	case 0x4:
		m.op4XNN(x, nn)
	case 0x5:
		m.execute5(pc, op, x, y)
	case 0x6:
		m.op6XNN(x, nn)
	case 0x7:
		m.op7XNN(x, nn)
	case 0x8:
		m.execute8(pc, op, x, y)
	case 0x9:
		if n != 0 {
			m.unknown(pc, op)

			return
		}

		m.op9XY0(x, y)
	case 0xA:
		m.opANNN(nnn)
	case 0xB:
		m.opBNNN(x, nnn)
	case 0xC:
		m.opCXNN(x, nn)
	case 0xD:
		m.opDXYN(x, y, int(n))
	case 0xE:
		m.executeE(pc, op, x)
	case 0xF:
		m.executeF(pc, op, x)
	}
}

// execute0 decodes the 0NNN group. Anything the platform doesn't define is a
// machine-code routine, which isn't supported.
func (m *Machine) execute0(pc, op uint16) {
	switch op {
	case 0x00E0:
		m.op00E0()
	case 0x00EE:
		m.op00EE()
	default:
		if !m.executeSuper0(op) {
			m.unknown(pc, op)
		}
	}
}

// execute5 decodes the 5XYN group.
func (m *Machine) execute5(pc, op uint16, x, y int) {
	switch {
	case op&0xF == 0:
		m.op5XY0(x, y)
	case m.ops&opsXO != 0 && op&0xF == 2:
		m.op5XY2(x, y)
	case m.ops&opsXO != 0 && op&0xF == 3:
		m.op5XY3(x, y)
	default:
		m.unknown(pc, op)
	}
}

// execute8 decodes the 8XYN arithmetic and logic group.
func (m *Machine) execute8(pc, op uint16, x, y int) {
	switch op & 0xF {
	case 0x0:
		m.op8XY0(x, y)
	case 0x1:
		m.op8XY1(x, y)
	case 0x2:
		m.op8XY2(x, y)
	case 0x3:
		m.op8XY3(x, y)
	case 0x4:
		m.op8XY4(x, y)
	case 0x5:
		m.op8XY5(x, y)
	case 0x6:
		m.op8XY6(x, y)
	case 0x7:
		m.op8XY7(x, y)
	case 0xE:
		m.op8XYE(x, y)
	default:
		m.unknown(pc, op)
	}
}

// executeE decodes the EXNN key group.
func (m *Machine) executeE(pc, op uint16, x int) {
	switch byte(op) {
	case 0x9E:
		m.opEX9E(x)
	case 0xA1:
		m.opEXA1(x)
	default:
		m.unknown(pc, op)
	}
}

// executeF decodes the FXNN group.
func (m *Machine) executeF(pc, op uint16, x int) {
	switch byte(op) {
	case 0x07:
		m.opFX07(x)
	case 0x0A:
		m.opFX0A(x)
	case 0x15:
		m.opFX15(x)
	case 0x18:
		m.opFX18(x)
	case 0x1E:
		m.opFX1E(x)
	case 0x29:
		m.opFX29(x)
	case 0x33:
		m.opFX33(x)
	case 0x55:
		m.opFX55(x)
	case 0x65:
		m.opFX65(x)
	default:
		if !m.executeSuperF(x, byte(op)) {
			m.unknown(pc, op)
		}
	}
}

// unknown halts on an opcode the platform doesn't have, leaving PC on it.
func (m *Machine) unknown(pc, op uint16) {
	m.pc = pc
	m.halt(&OpcodeError{PC: pc, Op: op})
}
