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
			promise := &Promise{
				state: tt.state,
			}

			require.ErrorIs(t, promise.Resolve(fakerInstance.Int()), ErrResolveNotPendingPromise)
		})
	}

	t.Run(fmt.Sprintf("Successfully manually Resolve promise in state: %s", StatePending), func(t *testing.T) {
		var resolutionValue = fakerInstance.Int()

		onFulfilledCallsCounter := 0
		observerPromise := Promise{
			onFulfilled: func(value interface{}) (interface{}, error) {
				if assert.Equal(t, resolutionValue, value) {
					onFulfilledCallsCounter++
				}

				return nil, nil
			},
		}

		promise := Pending()
		promise.observers = append(promise.observers, &observerPromise)

		require.Nil(t, promise.Resolve(resolutionValue))
		observerPromise.wg.Wait()
		require.Equal(t, 1, onFulfilledCallsCounter)
		require.True(t, assertPromise(t, promise, StateFulfilled, resolutionValue, nil))
	})
}

/**
 * @depends TestPending
 */
func TestPromise_Reject(t *testing.T) {
	for _, tt := range []struct {
		state State
	}{
		{state: StateSettling},
		{state: StateFulfilled},
		{state: StateRejected},
	} {
		t.Run(fmt.Sprintf("Cannot manually Reject promise in state: %s", tt.state), func(t *testing.T) {
			promise := &Promise{
				state: tt.state,
			}

			require.ErrorIs(t, promise.Reject(errors.New("some error")), ErrRejectNotPendingPromise)
		})
	}

	t.Run(fmt.Sprintf("Successfully manually Reject promise in state: %s", StatePending), func(t *testing.T) {
		var rejectionReason = errors.New("some rejection error")

		onRejectedCallsCounter := 0
		observerPromise := Promise{
			onRejected: func(reason error) {
				if assert.Same(t, rejectionReason, reason) {
					onRejectedCallsCounter++
				}
			},
		}

		promise := Pending()
		promise.observers = append(promise.observers, &observerPromise)

		require.Nil(t, promise.Reject(rejectionReason))
		observerPromise.wg.Wait()
		require.Equal(t, 1, onRejectedCallsCounter)
		require.True(t, assertPromise(t, promise, StateRejected, nil, rejectionReason))
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
		t.Run(fmt.Sprintf("Returns new Promise and registers as observer for Promise in state: %s", tt.state), func(t *testing.T) {
			promise := Promise{
				state: tt.state,
			}

			require.Empty(t, promise.observers)

			thenPromiseHasBeenCalledTimes := 0
			thenPromise := promise.Then(func(value interface{}) (interface{}, error) {
				thenPromiseHasBeenCalledTimes++

				return nil, nil
			})

			require.NotSame(t, promise, thenPromise)
			require.Len(t, promise.observers, 1)
			require.Same(t, thenPromise, promise.observers[0])

			promise.wg.Wait()
			thenPromise.(*Promise).wg.Wait()
			require.Equal(t, 0, thenPromiseHasBeenCalledTimes)
		})
	}

	t.Run(fmt.Sprintf("Returns new Promise, does not register as observer and executes Then immidiately for Promise in state: %s", StateFulfilled), func(t *testing.T) {
		promise := Promise{
			state: StateFulfilled,
			value: fakerInstance.Int(),
		}

		require.Empty(t, promise.observers)

		thenPromiseHasBeenCalledTimes := 0
		thenPromise := promise.Then(func(value interface{}) (interface{}, error) {
			if promise.value == value {
				thenPromiseHasBeenCalledTimes++
			}

			return nil, nil
		})

		require.NotSame(t, promise, thenPromise)
		require.Empty(t, promise.observers)

		promise.wg.Wait()
		thenPromise.(*Promise).wg.Wait()
		require.Equal(t, 1, thenPromiseHasBeenCalledTimes)
	})

	t.Run(fmt.Sprintf("Returns new Promise, does not register as observer and skips Then for Promise in state: %s", StateRejected), func(t *testing.T) {
		promise := Promise{
			state: StateRejected,
		}

		require.Empty(t, promise.observers)

		thenPromiseHasBeenCalledTimes := 0
		thenPromise := promise.Then(func(value interface{}) (interface{}, error) {
			thenPromiseHasBeenCalledTimes++

			return nil, nil
		})

		require.NotSame(t, promise, thenPromise)
		require.Empty(t, promise.observers)

		promise.wg.Wait()
		thenPromise.(*Promise).wg.Wait()
		require.Equal(t, 0, thenPromiseHasBeenCalledTimes)
	})
}

func TestPromise_Catch(t *testing.T) {
	for _, tt := range []struct {
		state State
	}{
		{state: StatePending},
		{state: StateSettling},
	} {
		t.Run(fmt.Sprintf("Returns new Promise and registers as observer for Promise in state: %s", tt.state), func(t *testing.T) {
			promise := Promise{
				state: tt.state,
			}

			require.Empty(t, promise.observers)

			catchPromiseHasBeenCalledTimes := 0
			catchPromise := promise.Catch(func(reason error) {
				catchPromiseHasBeenCalledTimes++
			})

			require.NotSame(t, promise, catchPromise)
			require.Len(t, promise.observers, 1)
			require.Same(t, catchPromise, promise.observers[0])
			require.Equal(t, 0, catchPromiseHasBeenCalledTimes)
		})
	}

	t.Run(fmt.Sprintf("Returns new Promise, does not register as observer and skips Catch for Promise in state: %s", StateFulfilled), func(t *testing.T) {
		promise := Promise{
			state: StateFulfilled,
		}

		require.Empty(t, promise.observers)

		catchPromiseHasBeenCalledTimes := 0
		catchPromise := promise.Catch(func(reason error) {
			catchPromiseHasBeenCalledTimes++
		})

		require.NotSame(t, promise, catchPromise)
		require.Empty(t, promise.observers)

		promise.wg.Wait()
		catchPromise.(*Promise).wg.Wait()
		require.Equal(t, 0, catchPromiseHasBeenCalledTimes)
	})

	t.Run(fmt.Sprintf("Returns new Promise, does not register as observer and executes Catch immidiately for Promise in state: %s", StateRejected), func(t *testing.T) {
		promise := Promise{
			state: StateRejected,
			err:   errors.New("rejection reason"),
		}

		require.Empty(t, promise.observers)

		catchPromiseHasBeenCalledTimes := 0
		catchPromise := promise.Catch(func(reason error) {
			if promise.err == reason {
				catchPromiseHasBeenCalledTimes++
			}
		})

		require.NotSame(t, promise, catchPromise)
		require.Empty(t, promise.observers)

		promise.wg.Wait()
		catchPromise.(*Promise).wg.Wait()
		require.Equal(t, 1, catchPromiseHasBeenCalledTimes)
	})
}

func TestPromise_Finally(t *testing.T) {
	for _, tt := range []struct {
		state State
	}{
		{state: StatePending},
		{state: StateSettling},
	} {
		t.Run(fmt.Sprintf("Returns new Promise and registers as observer for Promise in state: %s", tt.state), func(t *testing.T) {
			promise := Promise{
				state: tt.state,
			}

			require.Empty(t, promise.observers)

			finallyPromiseHasBeenCalledTimes := 0
			finallyPromise := promise.Finally(func() {
				finallyPromiseHasBeenCalledTimes++
			})

			require.NotSame(t, promise, finallyPromise)
			require.Len(t, promise.observers, 1)
			require.Same(t, finallyPromise, promise.observers[0])
			require.Equal(t, 0, finallyPromiseHasBeenCalledTimes)
		})
	}

	for _, tt := range []struct {
		state State
	}{
		{state: StateFulfilled},
		{state: StateRejected},
	} {
		t.Run(fmt.Sprintf("Returns new Promise, does not register as observer and executes Finally immidiately for Promise in state: %s", tt.state), func(t *testing.T) {
			promise := Promise{
				state: tt.state,
			}

			require.Empty(t, promise.observers)

			finallyPromiseHasBeenCalledTimes := 0
			finallyPromise := promise.Finally(func() {
				finallyPromiseHasBeenCalledTimes++
			})

			require.NotSame(t, promise, finallyPromise)
			require.Empty(t, promise.observers)
			require.Equal(t, 0, finallyPromiseHasBeenCalledTimes)
		})
	}
}

func TestNewPromise(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("Not resolved and not rejected Promise becomes pending", func(t *testing.T) {
		callbackCallsCounter := 0
		promise := NewPromise(func(_ Resolver, _ Rejector) {
			time.Sleep(time.Millisecond * 5)

			callbackCallsCounter++
		})

		require.Equal(t, 0, callbackCallsCounter)
		require.True(t, assertPromise(t, promise, StateSettling, nil, nil))
		promise.wg.Wait()
		require.Equal(t, 1, callbackCallsCounter)
		require.True(t, assertPromise(t, promise, StatePending, nil, nil))
	})

	t.Run("Resolved and not rejected Promise is completed", func(t *testing.T) {
		resolutionValue := fakerInstance.Int()
		callbackCallsCounter := 0
		promise := NewPromise(func(resolve Resolver, _ Rejector) {
			time.Sleep(time.Millisecond * 5)

			resolve(resolutionValue)

			callbackCallsCounter++
		})

		require.Equal(t, 0, callbackCallsCounter)
		require.True(t, assertPromise(t, promise, StateSettling, nil, nil))
		promise.wg.Wait()
		require.Equal(t, 1, callbackCallsCounter)
		require.True(t, assertPromise(t, promise, StateFulfilled, resolutionValue, nil))
	})

	t.Run("Not resolved and rejected Promise is completed", func(t *testing.T) {
		rejectionReason := errors.New("some rejection reason")
		callbackCallsCounter := 0
		promise := NewPromise(func(_ Resolver, reject Rejector) {
			time.Sleep(time.Millisecond * 5)

			reject(rejectionReason)

			callbackCallsCounter++
		})

		require.Equal(t, 0, callbackCallsCounter)
		require.True(t, assertPromise(t, promise, StateSettling, nil, nil))
		promise.wg.Wait()
		require.Equal(t, 1, callbackCallsCounter)
		require.True(t, assertPromise(t, promise, StateRejected, nil, rejectionReason))
	})
}

func TestPromise(t *testing.T) {
	fakerInstance := faker.New()

	t.Run("Finally is called after Then", func(t *testing.T) {
		callsStack := NewCallsRegistry(4)

		var resolvedValue = fakerInstance.Int()

		promise := NewPromise(func(resolve Resolver, _ Rejector) {
			callsStack.Register("NewPromise.1")

			resolve(resolvedValue)
		})

		promise.Catch(func(_ error) {
			callsStack.Register("Catch.1")
		})

		promise.
			Then(func(value interface{}) (interface{}, error) {
				require.Equal(t, resolvedValue, value)

				callsStack.Register("Then.1")

				return nil, nil
			}).
			Finally(func() {
				callsStack.Register("Finally.2")
			})

		promise.Finally(func() {
			callsStack.Register("Finally.1")
		})

		callsStack.AssertCompletedBefore(t, "NewPromise.1|Finally.1|Then.1|Finally.2", time.Second)
	})

	t.Run("Finally is called after Catch", func(t *testing.T) {
		callsStack := NewCallsRegistry(4)

		var rejectionReason = fakerInstance.Lorem().Sentence(6)

		promise := NewPromise(func(_ Resolver, reject Rejector) {
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

				callsStack.Register("Catch.1")
			}).
			Finally(func() {
				callsStack.Register("Finally.2")
			})

		promise.Finally(func() {
			callsStack.Register("Finally.1")
		})

		callsStack.AssertCompletedBefore(t, "NewPromise.1|Finally.1|Catch.1|Finally.2", time.Second)
	})

	t.Run("Then can return another Promise", func(t *testing.T) {
		t.Run("Already resolved Promise", func(t *testing.T) {
			callsStack := NewCallsRegistry(5)

			var resolvedValue = fakerInstance.Int()

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				time.Sleep(time.Millisecond * 5)

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

			callsStack.AssertCompletedBefore(t, "NewPromise.1|Then.1|Finally.1|Then.2|Finally.3", time.Second)
		})

		t.Run("Already rejected Promise", func(t *testing.T) {
			callsStack := NewCallsRegistry(5)

			var resolvedValue = fakerInstance.Int()

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				time.Sleep(time.Millisecond * 5)

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
					callsStack.Register("Finally.3")
				})

			promise.Finally(func() {
				callsStack.Register("Finally.1")
			})

			callsStack.AssertCompletedBefore(t, "NewPromise.1|Then.1|Finally.1|Catch.2|Finally.3", time.Second)
		})

		t.Run("Settling Promise", func(t *testing.T) {
			callsStack := NewCallsRegistry(6)

			var resolvedValue = fakerInstance.Int()

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				time.Sleep(time.Millisecond * 5)

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
						time.Sleep(time.Millisecond * 5)

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

			callsStack.AssertCompletedBefore(t, "NewPromise.1|Then.1|Finally.1|NewPromise.1.1|Then.2|Finally.3", time.Second)
		})

		t.Run("Resolve pending Promise", func(t *testing.T) {
			callsStack := NewCallsRegistry(5)

			var resolvedValue = fakerInstance.Int()

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				time.Sleep(time.Millisecond * 5)

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
				callsStack.Register("Finally.1")
			})

			// Wait for NewPromise to be resolved and Then to be called
			time.Sleep(time.Millisecond * 10)

			callsStack.AssertCurrentCallsStackIs(t, "NewPromise.1|Then.1|Finally.1")
			callsStack.AssertThereAreNCallsLeft(t, 2)

			// Manually resolve pending promise
			require.Nil(t, pendingPromise.Resolve(resolvedThenValue))

			// Wait for next Then and Finally to be called
			time.Sleep(time.Millisecond * 5)

			callsStack.AssertCompletedBefore(t, "NewPromise.1|Then.1|Finally.1|Then.2|Finally.3", time.Second)
		})
	})

	t.Run("Then after Catch", func(t *testing.T) {
		t.Run("Then receives resolved value when Catch was not called", func(t *testing.T) {
			callsStack := NewCallsRegistry(3)

			var resolvedValue = fakerInstance.Int()

			promise := NewPromise(func(resolve Resolver, _ Rejector) {
				time.Sleep(time.Millisecond * 5)

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

			callsStack.AssertCompletedBefore(t, "NewPromise.1|Then.2|Finally.3", time.Second)
		})

		t.Run("Then receives nil value when Catch was called", func(t *testing.T) {
			callsStack := NewCallsRegistry(4)

			var rejectionValue = fakerInstance.Lorem().Sentence(6)

			promise := NewPromise(func(_ Resolver, reject Rejector) {
				time.Sleep(time.Millisecond * 5)

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

			callsStack.AssertCompletedBefore(t, "NewPromise.1|Catch.1|Then.2|Finally.3", time.Second)
		})
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
