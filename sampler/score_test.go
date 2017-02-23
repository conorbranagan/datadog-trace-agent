package sampler

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func randomTraceID() uint64 {
	return uint64(rand.Int63())
}

func TestTrivialSampleByRate(t *testing.T) {
	assert := assert.New(t)

	assert.False(SampleByRate(randomTraceID(), 0))
	assert.True(SampleByRate(randomTraceID(), 1))
}

func TestSampleRateManyTraces(t *testing.T) {
	// Test that the effective sample rate isn't far from the theoretical
	// Test with multiple sample rates
	assert := assert.New(t)

	times := 1e6

	for _, rate := range []float64{0.0, 1, 0.1, 0.5, 0.99} {
		sampled := 0
		for i := 0; i < int(times); i++ {
			if SampleByRate(randomTraceID(), rate) {
				sampled++
			}
		}
		assert.InEpsilon(sampled, times*rate, 0.01)
	}
}
