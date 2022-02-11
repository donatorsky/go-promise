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
	wg          sync.WaitGroup
	state       State
	observers   []*Promise
	onFulfilled FulfillHandler
	onRejected  RejectHandler
	onFinalized FinallyHandler

	value interface{}
	err   error
}

func NewPromise(callback func(resolve Resolver, reject Rejector)) *Promise {
	p := &Promise{
		state: StateSettling,
	}

	p.wg.Add(1)

	go func() {
		callback(p.resolve, p.reject)

		p.wg.Done()

		p.notifyObservers(p)
	}()

	return p
}

func Pending() *Promise {
	p := Promise{
		state: StatePending,
	}

	p.wg.Add(1)

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
	if StatePending != p.state {
		return ErrResolveNotPendingPromise
	}

	p.state = StateFulfilled
	p.value = value

	p.wg.Done()

	p.notifyObservers(p)

	return nil
}

func (p *Promise) Reject(reason error) error {
	if StatePending != p.state {
		return ErrRejectNotPendingPromise
	}

	p.state = StateRejected
	p.err = reason

	p.wg.Done()

	p.notifyObservers(p)

	return nil
}

func (p *Promise) registerHandlers(
	fulfillHandler FulfillHandler,
	rejectHandler RejectHandler,
	finallyHandler FinallyHandler,
) *Promise {
	newPromise := &Promise{
		state:       StateSettling,
		onFulfilled: fulfillHandler,
		onRejected:  rejectHandler,
		onFinalized: finallyHandler,
	}

	if StatePending == p.state || StateSettling == p.state {
		p.addObserver(newPromise)
	} else {
		newPromise.receiveNotification(p)
	}

	return newPromise
}

func (p *Promise) addObserver(promise *Promise) {
	p.observers = append(p.observers, promise)
}

func (p *Promise) receiveNotification(promise *Promise) {
	promise.wg.Wait()

	switch promise.state {
	case StateFulfilled:
		p.wg.Add(1)

		go func() {
			if nil == p.onFulfilled {
				p.resolve(promise.value)
			} else {
				if result, err := p.onFulfilled(promise.value); nil == err {
					p.resolve(result)

					if promiseResult, ok := result.(*Promise); ok {
						if nil != p.onFinalized {
							p.onFinalized()
						}

						p.wg.Done()

						p.notifyObservers(promiseResult)

						return
					}
				} else {
					p.reject(err)
				}
			}

			if nil != p.onFinalized {
				p.onFinalized()
			}

			p.wg.Done()

			p.notifyObservers(p)
		}()

	case StateRejected:
		p.wg.Add(1)

		go func() {
			if nil == p.onRejected {
				defer p.notifyObservers(p)
			} else {
				p.onRejected(promise.err)

				defer p.notifyObservers(Resolve(nil))
			}

			p.reject(promise.err)

			if nil != p.onFinalized {
				p.onFinalized()
			}

			p.wg.Done()
		}()

	default:
		panic("unexpected promise state: " + promise.state)
	}
}

func (p *Promise) notifyObservers(promise *Promise) {
	for _, observer := range p.observers {
		observer.receiveNotification(promise)
	}

	p.observers = nil
}

func (p *Promise) resolve(value interface{}) {
	if StateSettling != p.state {
		return
	}

	p.state = StateFulfilled
	p.value = value
}

func (p *Promise) reject(reason error) {
	if StateSettling != p.state {
		return
	}

	p.state = StateRejected
	p.err = reason
}
