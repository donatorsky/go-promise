package promise

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jaswdr/faker"
	"github.com/stretchr/testify/assert"
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
	fakerInstance := faker.New()

	t.Run("Resolved promise can be created", func(t *testing.T) {
		value := fakerInstance.Int()
		promise := Resolve(value)

		require.Implements(t, (*Promiser)(nil), promise)
		require.Equal(t, StateFulfilled, promise.state)
		require.Equal(t, value, promise.value)
		require.Nil(t, promise.err)
	})
}

/**
 * @depends TestPending
 */
func TestPromise_Resolve(t *testing.T) {
	fakerInstance := faker.New()

	for _, tt := range []struct {
		state State
	}{
		{state: StateSettling},
		{state: StateFulfilled},
		{state: StateRejected},
	} {
		t.Run(fmt.Sprintf("Cannot manually Resolve promise in state: %s", tt.state), func(t *testing.T) {
			promise := Promise{
				state: tt.state,
			}

			require.ErrorIs(t, promise.Resolve(fakerInstance.Int()), ErrResolveNotPendingPromise)
		})
	}

	t.Run(fmt.Sprintf("Successfully manually Resolve promise in state: %s", StatePending), func(t *testing.T) {
		callsStack := NewCallsRegistry(1)

		var resolutionValue = fakerInstance.Int()

		promise := Promise{
			state: StatePending,
			handlers: []func(){
				func() {
					callsStack.Register("Fulfilled")
				},
			},
		}

		require.Nil(t, promise.Resolve(resolutionValue))
		callsStack.AssertCompletedInOrderBefore(t, []string{"Fulfilled"}, time.Millisecond*100)
		require.True(t, assertPromise(t, &promise, StateFulfilled, resolutionValue, nil))
	})
}

/**
 * @depends TestPending
 */
func TestPromise_Reject(t *testing.T) {
	fakerInstance := faker.New()

	for _, tt := range []struct {
		state State
	}{
		{state: StateSettling},
		{state: StateFulfilled},
		{state: StateRejected},
	} {
		t.Run(fmt.Sprintf("Cannot manually Reject promise in state: %s", tt.state), func(t *testing.T) {
			promise := Promise{
				state: tt.state,
			}

			require.ErrorIs(t, promise.Reject(errors.New("some error")), ErrRejectNotPendingPromise)
		})
	}

	t.Run(fmt.Sprintf("Successfully manually Reject promise in state: %s", StatePending), func(t *testing.T) {
		callsStack := NewCallsRegistry(1)

		var rejectionReason = errors.New(fakerInstance.Lorem().Sentence(6))

		promise := Promise{
			state: StatePending,
			handlers: []func(){
				func() {
					callsStack.Register("Rejected")
				},
			},
		}

		require.Nil(t, promise.Reject(rejectionReason))
		callsStack.AssertCompletedInOrderBefore(t, []string{"Rejected"}, time.Millisecond*100)
		require.True(t, assertPromise(t, &promise, StateRejected, nil, rejectionReason))
	})
}

func TestPromise_Then(t *testing.T) {
	fakerInstance := faker.New()

	for _, tt := range []struct {
		state State
	}{
		{state: StatePending},
		{state: StateSettling},
	} {
		t.Run(fmt.Sprintf("Returns new Promise and registers handler for Promise in state: %s", tt.state), func(t *testing.T) {
			callsStack := NewCallsRegistry(0)

			promise := Promise{
				state: tt.state,
			}

			thenPromise := promise.Then(func(value interface{}) (interface{}, error) {
				callsStack.Register("Then")

				return nil, nil
			})

			require.NotSame(t, &promise, thenPromise)
			require.Len(t, promise.handlers, 1)
			callsStack.AssertCompletedCallsStackIsEmpty(t)
		})
	}

	t.Run(fmt.Sprintf("Returns new Promise, does not register handler and executes Then immidiately for Promise in state: %s", StateFulfilled), func(t *testing.T) {
		callsStack := NewCallsRegistry(1)

		promise := Promise{
			state: StateFulfilled,
			value: fakerInstance.Int(),
		}

		thenPromise := promise.Then(func(value interface{}) (interface{}, error) {
			require.Equal(t, promise.value, value)

			callsStack.Register("Then")

			return nil, nil
		})

		require.NotSame(t, &promise, thenPromise)
		require.Empty(t, promise.handlers)
		callsStack.AssertCompletedInOrderBefore(t, []string{"Then"}, time.Millisecond*100)
	})

	t.Run(fmt.Sprintf("Returns new Promise, does not register handler and skips Then for Promise in state: %s", StateRejected), func(t *testing.T) {
		callsStack := NewCallsRegistry(0)

		promise := Promise{
			state: StateRejected,
		}

		thenPromise := promise.Then(func(value interface{}) (interface{}, error) {
			callsStack.Register("Then")

			return nil, nil
		})

		require.NotSame(t, &promise, thenPromise)
		require.Empty(t, promise.handlers)
		callsStack.AssertCompletedCallsStackIsEmpty(t)
	})
}

