package promise

import (
	"errors"
	"sync"
)

var (
	ErrResolveNotPendingPromise = errors.New("cannot resolve promise that is not in pending state")
	ErrRejectNotPendingPromise  = errors.New("cannot reject promise that is not in pending state")
)

type Promise struct {
	mutex sync.RWMutex
	state State

	handlers   []func()
	operations []func()

	value interface{}
	err   error
}

func NewPromise(callback func(resolve Resolver, reject Rejector)) *Promise {
	p := Promise{
		state: StateSettling,
	}

	go func() {
		callback(p.resolve, p.reject)

		p.mutex.RLock()

		if StateSettling == p.state {
			p.state = StatePending

			p.mutex.RUnlock()

			return
		}

		p.mutex.RUnlock()

		p.notifyObservers()
	}()

	return &p
}

func Pending() *Promise {
	p := Promise{
		state: StatePending,
	}

	return &p
}

func Resolve(value interface{}) *Promise {
	return &Promise{
		state: StateFulfilled,
		value: value,
	}
}

func Reject(reason error) *Promise {
	return &Promise{
		state: StateRejected,
		err:   reason,
	}
}

func (p *Promise) Then(handler FulfillHandler) Promiser {
	return p.registerHandlers(handler, nil, nil)
}

func (p *Promise) Catch(handler RejectHandler) Promiser {
	return p.registerHandlers(nil, handler, nil)
}

func (p *Promise) Finally(handler FinallyHandler) Promiser {
	return p.registerHandlers(nil, nil, handler)
}

func (p *Promise) Resolve(value interface{}) error {
	p.mutex.Lock()

	if StatePending != p.state {
		p.mutex.Unlock()

		return ErrResolveNotPendingPromise
	}

	p.state = StateFulfilled
	p.value = value

	p.mutex.Unlock()

	p.notifyObservers()

	return nil
}

func (p *Promise) Reject(reason error) error {
	p.mutex.Lock()

	if StatePending != p.state {
		p.mutex.Unlock()

		return ErrRejectNotPendingPromise
	}

	p.state = StateRejected
	p.err = reason

	p.mutex.Unlock()

	p.notifyObservers()

	return nil
}

func (p *Promise) registerHandlers(
	fulfillHandler FulfillHandler,
	rejectHandler RejectHandler,
	finallyHandler FinallyHandler,
) *Promise {
	newPromise := Promise{
		state: StateSettling,
	}

	if nil != fulfillHandler {
		handler := func() {
			if StateRejected == p.state {
				p.operations = append(p.operations, func() {
					newPromise.state = StatePending

					_ = newPromise.Reject(p.err)
				})

				return
			}

			if result, err := fulfillHandler(p.value); err == nil {
				if promiseResult, ok := result.(*Promise); ok {
					p.operations = append(p.operations, func() {
						newPromise.state = StatePending

						promiseResult.Then(func(value interface{}) (interface{}, error) {
							_ = newPromise.Resolve(value)

							return value, nil
						})

						promiseResult.Catch(func(reason error) {
							_ = newPromise.Reject(reason)
						})
					})
				} else {
					p.operations = append(p.operations, func() {
						newPromise.state = StatePending

						_ = newPromise.Resolve(result)
					})
				}
			} else {
				p.operations = append(p.operations, func() {
					newPromise.state = StatePending

					_ = newPromise.Reject(err)
				})
			}
		}

		p.mutex.Lock()
		p.handlers = append(p.handlers, handler)
		p.mutex.Unlock()
	}

	if nil != rejectHandler {
		handler := func() {
			if StateFulfilled == p.state {
				p.operations = append(p.operations, func() {
					newPromise.state = StatePending

					_ = newPromise.Resolve(p.value)
				})

				return
			}

			rejectHandler(p.err)

			p.operations = append(p.operations, func() {
				newPromise.state = StatePending

				_ = newPromise.Resolve(nil)
			})
		}

		p.mutex.Lock()
		p.handlers = append(p.handlers, handler)
		p.mutex.Unlock()
	}

	if nil != finallyHandler {
		handler := func() {
			finallyHandler()

			p.operations = append(p.operations, func() {
				newPromise.state = StatePending

				if StateFulfilled == p.state {
					_ = newPromise.Resolve(p.value)
				} else {
					_ = newPromise.Reject(p.err)
				}
			})
		}

		p.mutex.Lock()
		p.handlers = append(p.handlers, handler)
		p.mutex.Unlock()
	}

	p.mutex.RLock()
	shouldCallHandlersImmediately := StatePending != p.state && StateSettling != p.state
	p.mutex.RUnlock()

	if shouldCallHandlersImmediately {
		p.notifyObservers()
	}

	return &newPromise
}

func (p *Promise) notifyObservers() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for _, handler := range p.handlers {
		handler()
	}

	for _, operation := range p.operations {
		operation()
	}

	p.handlers = nil
	p.operations = nil
}

func (p *Promise) resolve(value interface{}) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if StateSettling != p.state {
		return
	}

	p.state = StateFulfilled
	p.value = value
}

func (p *Promise) reject(reason error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if StateSettling != p.state {
		return
	}

	p.state = StateRejected
	p.err = reason
}
