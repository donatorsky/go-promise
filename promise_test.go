package promise

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPending(t *testing.T) {
	t.Run("Pending promise can be created", func(t *testing.T) {
		promise := Pending()

		require.Implements(t, (*Promiser)(nil), promise)
		require.Equal(t, StatePending, promise.state)
		require.Nil(t, promise.value)
		require.Nil(t, promise.err)
	})
}

func TestReject(t *testing.T) {
	t.Run("Rejected promise can be created", func(t *testing.T) {
		reason := errors.New("error reason")
		promise := Reject(reason)

		require.Implements(t, (*Promiser)(nil), promise)
		require.Equal(t, StateRejected, promise.state)
		require.Nil(t, promise.value)
		require.Same(t, reason, promise.err)
	})
}

func TestResolve(t *testing.T) {
	t.Run("Resolved promise can be created", func(t *testing.T) {
		value := 123
		promise := Resolve(value)

		require.Implements(t, (*Promiser)(nil), promise)
		require.Equal(t, StateFulfilled, promise.state)
		require.Equal(t, value, promise.value)
		require.Nil(t, promise.err)
	})
}
