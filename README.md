# assert-go

[![CircleCI](https://img.shields.io/circleci/build/github/deliveroo/assert-go)](https://circleci.com/gh/deliveroo/assert-go/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/deliveroo/assert-go)](https://goreportcard.com/report/github.com/deliveroo/assert-go)
[![GoDoc](https://godoc.org/net/http?status.svg)](https://godoc.org/github.com/deliveroo/assert-go)
[![go.dev](https://img.shields.io/badge/go.dev-pkg-007d9c.svg?style=flat)](https://pkg.go.dev/github.com/deliveroo/assert-go)

Package assert simplifies writing test assertions.

Output will contain a helpful diff rendered using as well as the source code of
the expression being tested. For example, if you call `assert.Equal(t, car.Name, "Porsche")`, the error message will include "car.Name".

Additional options and custom comparators can be registered using
`RegisterOptions`, or passed in as the last parameter to the function call. For
example, to indicate that unexported fields should be ignored on `MyType`, you
can use:

```go
 assert.RegisterOptions(
     cmpopts.IgnoreUnexported(MyType{}),
 )
```

See the [go-cmp docs](https://godoc.org/github.com/google/go-cmp/cmp) for more
options.

## Usage

```go
func Test(t *testing.T) {
    message := "foo"
    assert.Equal(t, message, "bar")
    // message (-got +want):  string(
    // - 	"foo",
    // + 	"bar",
    //   )
    p := Person{Name: "Alice"}
    assert.Equal(t, p, Person{Name: "Bob"})
    // p (-got +want): domain_test.Person{
    // - 	Name: "Alice",
    // + 	Name: "Bob",
    //  }
}
```
