// Package assert simplifies writing test assertions.
//
// Output will contain a helpful diff rendered using as well as the source code
// of the expression being tested. For example, if you call
// assert.Equal(t, car.Name, "Porsche"), the error message will include
// "car.Name".
//
// Additional options and custom comparators can be registered using
// RegisterOptions, or passed in as the last parameter to the function call. For
// example, to indicate that unexported fields should be ignored on MyType, you
// can use:
//
//      assert.RegisterOptions(
//          cmpopts.IgnoreUnexported(MyType{}),
//      )
//
// See the go-cmp docs for more options:
// https://godoc.org/github.com/google/go-cmp/cmp.
package assert

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/oliveagle/jsonpath"
)

// testingT is a simplified interface of the testing.T.
type testingT interface {
	Helper()
	Error(args ...interface{})
	Fatal(args ...interface{})
}

// defaultOpts is the default set of options passed to cmp.Diff for
// assert.Equals.
var defaultOpts = []cmp.Option{
	// Compare errors by their messages.
	cmp.Comparer(func(x, y error) bool {
		if x == nil && y == nil {
			return true
		} else if x != nil && y != nil {
			return x.Error() == y.Error()
		}
		return false
	}),
}

// RegisterOptions registers a default option for all tests in the current
// package. It's intended to be used in an init function, like:
//
//     func init() {
//         assert.RegisterOptions(
//             cmp.Comparer(func(x, y *Thing) bool {
//                 return x.ID == y.ID
//             }),
//         )
//     }
//
// Note that due to how "go test" operates, these options will not leak between
// packages.
func RegisterOptions(opts ...cmp.Option) {
	defaultOpts = append(defaultOpts, opts...)
}

// Equal asserts that got and want are assertEqual.
func Equal(t testingT, got, want interface{}, opts ...cmp.Option) bool {
	t.Helper()
	return assertEqual(t, getArg(1), got, want, opts)
}

// JSONEqual asserts that got and want are assertEqual when marshaled as JSON.
func JSONEqual(t testingT, got, want interface{}, opts ...cmp.Option) bool {
	t.Helper()
	return assertEqual(t, getArg(1), toJSON(got), toJSON(want), opts)
}

// JSONPath asserts that evaluating the path expression against the subject
// results in want. The subject and want parameters are both converted to their
// JSON representation before being evaluated.
func JSONPath(t testingT, subject interface{}, path string, want interface{}, opts ...cmp.Option) bool {
	t.Helper()
	subject, want = toJSON(subject), toJSON(want)
	if !strings.HasPrefix(path, "$.") {
		path = "$." + path
	}
	var err interface{}
	got, err := jsonpath.JsonPathLookup(subject, path)
	if err != nil {
		t.Error(err)
		return false
	}
	return assertEqual(t, func() string { return path }, got, want, opts)
}

// Contains asserts that got contains want.
func Contains(t testingT, got, want string) bool {
	t.Helper()
	if !strings.Contains(got, want) {
		msg := fmt.Sprintf("got %q, which doesn't contain %q", got, want)
		if expr := getArg(1)(); expr != "" {
			msg = expr + ": " + msg
		}
		t.Error(msg)
		return false
	}
	return true
}

// True asserts that got is true.
func True(t testingT, got bool) bool {
	t.Helper()
	return assertEqual(t, getArg(1), got, true, nil)
}

// False asserts that got is false.
func False(t testingT, got bool) bool {
	t.Helper()
	return assertEqual(t, getArg(1), got, false, nil)
}

// Match asserts that got matches the regex want.
func Match(t testingT, got, want string) bool {
	t.Helper()
	match, err := regexp.MatchString(want, got)
	if err != nil {
		t.Error("regexp error: ", err)
		return false
	}
	if !match {
		msg := fmt.Sprintf("got %q, which doesn't match %q", got, want)
		if expr := getArg(1)(); expr != "" {
			msg = expr + ": " + msg
		}
		t.Error(msg)
		return false
	}
	return true
}

// Must asserts that err is nil, calling t.Fatal otherwise.
func Must(t testingT, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// Nil asserts that got is nil.
func Nil(t testingT, got interface{}) bool {
	t.Helper()
	if isNil(got) {
		return true
	}
	return assertEqual(t, getArg(1), got, nil, nil)
}

// NotNil asserts that got is not nil.
func NotNil(t testingT, got interface{}) bool {
	t.Helper()
	if isNil(got) {
		msg := "got <nil>, want not nil"
		if expr := getArg(1)(); expr != "" {
			msg = expr + ": " + msg
		}
		t.Error(msg)
		return false
	}
	return true
}

// isNil returns true if v is nil, or if v is an interface value containing a
// nil element.
func isNil(v interface{}) bool {
	return v == nil || reflect.ValueOf(v).IsNil()
}

func assertEqual(t testingT, expr func() string, got, want interface{}, opts []cmp.Option) bool {
	defer func() {
		if err := recover(); err != nil {
			t.Error("diff error:", err)
		}
	}()
	t.Helper()
	opts = append(opts, defaultOpts...)
	if diff := cmp.Diff(got, want, opts...); diff != "" {
		expr := expr()
		msg := "(-got +want): " + diff
		if expr != "" {
			msg = expr + " " + msg
		}
		t.Error(msg)
		return false
	}
	return true
}

// getArg finds the source code for the given function argument. For example, if
// function f was called like `f(id)`, getArg(0) would return "id".
func getArg(arg int) func() string {
	// Find the name of the assertion function (e.g. Equal).
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc).Name()
	if idx := strings.LastIndex(fn, "."); idx != -1 {
		fn = fn[idx+1:]
	}

	// Open the source code of the calling function, find the function call, and
	// return the source for the argument.
	_, filename, line, _ := runtime.Caller(2)
	return func() string {
		file, err := os.Open(filename)
		if err != nil {
			panic(err)
		}
		b, err := ioutil.ReadAll(file)
		if err != nil {
			panic(err)
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", string(b), parser.ParseComments)
		if err != nil {
			panic(err)
		}
		expr := ""
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				return false
			}
			if fset.Position(n.Pos()).Line == line {
				switch x := n.(type) {
				case *ast.CallExpr:
					if !isFunc(x, fn) {
						return true
					}
					arg := x.Args[arg]
					start, end := fset.Position(arg.Pos()), fset.Position(arg.End())
					expr = string(b)[start.Offset:end.Offset]
				}
			}
			return true
		})
		return expr
	}
}

func isFunc(expr *ast.CallExpr, name string) bool {
	switch x := expr.Fun.(type) {
	case *ast.SelectorExpr:
		return x.Sel.Name == name
	case *ast.Ident:
		return x.Name == name
	}
	return false
}

// toJSON transforms v into simple JSON types (maps and arrays).
func toJSON(v interface{}) interface{} {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var r interface{}
	if err := json.Unmarshal(b, &r); err != nil {
		panic(err)
	}
	return r
}
