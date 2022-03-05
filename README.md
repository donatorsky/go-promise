# Go Promise
Promise library for Go.

[![GitHub license](https://img.shields.io/github/license/donatorsky/go-promise)](https://github.com/donatorsky/go-promise/blob/main/LICENSE)
[![Build](https://github.com/donatorsky/go-promise/workflows/Tests/badge.svg?branch=main)](https://github.com/donatorsky/go-promise/actions?query=branch%3Amain)

## Installation

```shell
go get github.com/donatorsky/go-promise
```

## Example

```go
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/donatorsky/go-promise"
)

func main() {
	p := promise.NewPromise(func(resolve Resolver, reject Rejector) {
		time.Sleep(time.Millisecond * 1000)

		resolve("foo")
		reject(errors.New("error from constructor"))

		fmt.Println("Constructor: actions after promise resolution")
	})

	p.
		Finally(func() {
			fmt.Println("Finally(1) <- constructor")
		}).
		Then(func(value interface{}) (result interface{}, err error) {
			fmt.Println("Then() <- Finally(1) <- constructor:", value)

			return 000, errors.New("error from Then()")
		}).
		Catch(func(reason error) {
			fmt.Println("Catch() <- Then() <- Finally(1) <- constructor:", reason)
		}).
		Finally(func() {
			fmt.Println("Finally(2) <- Catch() <- Then() <- Finally(1) <- constructor")
		})

	p.
		Then(func(value interface{}) (result interface{}, err error) {
			fmt.Println("Then() returning resolved Promise <- constructor:", value)

			time.Sleep(time.Millisecond * 150)

			return promise.Resolve("Immediately resolved Promise"), nil
		}).
		Then(func(value interface{}) (result interface{}, err error) {
			fmt.Println("Then() returning pending Promise <- Then() returning resolved Promise <- constructor:", value)

			time.Sleep(time.Millisecond * 150)

			return promise.NewPromise(func(resolve Resolver, reject Rejector) {
				time.Sleep(time.Millisecond * 250)

				resolve("Inner Promise")
				reject(errors.New("inner error"))
			}), nil
		}).
		Catch(func(reason error) {
			fmt.Println("Catch() <- Then() returning pending Promise <- Then() returning resolved Promise <- constructor:", reason)
		}).
		Then(func(value interface{}) (result interface{}, err error) {
			fmt.Println("Then() <- Catch() <- Then() returning pending Promise <- Then() returning resolved Promise <- constructor:", value)

			return nil, nil
		})

	p.
		Catch(func(reason error) {
			time.Sleep(time.Millisecond * 101)

			fmt.Println("Catch() <- constructor:", reason)
		}).
		Then(func(value interface{}) (interface{}, error) {
			time.Sleep(time.Millisecond * 101)

			fmt.Println("Then() <- Catch() <- constructor:", value)

			return 111, nil
		}).
		Then(func(value interface{}) (interface{}, error) {
			time.Sleep(time.Millisecond * 101)

			fmt.Println("Then() <- Then() <- Catch() <- constructor:", value)

			return nil, nil
		})

	p.
		Then(func(value interface{}) (interface{}, error) {
			time.Sleep(time.Millisecond * 102)

			fmt.Println("Then(1a) <- constructor:", value)

			return 222, nil
		}).
		Catch(func(reason error) {
			time.Sleep(time.Millisecond * 102)

			fmt.Println("Catch() <- Then(1a) <- constructor:", reason)
		}).
		Then(func(value interface{}) (interface{}, error) {
			time.Sleep(time.Millisecond * 101)

			fmt.Println("Then() <- Catch() <- Then(1a) <- constructor:", value)

			return 333, nil
		})

	p.
		Then(func(value interface{}) (interface{}, error) {
			time.Sleep(time.Millisecond * 103)

			fmt.Println("Then(1b) <- constructor:", value)

			return 444, nil
		}).
		Then(func(value interface{}) (interface{}, error) {
			time.Sleep(time.Millisecond * 103)

			fmt.Println("Then() <- Then(1b) <- constructor:", value)

			return 555, nil
		}).
		Catch(func(reason error) {
			time.Sleep(time.Millisecond * 102)

			fmt.Println("Catch() <- Then() <- Then(1b) <- constructor:", reason)
		}).
		Then(func(value interface{}) (interface{}, error) {
			time.Sleep(time.Millisecond * 101)

			fmt.Println("Then() <- Catch() <- Then() <- Then(1b) <- constructor:", value)

			return 666, nil
		})

	time.Sleep(time.Second * 3)

	fmt.Println(p)
	fmt.Println(promise.Resolve(5))
	fmt.Println(promise.Reject(errors.New("nope")))
}
```

The output:
```text
Constructor: actions after promise resolution
Finally(1) <- constructor
Then() returning resolved Promise <- constructor: foo
Then(1a) <- constructor: foo
Then(1b) <- constructor: foo
Then() <- Finally(1) <- constructor: foo
Catch() <- Then() <- Finally(1) <- constructor: error from Then()
Finally(2) <- Catch() <- Then() <- Finally(1) <- constructor
Then() returning pending Promise <- Then() returning resolved Promise <- constructor: Immediately resolved Promise
Then() <- Catch() <- constructor: foo
Then() <- Then() <- Catch() <- constructor: 111
Then() <- Catch() <- Then() returning pending Promise <- Then() returning resolved Promise <- constructor: Inner Promise
Then() <- Catch() <- Then(1a) <- constructor: 222
Then() <- Then(1b) <- constructor: 444
Then() <- Catch() <- Then() <- Then(1b) <- constructor: 555
&{fulfilled [] [] foo <nil>}
&{fulfilled [] [] 5 <nil>}
&{rejected [] [] <nil> 0xc000180040}
```