func TestPromise_Catch(t *testing.T) {
	fakerInstance := faker.New()

	for _, tt := range []struct {
		state State
	}{
		{state: StatePending},
		{state: StateSettling},
	} {
		t.Run(fmt.Sprintf("Returns new Promise and registers handler for Promise in state: %s", tt.state), func(t *testing.T) {
			callsStack := NewCallsRegistry(0)

			promise := Promise{
				state: tt.state,
			}

			catchPromise := promise.Catch(func(reason error) {
				callsStack.Register("Catch")
			})

			require.NotSame(t, &promise, catchPromise)
			require.Len(t, promise.handlers, 1)
			callsStack.AssertCompletedCallsStackIsEmpty(t)
		})
	}

	t.Run(fmt.Sprintf("Returns new Promise, does not register handler and skips Catch for Promise in state: %s", StateFulfilled), func(t *testing.T) {
		callsStack := NewCallsRegistry(0)

		promise := Promise{
			state: StateFulfilled,
		}

		catchPromise := promise.Catch(func(reason error) {
			callsStack.Register("Catch")
		})

		require.NotSame(t, &promise, catchPromise)
		require.Empty(t, promise.handlers)
		callsStack.AssertCompletedCallsStackIsEmpty(t)
	})

	t.Run(fmt.Sprintf("Returns new Promise, does not register handler and executes Catch immidiately for Promise in state: %s", StateRejected), func(t *testing.T) {
		callsStack := NewCallsRegistry(1)

		promise := Promise{
			state: StateRejected,
			err:   errors.New(fakerInstance.Lorem().Sentence(6)),
		}

		catchPromise := promise.Catch(func(reason error) {
			require.ErrorIs(t, promise.err, reason)

			callsStack.Register("Catch")
		})

		require.NotSame(t, &promise, catchPromise)
		require.Empty(t, promise.handlers)
		callsStack.AssertCompletedInOrderBefore(t, []string{"Catch"}, time.Millisecond*100)
	})
}

func TestPromise_Finally(t *testing.T) {
	for _, tt := range []struct {
		state State
	}{
		{state: StatePending},
		{state: StateSettling},
	} {
		t.Run(fmt.Sprintf("Returns new Promise and registers handler for Promise in state: %s", tt.state), func(t *testing.T) {
			callsStack := NewCallsRegistry(0)

			promise := Promise{
				state: tt.state,
			}

			finallyPromise := promise.Finally(func() {
				callsStack.Register("Finally")
			})

			require.NotSame(t, &promise, finallyPromise)
			require.Len(t, promise.handlers, 1)
			callsStack.AssertCompletedCallsStackIsEmpty(t)
		})
	}

	for _, tt := range []struct {
		state State
	}{
		{state: StateFulfilled},
		{state: StateRejected},
	} {
		t.Run(fmt.Sprintf("Returns new Promise, does not register handler and executes Finally immidiately for Promise in state: %s", tt.state), func(t *testing.T) {
			callsStack := NewCallsRegistry(1)

			promise := Promise{
				state: tt.state,
			}

			finallyPromise := promise.Finally(func() {
				callsStack.Register("Finally")
			})

			require.NotSame(t, &promise, finallyPromise)
			require.Empty(t, promise.handlers)
			callsStack.AssertCompletedInOrderBefore(t, []string{"Finally"}, time.Millisecond*100)
		})
	}
}

