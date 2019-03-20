package assert

import "github.com/google/go-cmp/cmp"

// Ignore configures assert to ignore the specified field paths when testing
// equality. Nested paths may be expressed with periods (e.g. "User.ID").
func Ignore(paths ...string) cmp.Option {
	var result cmp.Options
	for _, path := range paths {
		path := path
		result = append(result, cmp.FilterPath(func(p cmp.Path) bool {
			return p.String() == path
		}, cmp.Ignore()))
	}
	return result
}
