package gopter

import (
	"math/rand"
	"time"
)

// GenParameters encapsulates the parameters for all generators.
type GenParameters struct {
	Size           int
	MaxShrinkCount int
	Rng            *rand.Rand
}

// WithSize modifies the size parameter. The size parameter defines an upper bound for the size of
// generated slices or strings.
func (p *GenParameters) WithSize(size int) *GenParameters {
	newParameters := *p
	newParameters.Size = size
	return &newParameters
}

// NextBool create a random boolean using the underlying Rng.
func (p *GenParameters) NextBool() bool {
	return p.Rng.Int63()&1 == 0
}

// NextInt64 create a random int64 using the underlying Rng.
func (p *GenParameters) NextInt64() int64 {
	v := p.Rng.Int63()
	if p.NextBool() {
		return -v
	}
	return v
}

// NextUint64 create a random uint64 using the underlying Rng.
func (p *GenParameters) NextUint64() uint64 {
	first := uint64(p.Rng.Int63())
	second := uint64(p.Rng.Int63())

	return (first << 1) ^ second
}

// DefaultGenParameters creates default GenParameters.
func DefaultGenParameters() *GenParameters {
	seed := time.Now().UnixNano()

	return &GenParameters{
		Size:           100,
		MaxShrinkCount: 1000,
		Rng:            rand.New(rand.NewSource(seed)),
	}
}
