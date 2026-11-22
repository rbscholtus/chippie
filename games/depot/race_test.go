//go:build race

package depot_test

// raceDetector is true in a -race build, which runs the emulator about 15
// times slower.
const raceDetector = true
