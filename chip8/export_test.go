package chip8

// Update reports the -update flag (defined in trace_test.go) to the
// external chip8_test package, which shares this test binary.
func Update() bool { return *update }

// TwoPageROM builds a 2 Page Display ROM around prog: 1260 and the
// article's patch, then either padding up to 0x260 (the article's layout)
// or, if selfPatching, the self-patching code of the ROMs in circulation
// up to 0x2C0. prog follows at 0x260 or 0x2C0.
func TwoPageROM(selfPatching bool, prog []byte) []byte {
	rom := append([]byte{twoPageStartOp >> 8, twoPageStartOp & 0xFF}, twoPagePatch...)

	if selfPatching {
		rom = append(rom, twoPageSelfPatch...)
		rom = append(rom, "REVIVAL2011"...)
		rom = append(rom, twoPageClearOp>>8, twoPageClearOp&0xFF)
	}

	for len(rom) < int(twoPageStart(selfPatching)-ProgramStart) {
		rom = append(rom, 0)
	}

	return append(rom, prog...)
}

// twoPageStart is where TwoPageROM puts the program.
func twoPageStart(selfPatching bool) uint16 {
	if selfPatching {
		return twoPageSelfPatchedProgram
	}

	return twoPageProgram
}
