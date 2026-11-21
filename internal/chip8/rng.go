package chip8

// rng is a xorshift64* generator for CXNN. It is seeded from the profile, so
// runs are repeatable; the core never uses global random state.
type rng struct{ state uint64 }

// defaultSeed replaces a zero seed, because xorshift never leaves state 0.
const defaultSeed = 0x9E3779B97F4A7C15

// xorshiftMultiplier is the xorshift64* output multiplier (Vigna, 2016).
const xorshiftMultiplier = 0x2545F4914F6CDD1D

func (r *rng) seed(s uint64) {
	if s == 0 {
		s = defaultSeed
	}

	r.state = s
}

// byte returns the next random byte.
func (r *rng) byte() byte {
	x := r.state
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	r.state = x

	return byte((x * xorshiftMultiplier) >> 56)
}