func TestNewPromise(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("Not resolved and not rejected Promise becomes pending", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(1)

		waitGroup.
			Initialize("root", 1).
			Initialize("NewPromise", 1)

		promise := NewPromise(func(_ Resolver, _ Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise")

			waitGroup.Done("NewPromise")
		})

		callsStack.AssertCurrentCallsStackIsEmpty(t)
		require.True(t, assertPromise(t, promise, StateSettling, nil, nil))

		waitGroup.Done("root")
		waitGroup.Wait("NewPromise")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise"}, time.Millisecond*100)
		time.Sleep(time.Millisecond * 50)
		require.True(t, assertPromise(t, promise, StatePending, nil, nil))
	})

	t.Run("Resolved and not rejected Promise is completed", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(1)

		var resolutionValue = fakerInstance.Int()

		waitGroup.
			Initialize("root", 1).
			Initialize("NewPromise", 1)

		promise := NewPromise(func(resolve Resolver, _ Rejector) {
			waitGroup.Wait("root")

			resolve(resolutionValue)

			callsStack.Register("NewPromise")

			waitGroup.Done("NewPromise")
		})

		callsStack.AssertCurrentCallsStackIsEmpty(t)
		require.True(t, assertPromise(t, promise, StateSettling, nil, nil))

		waitGroup.Done("root")
		waitGroup.Wait("NewPromise")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise"}, time.Millisecond*200)
		time.Sleep(time.Millisecond * 50)
		require.True(t, assertPromise(t, promise, StateFulfilled, resolutionValue, nil))
	})

	t.Run("Not resolved and rejected Promise is completed", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(1)

		var rejectionReason = errors.New(fakerInstance.Lorem().Sentence(6))

		waitGroup.Initialize("root", 1)

		promise := NewPromise(func(_ Resolver, reject Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise")

			reject(rejectionReason)
		})

		callsStack.AssertCurrentCallsStackIsEmpty(t)
		require.True(t, assertPromise(t, promise, StateSettling, nil, nil))

		waitGroup.Done("root")
		time.Sleep(time.Millisecond * 50)

		callsStack.AssertCompletedInOrder(t, []string{"NewPromise"})
		require.True(t, assertPromise(t, promise, StateRejected, nil, rejectionReason))
	})

	t.Run("Resolved and rejected Promise is only resolved", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(1)

		var resolvedValue = fakerInstance.Int()
		var rejectionReason = errors.New(fakerInstance.Lorem().Sentence(6))

		waitGroup.
			Initialize("root", 1).
			Initialize("NewPromise", 1)

		promise := NewPromise(func(resolve Resolver, reject Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise")

			resolve(resolvedValue)
			reject(rejectionReason)

			waitGroup.Done("NewPromise")
		})

		callsStack.AssertCurrentCallsStackIsEmpty(t)
		require.True(t, assertPromise(t, promise, StateSettling, nil, nil))

		waitGroup.Done("root")
		waitGroup.Wait("NewPromise")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise"}, time.Millisecond*100)
		time.Sleep(time.Millisecond * 50)
		require.True(t, assertPromise(t, promise, StateFulfilled, resolvedValue, nil))
	})

	t.Run("Rejected and resolved Promise is only rejected", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(1)

		var resolvedValue = fakerInstance.Int()
		var rejectionReason = errors.New(fakerInstance.Lorem().Sentence(6))

		waitGroup.
			Initialize("root", 1).
			Initialize("NewPromise", 1)

		promise := NewPromise(func(resolve Resolver, reject Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise")

			reject(rejectionReason)
			resolve(resolvedValue)

			waitGroup.Done("NewPromise")
		})

		callsStack.AssertCurrentCallsStackIsEmpty(t)
		require.True(t, assertPromise(t, promise, StateSettling, nil, nil))

		waitGroup.Done("root")
		waitGroup.Wait("NewPromise")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise"}, time.Millisecond*100)
		time.Sleep(time.Millisecond * 50)
		require.True(t, assertPromise(t, promise, StateRejected, nil, rejectionReason))
	})
}

