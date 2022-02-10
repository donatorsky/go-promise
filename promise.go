package promise

import "sync"

type State string

const (
	StatePending   = State("pending")
	StateFulfilled = State("fulfilled")
	StateRejected  = State("rejected")
)

type Resolver func(value interface{})
type Rejector func(reason error)
type FulfillHandler func(value interface{}) (result interface{}, err error)
type RejectHandler func(reason error)
type FinallyHandler func()

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
		state: StatePending,
	}

	p.wg.Add(1)

	go func() {
		callback(p.resolve, p.reject)

		p.wg.Done()

		p.notifyObservers(p)
	}()

	return p
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

func (p *Promise) Then(handler FulfillHandler) *Promise {
	return p.registerHandlers(handler, nil, nil)
}

func (p *Promise) Catch(handler RejectHandler) *Promise {
	return p.registerHandlers(nil, handler, nil)
}

func (p *Promise) Finally(handler FinallyHandler) *Promise {
	return p.registerHandlers(nil, nil, handler)
}

func (p *Promise) registerHandlers(fulfillHandler FulfillHandler, rejectHandler RejectHandler, finallyHandler FinallyHandler) *Promise {
	newPromise := &Promise{
		state:       StatePending,
		onFulfilled: fulfillHandler,
		onRejected:  rejectHandler,
		onFinalized: finallyHandler,
	}

	if StatePending == p.state {
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
			defer p.wg.Done()

			if nil != p.onFinalized {
				defer p.onFinalized()
			}

			if nil == p.onFulfilled {
				p.resolve(promise.value)
			} else {
				if result, err := p.onFulfilled(promise.value); nil == err {
					p.resolve(result)

					if promiseResult, ok := result.(*Promise); ok {
						p.notifyObservers(promiseResult)

						return
					}
				} else {
					p.reject(err)
				}
			}

			p.notifyObservers(p)
		}()

	case StateRejected:
		p.wg.Add(1)

		go func() {
			if nil == p.onRejected {
				p.reject(promise.err)

				if nil != p.onFinalized {
					p.onFinalized()
				}

				p.wg.Done()

				p.notifyObservers(p)
			} else {
				p.onRejected(promise.err)

				if nil != p.onFinalized {
					p.onFinalized()
				}

				p.reject(promise.err)

				p.wg.Done()

				p.notifyObservers(Resolve(nil))
			}
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
	if StatePending != p.state {
		return
	}

	p.state = StateFulfilled
	p.value = value
}

func (p *Promise) reject(reason error) {
	if StatePending != p.state {
		return
	}

	p.state = StateRejected
	p.err = reason
}
