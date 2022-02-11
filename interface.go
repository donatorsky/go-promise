package promise

type State string

const (
	StatePending   = State("pending")
	StateSettling  = State("settling")
	StateFulfilled = State("fulfilled")
	StateRejected  = State("rejected")
)

type Resolver func(value interface{})
type Rejector func(reason error)
type FulfillHandler func(value interface{}) (result interface{}, err error)
type RejectHandler func(reason error)
type FinallyHandler func()

type Promiser interface {
	Then(handler FulfillHandler) Promiser
	Catch(handler RejectHandler) Promiser
	Finally(handler FinallyHandler) Promiser
	Resolve(value interface{}) error
	Reject(reason error) error
}