func TestPromise(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("Multiple Then callbacks receive the same resolution value, pass modified value", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(7)

		var resolvedValue = fakerInstance.IntBetween(2, 999)

		waitGroup.
			Initialize("root", 1).
			Initialize("level-2", 3)

		promise := NewPromise(func(resolve Resolver, _ Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise.1")

			resolve(resolvedValue)
		})

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				callsStack.Register("Then.1")

				return resolvedValue + 0, nil
			}).
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue+0, value)

				defer waitGroup.Done("level-2")

				time.Sleep(time.Millisecond * 150)

				callsStack.Register("Then.1.1")

				return nil, nil
			})

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				callsStack.Register("Then.2")

				return resolvedValue + 1, nil
			}).
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue+1, value)

				defer waitGroup.Done("level-2")

				time.Sleep(time.Millisecond * 100)

				callsStack.Register("Then.2.1")

				return nil, nil
			})

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				callsStack.Register("Then.3")

				return resolvedValue + 2, nil
			}).
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue+2, value)

				defer waitGroup.Done("level-2")

				time.Sleep(time.Millisecond * 50)

				callsStack.Register("Then.3.1")

				return nil, nil
			})

		waitGroup.Done("root")
		waitGroup.Wait("level-2")

		callsStack.AssertCompletedInOrder(t, []string{"NewPromise.1", "Then.1", "Then.2", "Then.3", "Then.1.1", "Then.2.1", "Then.3.1"})
	})

	t.Run("Multiple Then callbacks receive the same resolution value, pass modified value as Promise", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(7)

		var resolvedValue = fakerInstance.IntBetween(2, 999)

		waitGroup.
			Initialize("root", 1).
			Initialize("defer-then.1.1", 1).
			Initialize("defer-then.2.1", 1).
			Initialize("defer-then.3.1", 1)

		promise := NewPromise(func(resolve Resolver, _ Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise.1")

			resolve(resolvedValue)
		})

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				callsStack.Register("Then.1")

				return NewPromise(func(resolve Resolver, _ Rejector) {
					waitGroup.Wait("defer-then.1.1")

					resolve(resolvedValue + 0)
				}), nil
			}).
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue+0, value)

				callsStack.Register("Then.1.1")

				return nil, nil
			})

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				callsStack.Register("Then.2")

				return NewPromise(func(resolve Resolver, _ Rejector) {
					waitGroup.Wait("defer-then.2.1")

					resolve(resolvedValue + 1)
				}), nil
			}).
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue+1, value)

				callsStack.Register("Then.2.1")

				return nil, nil
			})

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				callsStack.Register("Then.3")

				return NewPromise(func(resolve Resolver, _ Rejector) {
					waitGroup.Wait("defer-then.3.1")

					resolve(resolvedValue + 2)
				}), nil
			}).
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue+2, value)

				callsStack.Register("Then.3.1")

				return nil, nil
			})

		waitGroup.Done("root")
		waitGroup.Done("defer-then.3.1")
		time.Sleep(time.Millisecond * 50)
		waitGroup.Done("defer-then.2.1")
		time.Sleep(time.Millisecond * 50)
		waitGroup.Done("defer-then.1.1")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Then.1", "Then.2", "Then.3", "Then.3.1", "Then.2.1", "Then.1.1"}, time.Millisecond*500)
	})

	t.Run("Multiple Catch callbacks receive the same rejection error, do not pass it further", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(4)

		var rejectionReason = fakerInstance.Lorem().Sentence(6)

		waitGroup.
			Initialize("root", 1).
			Initialize("catch", 3)

		promise := NewPromise(func(_ Resolver, reject Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise.1")

			reject(errors.New(rejectionReason))
		})

		promise.
			Catch(func(reason error) {
				require.EqualError(t, reason, rejectionReason)

				defer waitGroup.Done("catch")

				callsStack.Register("Catch.1")
			}).
			Catch(func(reason error) {
				callsStack.Register("Catch.1.1")
			})

		promise.
			Catch(func(reason error) {
				require.EqualError(t, reason, rejectionReason)

				defer waitGroup.Done("catch")

				callsStack.Register("Catch.2")
			}).
			Catch(func(reason error) {
				callsStack.Register("Catch.2.1")
			})

		promise.
			Catch(func(reason error) {
				require.EqualError(t, reason, rejectionReason)

				defer waitGroup.Done("catch")

				callsStack.Register("Catch.3")
			}).
			Catch(func(reason error) {
				callsStack.Register("Catch.3.1")
			})

		waitGroup.Done("root")
		waitGroup.Wait("catch")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Catch.1", "Catch.2", "Catch.3"}, time.Millisecond*100)
	})

	t.Run("Finally is called after Then", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(4)

		var resolvedValue = fakerInstance.Int()

		waitGroup.
			Initialize("root", 1).
			Initialize("level-1", 2).
			Initialize("level-2", 1)

		promise := NewPromise(func(resolve Resolver, _ Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise.1")

			resolve(resolvedValue)
		})

		promise.Catch(func(_ error) {
			callsStack.Register("Catch.1")
		})

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				defer waitGroup.Done("level-1")

				callsStack.Register("Then.1")

				return nil, nil
			}).
			Finally(func() {
				defer waitGroup.Done("level-2")

				callsStack.Register("Finally.2")
			})

		promise.Finally(func() {
			defer waitGroup.Done("level-1")

			callsStack.Register("Finally.1")
		})

		waitGroup.Done("root")
		waitGroup.Wait("level-1")
		waitGroup.Wait("level-2")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Then.1", "Finally.1", "Finally.2"}, time.Millisecond*100)
	})

	t.Run("Finally is called after Catch", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(4)

		var rejectionReason = fakerInstance.Lorem().Sentence(6)

		waitGroup.
			Initialize("root", 1).
			Initialize("level-1", 2)

		promise := NewPromise(func(_ Resolver, reject Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise.1")

			reject(errors.New(rejectionReason))
		})

		promise.Then(func(_ interface{}) (interface{}, error) {
			callsStack.Register("Then.1")

			return nil, nil
		})

		promise.
			Catch(func(reason error) {
				require.EqualError(t, reason, rejectionReason)

				defer waitGroup.Done("level-1")

				callsStack.Register("Catch.1")
			}).
			Finally(func() {
				callsStack.Register("Finally.2")
			})

		promise.Finally(func() {
			defer waitGroup.Done("level-1")

			callsStack.Register("Finally.1")
		})

		waitGroup.Done("root")
		waitGroup.Wait("level-1")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Catch.1", "Finally.1", "Finally.2"}, time.Millisecond*100)
	})

	t.Run("Then can return another Promise", func(t *testing.T) {
		t.Run("Already resolved Promise", func(t *testing.T) {
			waitGroup := NewWaitGroup()
			callsStack := NewCallsRegistry(5)

			var resolvedValue = fakerInstance.Int()

			waitGroup.Initialize("root", 1)

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				waitGroup.Wait("root")

				callsStack.Register("NewPromise.1")

				resolve(resolvedValue)
			})

			promise.Catch(func(_ error) {
				callsStack.Register("Catch.1")
			})

			var resolvedThenValue = fakerInstance.Lorem().Word()

			promise.
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedValue, value)

					callsStack.Register("Then.1")

					return Resolve(resolvedThenValue), nil
				}).
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedThenValue, value)

					callsStack.Register("Then.2")

					return nil, nil
				}).
				Finally(func() {
					callsStack.Register("Finally.3")
				})

			promise.Finally(func() {
				callsStack.Register("Finally.1")
			})

			waitGroup.Done("root")

			callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Then.1", "Finally.1", "Then.2", "Finally.3"}, time.Second)
		})

		t.Run("Already rejected Promise", func(t *testing.T) {
			waitGroup := NewWaitGroup()
			callsStack := NewCallsRegistry(5)

			var resolvedValue = fakerInstance.Int()

			waitGroup.
				Initialize("root", 1).
				Initialize("finally.3", 1)

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				waitGroup.Wait("root")

				callsStack.Register("NewPromise.1")

				resolve(resolvedValue)
			})

			promise.Catch(func(_ error) {
				callsStack.Register("Catch.1")
			})

			var rejectedThenValue = fakerInstance.Lorem().Sentence(6)

			promise.
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedValue, value)

					callsStack.Register("Then.1")

					return Reject(errors.New(rejectedThenValue)), nil
				}).
				Catch(func(reason error) {
					require.EqualError(t, reason, rejectedThenValue)

					callsStack.Register("Catch.2")
				}).
				Finally(func() {
					defer waitGroup.Done("finally.3")

					callsStack.Register("Finally.3")
				})

			promise.Finally(func() {
				callsStack.Register("Finally.1")
			})

			waitGroup.Done("root")
			waitGroup.Wait("finally.3")

			callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Then.1", "Finally.1", "Catch.2", "Finally.3"}, time.Second)
		})

		t.Run("Settling Promise", func(t *testing.T) {
			waitGroup := NewWaitGroup()
			callsStack := NewCallsRegistry(6)

			var resolvedValue = fakerInstance.Int()

			waitGroup.Initialize("root", 1)

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				waitGroup.Wait("root")

				callsStack.Register("NewPromise.1")

				resolve(resolvedValue)
			})

			promise.Catch(func(_ error) {
				callsStack.Register("Catch.1")
			})

			var resolvedThenValue = fakerInstance.Lorem().Sentence(6)

			promise.
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedValue, value)

					callsStack.Register("Then.1")

					return NewPromise(func(resolve Resolver, _ Rejector) {
						callsStack.Register("NewPromise.1.1")

						resolve(resolvedThenValue)
					}), nil
				}).
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedThenValue, value)

					callsStack.Register("Then.2")

					return nil, nil
				}).
				Finally(func() {
					callsStack.Register("Finally.3")
				})

			promise.Finally(func() {
				callsStack.Register("Finally.1")
			})

			waitGroup.Done("root")

			callsStack.AssertCompletedInOrderBefore(
				t,
				[]string{"NewPromise.1", "Then.1", "Finally.1", "NewPromise.1.1", "Then.2", "Finally.3"},
				time.Millisecond*200,
			)
		})

		t.Run("Resolve pending Promise", func(t *testing.T) {
			waitGroup := NewWaitGroup()
			callsStack := NewCallsRegistry(5)

			var resolvedValue = fakerInstance.Int()

			waitGroup.
				Initialize("root", 1).
				Initialize("level-1", 2)

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				waitGroup.Wait("root")

				callsStack.Register("NewPromise.1")

				resolve(resolvedValue)
			})

			promise.Catch(func(_ error) {
				callsStack.Register("Catch.1")
			})

			var resolvedThenValue = fakerInstance.Lorem().Sentence(6)
			pendingPromise := Pending()

			promise.
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedValue, value)

					defer waitGroup.Done("level-1")

					callsStack.Register("Then.1")

					return pendingPromise, nil
				}).
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedThenValue, value)

					callsStack.Register("Then.2")

					return nil, nil
				}).
				Finally(func() {
					callsStack.Register("Finally.3")
				})

			promise.Finally(func() {
				defer waitGroup.Done("level-1")

				callsStack.Register("Finally.1")
			})

			// Resume NewPromise, let it be resolved and call Then
			waitGroup.Done("root")
			waitGroup.Wait("level-1")

			callsStack.AssertCurrentCallsStackIs(t, []string{"NewPromise.1", "Then.1", "Finally.1"})
			callsStack.AssertThereAreNCallsLeft(t, 2)

			// Manually resolve pending promise
			require.Nil(t, pendingPromise.Resolve(resolvedThenValue))

			// Wait for next Then and Finally to be called
			callsStack.AssertCompletedInOrderBefore(
				t,
				[]string{"NewPromise.1", "Then.1", "Finally.1", "Then.2", "Finally.3"},
				time.Millisecond*100,
			)
		})

		t.Run("Reject pending Promise", func(t *testing.T) {
			waitGroup := NewWaitGroup()
			callsStack := NewCallsRegistry(5)

			var resolvedValue = fakerInstance.Int()

			waitGroup.
				Initialize("root", 1).
				Initialize("level-1", 2)

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				waitGroup.Wait("root")

				callsStack.Register("NewPromise.1")

				resolve(resolvedValue)
			})

			promise.Catch(func(_ error) {
				callsStack.Register("Catch.1")
			})

			var resolvedThenValue = fakerInstance.Lorem().Sentence(6)
			pendingPromise := Pending()

			promise.
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedValue, value)

					defer waitGroup.Done("level-1")

					callsStack.Register("Then.1")

					return pendingPromise, nil
				}).
				Catch(func(reason error) {
					require.EqualError(t, reason, resolvedThenValue)

					callsStack.Register("Catch.2")
				}).
				Finally(func() {
					callsStack.Register("Finally.3")
				})

			promise.Finally(func() {
				defer waitGroup.Done("level-1")

				callsStack.Register("Finally.1")
			})

			// Resume NewPromise, let it be resolved and call Then
			waitGroup.Done("root")
			waitGroup.Wait("level-1")

			callsStack.AssertCurrentCallsStackIs(t, []string{"NewPromise.1", "Then.1", "Finally.1"})
			callsStack.AssertThereAreNCallsLeft(t, 2)

			// Manually resolve pending promise
			require.Nil(t, pendingPromise.Reject(errors.New(resolvedThenValue)))

			// Wait for next Then and Finally to be called
			callsStack.AssertCompletedInOrderBefore(
				t,
				[]string{"NewPromise.1", "Then.1", "Finally.1", "Catch.2", "Finally.3"},
				time.Millisecond*100,
			)
		})
	})

	t.Run("Then after Catch", func(t *testing.T) {
		t.Run("Then receives resolved value when Catch was not called", func(t *testing.T) {
			waitGroup := NewWaitGroup()
			callsStack := NewCallsRegistry(3)

			var resolvedValue = fakerInstance.Int()

			waitGroup.Initialize("root", 1)

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				waitGroup.Wait("root")

				callsStack.Register("NewPromise.1")

				resolve(resolvedValue)
			})

			promise.
				Catch(func(_ error) {
					callsStack.Register("Catch.1")
				}).
				Then(func(value interface{}) (interface{}, error) {
					require.Equal(t, resolvedValue, value)

					callsStack.Register("Then.2")

					return nil, nil
				}).
				Finally(func() {
					callsStack.Register("Finally.3")
				})

			waitGroup.Done("root")

			callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Then.2", "Finally.3"}, time.Millisecond*100)
		})

		t.Run("Then receives nil value when Catch was called", func(t *testing.T) {
			waitGroup := NewWaitGroup()
			callsStack := NewCallsRegistry(4)

			var rejectionValue = fakerInstance.Lorem().Sentence(6)

			waitGroup.Initialize("root", 1)

			promise := NewPromise(func(_ Resolver, reject Rejector) {
				waitGroup.Wait("root")

				callsStack.Register("NewPromise.1")

				reject(errors.New(rejectionValue))
			})

			promise.
				Catch(func(reason error) {
					require.EqualError(t, reason, rejectionValue)

					callsStack.Register("Catch.1")
				}).
				Then(func(value interface{}) (interface{}, error) {
					require.Nil(t, value)

					callsStack.Register("Then.2")

					return nil, nil
				}).
				Finally(func() {
					callsStack.Register("Finally.3")
				})

			waitGroup.Done("root")

			callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Catch.1", "Then.2", "Finally.3"}, time.Millisecond*100)
		})
	})

	t.Run("Then returns error and Catch receives it", func(t *testing.T) {
		waitGroup := NewWaitGroup()
		callsStack := NewCallsRegistry(6)

		var resolvedValue = fakerInstance.Int()

		waitGroup.Initialize("root", 1)

		promise := NewPromise(func(resolve Resolver, _ Rejector) {
			waitGroup.Wait("root")

			callsStack.Register("NewPromise.1")

			resolve(resolvedValue)
		})

		promise.Catch(func(_ error) {
			callsStack.Register("Catch.1")
		}).Finally(func() {
			callsStack.Register("Finally.1.1")
		})

		promise.Finally(func() {
			callsStack.Register("Finally.2")
		})

		var rejectionReason = fakerInstance.Lorem().Sentence(6)

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				callsStack.Register("Then.3")

				return nil, errors.New(rejectionReason)
			}).
			Catch(func(reason error) {
				require.EqualError(t, reason, rejectionReason)

				callsStack.Register("Catch.3.1")
			}).
			Finally(func() {
				callsStack.Register("Finally.3.1.1")
			})

		waitGroup.Done("root")

		callsStack.AssertCompletedInOrderBefore(t, []string{"NewPromise.1", "Finally.2", "Then.3", "Finally.1.1", "Catch.3.1", "Finally.3.1.1"}, time.Millisecond*100)
	})
}

func assertPromise(t *testing.T, promise *Promise, state State, value interface{}, reason error) bool {
	isSuccessful := assert.Equal(t, state, promise.state)

	if nil == value {
		isSuccessful = isSuccessful && assert.Nil(t, promise.value)
	} else {
		isSuccessful = isSuccessful && assert.Equal(t, value, promise.value)
	}

	if nil == reason {
		isSuccessful = isSuccessful && assert.Nil(t, promise.err)
	} else {
		isSuccessful = isSuccessful && assert.Equal(t, reason, promise.err)
	}

	return isSuccessful
}
