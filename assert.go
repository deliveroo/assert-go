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
	"strconv"
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

// NotEqual asserts that got and want are not equal.
func NotEqual(t testingT, got, want interface{}, opts ...cmp.Option) bool {
	t.Helper()
	return assertNotEqual(t, getArg(1), got, want, opts)
}

// ErrorContains asserts that the error message contains the wanted string.
func ErrorContains(t testingT, got error, want string) bool {
	t.Helper()
	if got == nil {
		msg := "got <nil>, want not nil"
		if expr := getArg(1)(); expr != "" {
			msg = expr + ": " + msg
		}
		t.Error(msg)
		return false
	}
	if !strings.Contains(got.Error(), want) {
		msg := fmt.Sprintf("got %q, which does not contain %q", got.Error(), want)
		if expr := getArg(1)(); expr != "" {
			msg = expr + ": " + msg
		}
		t.Error(msg)
		return false
	}
	return true
}

// JSONEqual asserts that got and want are equal when represented as JSON. If
// either are already strings, they will be considered raw JSON. Otherwise, they
// will be marshaled to JSON before comparison.
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

// JSONLookup fetches a value from a JSON object using the path expression.
func JSONLookup(t testingT, subject interface{}, path string) interface{} {
	t.Helper()
	if !strings.HasPrefix(path, "$.") {
		path = "$." + path
	}
	got, err := jsonpath.JsonPathLookup(subject, path)
	Must(t, err)
	return got
}

// Contains asserts that got contains want.
// Works with strings and slices.
func Contains(t testingT, got, want interface{}, opts ...cmp.Option) bool {
	t.Helper()

	switch reflect.TypeOf(got).Kind() {
	case reflect.String:
		got2 := got.(string)
		want2 := want.(string)
		if !strings.Contains(got2, want2) {
			msg := fmt.Sprintf("(%q) does not contain: %q", got2, want2)
			if expr := getArg(1)(); expr != "" {
				msg = expr + " " + msg
			}
			t.Error(msg)
			return false
		}
		return true
	case reflect.Slice:
		return sliceContains(t, castInterfaceToSlice(got), want, getArg(1)(), opts...)
	default:
		msg := fmt.Sprintf("has unsupported type for Contains: %q", reflect.TypeOf(got).Kind())
		if expr := getArg(1)(); expr != "" {
			msg = expr + " " + msg
		}
		t.Error(msg)
		return false
	}

}

func castInterfaceToSlice(inter interface{}) []interface{} {
	v := reflect.ValueOf(inter)
	ii := make([]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		ii[i] = v.Index(i).Interface()
	}
	return ii
}

func sliceContains(t testingT, got []interface{}, want interface{}, expr string, opts ...cmp.Option) bool {
	for i := 0; i < len(got); i++ {
		opts = append(opts, defaultOpts...)
		if eq := cmp.Equal(got[i], want, opts...); eq {
			return true
		}
	}

	diff := cmp.Diff(want, nil, opts...)
	t.Error(formatDiff(expr, "does not contain: ", diff))
	return false
}

func SliceContainsAllOf(t testingT, got, want interface{}, opts ...cmp.Option) bool {
	t.Helper()

	if reflect.TypeOf(got).Kind() != reflect.Slice || reflect.TypeOf(want).Kind() != reflect.Slice {
		t.Error("SliceContainsAllOf requires slice")
		return false
	}

	missing := sliceContainsAllOf(
		t,
		castInterfaceToSlice(got),
		castInterfaceToSlice(want),
		[]interface{}{},
		opts...,
	)

	if len(missing) > 0 {
		diff := cmp.Diff(missing, nil, opts...)
		t.Error(formatDiff(getArg(1)(), "does not contain: ", diff))
		return false
	}

	return true
}

func sliceContainsAllOf(t testingT, got, want, missing []interface{}, opts ...cmp.Option) []interface{} {
	for i := 0; i < len(got); i++ {
		if eq := cmp.Equal(got[i], want[0], opts...); eq {
			if len(want) > 1 {
				return sliceContainsAllOf(t, append(got[:i], got[i+1:]...), want[1:], missing, opts...)
			} else {
				return missing
			}
		}
	}

	missing = append(missing, want[0])
	if len(want) > 1 {
		return sliceContainsAllOf(t, got, want[1:], missing, opts...)
	} else {
		return missing
	}
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

// Empty asserts that got is empty.
func Empty(t testingT, got interface{}) bool {
	t.Helper()
	if !isEmpty(got) {
		msg := fmt.Sprintf("got %s, want empty", fmtVal(got))
		if expr := getArg(1)(); expr != "" {
			msg = expr + ": " + msg
		}
		t.Error(msg)
		return false
	}
	return true
}

// NotEmpty asserts that got is not empty.
func NotEmpty(t testingT, got interface{}) bool {
	t.Helper()
	if isEmpty(got) {
		msg := fmt.Sprintf("got %s, want not empty", fmtVal(got))
		if expr := getArg(1)(); expr != "" {
			msg = expr + ": " + msg
		}
		t.Error(msg)
		return false
	}
	return true
}

// isEmpty returns true if v is nil, empty string, or a zero value.
func isEmpty(v interface{}) bool {
	if v == nil {
		return true
	}
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return value.Len() == 0
	case reflect.Ptr:
		if value.IsNil() {
			return true
		}
		return isEmpty(value.Elem().Interface())
	default:
		zeroValue := reflect.Zero(value.Type()).Interface()
		return reflect.DeepEqual(v, zeroValue)
	}
}

// isNil returns true if v is nil, or if v is an interface value containing a
// nil element.
func isNil(v interface{}) bool {
	if v == nil {
		return true
	}
	value := reflect.ValueOf(v)
	switch value.Kind() {
	case reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map,
		reflect.Ptr, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
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
		t.Error(formatDiff(expr(),"(-got +want): ", diff))
		return false
	}
	return true
}

func assertNotEqual(t testingT, expr func() string, got, notWant interface{}, opts []cmp.Option) bool {
	defer func() {
		if err := recover(); err != nil {
			t.Error("diff error:", err)
		}
	}()
	t.Helper()
	opts = append(opts, defaultOpts...)
	if diff := cmp.Diff(got, notWant, opts...); diff == "" {
		expr := expr()
		msg := fmt.Sprintf("should not equal %#v", notWant)
		if expr != "" {
			msg = expr + ": " + msg
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

func fmtVal(v interface{}) string {
	switch v := v.(type) {
	case string:
		return strconv.Quote(v)
	default:
		return fmt.Sprint(v)
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
	// Special case: if v is a string and begins with `[` or `{`, assume it's a
	// raw JSON string and unmarshal it directly.
	if s, ok := v.(string); ok {
		if strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[") {
			var r interface{}
			if err := json.Unmarshal([]byte(s), &r); err != nil {
				panic(err)
			}
			return r
		}
	}
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

func formatDiff(expr, prefix, diff string) string {
	msg := prefix + strings.TrimSpace(diff)
	if expr != "" {
		msg = expr + " " + msg
	}
	return msg
}
