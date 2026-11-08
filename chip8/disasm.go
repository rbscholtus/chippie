package chip8

import (
	"fmt"
	"strings"
)

// Disassemble returns the mnemonic for an opcode on a platform, in the
// common Cowgod style (e.g. "LD V1, 0x2A"). Opcodes the platform doesn't
// have are shown as "DATA 0x1234". It is meant for viewing traces, not for
// the hot path.
func Disassemble(p Platform, op uint16) string { //nolint:funlen // one case per opcode group
	ops := p.info().ops
	x := op >> 8 & 0xF
	y := op >> 4 & 0xF
	n := op & 0xF
	nn := op & 0xFF
	nnn := op & 0xFFF

	switch op >> 12 {
	case 0x0:
		if s, ok := disasm0(op, ops); ok {
			return s
		}
	case 0x1:
		return fmt.Sprintf("JP 0x%03X", nnn)
	case 0x2:
		return fmt.Sprintf("CALL 0x%03X", nnn)
	case 0x3:
		return fmt.Sprintf("SE V%X, 0x%02X", x, nn)
	case 0x4:
		return fmt.Sprintf("SNE V%X, 0x%02X", x, nn)
	case 0x5:
		if s, ok := disasm5(x, y, n, ops); ok {
			return s
		}
	case 0x6:
		return fmt.Sprintf("LD V%X, 0x%02X", x, nn)
	case 0x7:
		return fmt.Sprintf("ADD V%X, 0x%02X", x, nn)
	case 0x8:
		if s, ok := disasm8(x, y, n); ok {
			return s
		}
	case 0x9:
		if n == 0 {
			return fmt.Sprintf("SNE V%X, V%X", x, y)
		}
	case 0xA:
		return fmt.Sprintf("LD I, 0x%03X", nnn)
	case 0xB:
		return fmt.Sprintf("JP V0, 0x%03X", nnn)
	case 0xC:
		return fmt.Sprintf("RND V%X, 0x%02X", x, nn)
	case 0xD:
		return fmt.Sprintf("DRW V%X, V%X, %d", x, y, n) //nolint:dupword // format verbs, not words
	case 0xE:
		switch nn {
		case 0x9E:
			return fmt.Sprintf("SKP V%X", x)
		case 0xA1:
			return fmt.Sprintf("SKNP V%X", x)
		}
	case 0xF:
		if s, ok := disasmF(x, nn, ops); ok {
			return s
		}
	}

	return fmt.Sprintf("DATA 0x%04X", op)
}

// disasm5 names the 5XYN instructions the platform has.
func disasm5(x, y, n uint16, ops opSet) (string, bool) {
	switch {
	case n == 0:
		return fmt.Sprintf("SE V%X, V%X", x, y), true
	case ops&opsXO != 0 && n == 2:
		return fmt.Sprintf("SAVE V%X-V%X", x, y), true
	case ops&opsXO != 0 && n == 3:
		return fmt.Sprintf("LOAD V%X-V%X", x, y), true
	}

	return "", false
}

// disasm0 names the 00NN instructions the platform has.
func disasm0(op uint16, ops opSet) (string, bool) {
	switch {
	case op == 0x00E0:
		return "CLS", true
	case op == 0x00EE:
		return "RET", true
	case ops&opsXO != 0 && op&0xFFF0 == 0x00D0:
		return fmt.Sprintf("SCU %d", op&0xF), true
	case ops&opsScroll != 0 && op&0xFFF0 == 0x00C0:
		return fmt.Sprintf("SCD %d", op&0xF), true
	case ops&opsScroll != 0 && op == 0x00FB:
		return "SCR", true
	case ops&opsScroll != 0 && op == 0x00FC:
		return "SCL", true
	case ops&opsSuper != 0 && op == 0x00FD:
		return "EXIT", true
	case ops&opsSuper != 0 && op == 0x00FE:
		return "LOW", true
	case ops&opsSuper != 0 && op == 0x00FF:
		return "HIGH", true
	}

	return "", false
}

var mnemonics8 = map[uint16]string{ //nolint:gochecknoglobals // read-only lookup table
	0x0: "LD", 0x1: "OR", 0x2: "AND", 0x3: "XOR", 0x4: "ADD",
	0x5: "SUB", 0x6: "SHR", 0x7: "SUBN", 0xE: "SHL",
}

func disasm8(x, y, n uint16) (string, bool) {
	name, ok := mnemonics8[n]
	if !ok {
		return "", false
	}

	return fmt.Sprintf("%s V%X, V%X", name, x, y), true
}

var formatsF = map[uint16]string{ //nolint:gochecknoglobals // read-only lookup table
	0x07: "LD V%X, DT", 0x0A: "LD V%X, K", 0x15: "LD DT, V%X", 0x18: "LD ST, V%X",
	0x1E: "ADD I, V%X", 0x29: "LD F, V%X", 0x33: "LD B, V%X",
	0x55: "LD [I], V%X", 0x65: "LD V%X, [I]",
}

// formatsSuperF are SUPER-CHIP's additions to the FXNN group.
var formatsSuperF = map[uint16]string{ //nolint:gochecknoglobals // read-only lookup table
	0x30: "LD HF, V%X", 0x75: "LD R, V%X", 0x85: "LD V%X, R",
}

func disasmF(x, nn uint16, ops opSet) (string, bool) {
	if ops&opsXO != 0 {
		switch {
		case x == 0 && nn == 0x00:
			return "LD I, long", true // the operand is the next word
		case nn == 0x01:
			return fmt.Sprintf("PLANE %d", x), true
		case x == 0 && nn == 0x02:
			return "AUDIO", true
		case nn == 0x3A:
			return fmt.Sprintf("PITCH V%X", x), true
		}
	}

	format, ok := formatsF[nn]
	if !ok && ops&opsSuper != 0 {
		format, ok = formatsSuperF[nn]
	}

	if !ok {
		return "", false
	}

	return fmt.Sprintf(format, x), true
}

// FormatTrace returns trace entries as text, one instruction per line:
// frame, address, opcode, I and the disassembly. The UI uses it to show a
// trace, or to dump it when the machine halts on an unknown opcode.
func FormatTrace(p Platform, entries []TraceEntry) string {
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "%6d  %03X  %04X  I=%03X  %s\n", e.Frame, e.PC, e.Op, e.I, Disassemble(p, e.Op))
	}

	return b.String()
}
