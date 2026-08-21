package lottery

import "errors"

var ErrInvalidDraw = errors.New("invalid lottery draw")

// Draw returns a deterministic sample without replacement. The same candidates,
// winnerCount, and seed must always produce the same result.
func Draw(candidates []string, winnerCount int, seed int64) ([]string, error) {
	// TODO(level-11): validate, copy input, shuffle with a local seeded RNG, slice.
	return nil, ErrInvalidDraw
}
