# Assert

[![CircleCI](https://circleci.com/gh/deliveroo/assert-go.svg?style=svg&circle-token=b3594d1037dc390b8d3e39383fdd70a02fced4fe)](https://circleci.com/gh/deliveroo/assert-go)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg)](http://godoc.deliveroo.net/github.com/deliveroo/assert-go) 

Package assert simplifies writing test assertions.

Output will contain a helpful diff rendered using as well as the source code of
the expression being tested. For example, if you call `assert.Equal(t, car.Name,
"Porsche")`, the error message will include "car.Name".

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
    // message (-got +want): {string}:
    //          -: "foo"
    //          +: "bar"

    p := Person{Name: "Alice"}
    assert.Equal(t, p, Person{Name: "Bob"})
    // p (-got +want): {domain_test.Person}.Name:
    //          -: "Alice"
    //          +: "Bob"
}
```
