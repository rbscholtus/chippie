package chip8

// Update reports the -update flag (defined in trace_test.go) to the
// external chip8_test package, which shares this test binary.
func Update() bool { return *update }
