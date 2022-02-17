package promise

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func NewCallsRegistry(expectedCalls uint) *callsRegistry {
	registry := callsRegistry{
		expectedCalls: expectedCalls,
	}

	return &registry
}

type callsRegistry struct {
	mutex sync.RWMutex

	registry      []string
	expectedCalls uint
}

func (r *callsRegistry) Register(place string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if 0 == r.expectedCalls {
		panic("trying to register unexpected call: " + place)
	}

	r.registry = append(r.registry, place)
	r.expectedCalls--
}

func (r *callsRegistry) Summarize() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	return strings.Join(r.registry, "|")
}

func (r *callsRegistry) AssertCompletedBefore(t *testing.T, expectedRegistry string, timeLimit time.Duration) {
	timeLimiter := time.After(timeLimit)

	for {
		select {
		case <-timeLimiter:
			require.FailNowf(
				t,
				"Calls registry assertion timeout",
				"There are still %d expected call(s) left. Calls registered: %v.",
				r.expectedCalls,
				r.registry,
			)
			return

		default:
			r.mutex.RLock()
			waitsForCalls := 0 != r.expectedCalls
			r.mutex.RUnlock()

			if waitsForCalls {
				continue
			}

			require.Equal(t, expectedRegistry, r.Summarize())
			return
		}
	}
}

func (r *callsRegistry) AssertCurrentCallsStackIs(t *testing.T, expectedRegistry string) {
	require.Equal(t, expectedRegistry, r.Summarize())
}

func (r *callsRegistry) AssertThereAreNCallsLeft(t *testing.T, callsLeftNumber uint) {
	require.Equal(t, callsLeftNumber, r.expectedCalls)
}
