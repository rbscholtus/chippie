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
	_ = p // the platform selects opcode sets once SUPER-CHIP and XO-CHIP are added

	x := op >> 8 & 0xF
	y := op >> 4 & 0xF
	n := op & 0xF
	nn := op & 0xFF
	nnn := op & 0xFFF

	switch op >> 12 {
	case 0x0:
		switch op {
		case 0x00E0:
			return "CLS"
		case 0x00EE:
			return "RET"
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
		if n == 0 {
			return fmt.Sprintf("SE V%X, V%X", x, y)
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
		if s, ok := disasmF(x, nn); ok {
			return s
		}
	}

	return fmt.Sprintf("DATA 0x%04X", op)
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

func disasmF(x, nn uint16) (string, bool) {
	format, ok := formatsF[nn]
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
