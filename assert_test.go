package assert

import (
	"errors"
	"fmt"
	"path"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func testGetArg(_ interface{}) string { return getArg(0)() }

func TestGetArgName(t *testing.T) {
	t.Run("variable", func(t *testing.T) {
		id := 1
		assertEQ(t, testGetArg(id), "id")
	})

	t.Run("func", func(t *testing.T) {
		id := func() int { return 1 }
		assertEQ(t, testGetArg(id()), "id()")
	})

	t.Run("field", func(t *testing.T) {
		var person struct{ id int }
		assertEQ(t, testGetArg(person.id), "person.id")
	})

	t.Run("literal", func(t *testing.T) {
		assertEQ(t, testGetArg(1), "1")
	})
}

func TestAssertEqual(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		return Equal(mt, 2, 2)
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			id := 1
			return Equal(mt, id, 2)
		},
		"id (-got +want):")
}

func TestAssertNotEqual(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		return NotEqual(mt, 1, 2)
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			id := 1
			return NotEqual(mt, id, 1)
		},
		`id should not equal 1`)

	assert(t,
		func(mt *mockTestingT) bool {
			subject := "noun"
			return NotEqual(mt, subject, subject)
		},
		`subject should not equal "noun"`)

	assert(t,
		func(mt *mockTestingT) bool {
			subject := struct {
				ID int `json:"id"`
			}{1}
			return NotEqual(mt, subject, subject)
		},
		`subject should not equal struct { ID int "json:\"id\"" }{ID:1}`)
}

func TestAssertJSONEqual(t *testing.T) {
	subject := struct {
		ID int `json:"id"`
	}{1}

	assert(t, func(mt *mockTestingT) bool {
		return JSONEqual(mt, subject, map[string]interface{}{"id": 1})
	}, ``)

	assert(t, func(mt *mockTestingT) bool {
		return JSONEqual(mt, `{"id": 1}`, map[string]interface{}{"id": 1})
	}, ``)

	assert(t, func(mt *mockTestingT) bool {
		return JSONEqual(mt, `{"id": 1}`, `{
			"id": 1
		}`)
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			return JSONEqual(mt, subject, map[string]interface{}{"id": 2})
		},
		`subject (-got +want):`)
}

func TestAssertJSONPath(t *testing.T) {
	subject := struct {
		ID string `json:"id"`
	}{"false"}

	assert(t, func(mt *mockTestingT) bool {
		return JSONPath(mt, subject, "id", "false")
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			return JSONPath(mt, subject, "id", "true")
		},
		`$.id (-got +want):`)

	assert(t,
		func(mt *mockTestingT) bool {
			return JSONPath(mt, subject, "nonexistent", 1)
		},
		`key error: nonexistent not found in object`,
	)
}

func TestAssertContains(t *testing.T) {
	t.Run("when input is string", func(t *testing.T) {
		t.Run("when contains input", func(t *testing.T) {
			assert(t, func(mt *mockTestingT) bool {
				out := "red orange yellow"
				return Contains(mt, out, "yellow")
			}, ``)
		})

		t.Run("when does not contain input", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := "red orange yellow"
					return Contains(mt, out, "blue")
				},
				`out ("red orange yellow") does not contain: "blue"`,
			)
		})
	})

	t.Run("when input is struct", func(t *testing.T) {
		type x struct {
			A int
			B bool
		}

		t.Run("when contains input", func(t *testing.T) {
			assert(t, func(mt *mockTestingT) bool {
				out := []x{
					{A: 1, B: true},
					{A: 2, B: true},
				}
				return Contains(mt, out, x{A: 1, B: true})
			}, ``)
		})

		t.Run("when contains input if cmpopts are passed", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := []x{
						{A: 1, B: true},
						{A: 2, B: true},
					}
					return Contains(
						mt,
						out,
						x{A: 1, B: false},
						cmpopts.IgnoreFields(x{}, "B"),
					)
				},
				"",
			)
		})

		t.Run("when does not contain input", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := []x{
						{A: 1, B: true},
						{A: 2, B: true},
					}
					want := x{A: 1, B: false}
					return Contains(mt, out, want)
				},
				`out does not contain:`,
			)
		})
	})

	t.Run("when input type is not supported", func(t *testing.T) {
		assert(t,
			func(mt *mockTestingT) bool {
				out := 1
				return Contains(mt, out, 1)
			},
			`out has unsupported type for Contains: "int"`,
		)
	})

	t.Run("when got is string but want is not", func(t *testing.T) {
		assert(t,
			func(mt *mockTestingT) bool {
				out := "red orange yellow"
				return Contains(mt, out, 1)
			},
			`got and want must be the same type`,
		)
	})
}

func TestAssertContainsAll(t *testing.T) {
	t.Run("when input is slice of strings", func(t *testing.T) {
		t.Run("when contains all of input", func(t *testing.T) {
			assert(t, func(mt *mockTestingT) bool {
				out := []string{"red", "orange", "yellow"}
				want := []string{"yellow"}
				return ContainsAll(mt, out, want)
			}, ``)
		})

		t.Run("when input has duplicates", func(t *testing.T) {
			assert(t, func(mt *mockTestingT) bool {
				out := []string{"red", "orange", "yellow"}
				want := []string{"yellow", "yellow"}
				return ContainsAll(mt, out, want)
			}, `out does not contain:`)
		})

		t.Run("when does not contain all of input", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := []string{"red", "orange", "yellow"}
					want := []string{"blue", "red", "purple"}
					return ContainsAll(mt, out, want)
				}, `out does not contain:`)
		})

		t.Run("when want contains more than got", func(t *testing.T) {
			assert(t, func(mt *mockTestingT) bool {
				out := []string{"red", "orange", "yellow"}
				want := []string{"red", "orange", "yellow", "blue"}
				return ContainsAll(mt, out, want)
			}, `out does not contain:`)
		})

		t.Run("when does not contain any of input", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := []string{"red", "orange", "yellow"}
					want := []string{"blue", "purple"}
					return ContainsAll(mt, out, want)
				}, `out does not contain:`)
		})
	})

	t.Run("when input is slice of structs", func(t *testing.T) {
		type x struct {
			A int
			B bool
		}

		t.Run("when contains all of input", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := []x{
						{A: 1, B: true},
						{A: 2, B: true},
					}
					want := []x{
						{A: 1, B: true},
					}
					return ContainsAll(mt, out, want)
				},
				``)
		})

		t.Run("when contains all of input if cmpopts are passed", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := []x{
						{A: 1, B: true},
						{A: 2, B: true},
					}
					want := []x{
						{A: 1, B: false},
					}
					return ContainsAll(mt, out, want, cmpopts.IgnoreFields(x{}, "B"))
				},
				``)
		})

		t.Run("when does not contain any of input", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := []x{
						{A: 1, B: true},
						{A: 2, B: true},
					}
					want := []x{
						{A: 1, B: true},
						{A: 3, B: true},
					}
					return ContainsAll(mt, out, want)
				},
				`out does not contain:`)
		})

		t.Run("when does not contain any of input", func(t *testing.T) {
			assert(t,
				func(mt *mockTestingT) bool {
					out := []x{
						{A: 1, B: true},
						{A: 2, B: true},
					}
					want := []x{
						{A: 3, B: true},
					}
					return ContainsAll(mt, out, want)
				},
				`out does not contain:`)
		})
	})

	t.Run("when want is not a slice", func(t *testing.T) {
		assert(t,
			func(mt *mockTestingT) bool {
				out := []int{1, 2, 3}
				want := 3
				return ContainsAll(mt, out, want)
			},
			`want must be slice`)
	})

	t.Run("when input is of unsupported type", func(t *testing.T) {
		assert(t,
			func(mt *mockTestingT) bool {
				out := -1
				return ContainsAll(mt, out, 3)
			},
			`out has unsupported type for ContainsAll: "int"`)
	})
}

func TestAssertTrue(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		const enabled = true
		return True(mt, enabled)
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			const enabled = false
			return True(mt, enabled)
		},
		`enabled (-got +want):`,
	)
}

func TestAssertFalse(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		const enabled = false
		return False(mt, enabled)
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			const enabled = true
			return False(mt, enabled)
		},
		`enabled (-got +want):`,
	)
}

func TestAssertMatch(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		log := "hello, world!"
		return Match(mt, log, "^hello.*$")
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			log := "hello, world!"
			return Match(mt, log, "^goodbye.*$")
		},
		`log ("hello, world!") doesn't match "^goodbye.*$"`,
	)

	assert(t, func(mt *mockTestingT) bool {
		return Match(mt, "", `(`)
	}, "regexp error: error parsing regexp: missing closing ): `(`")
}

func TestAssertMust(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		mt := &mockTestingT{}
		var err error
		Must(mt, err)
		assertEQ(t, mt.fatal, "")
	})
	t.Run("with error", func(t *testing.T) {
		mt := &mockTestingT{}
		err := errors.New("an error occurred")
		Must(mt, err)
		assertEQ(t, mt.fatal, "an error occurred")
	})
}

func TestAssertNil(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		return Nil(mt, nil)
	}, ``)

	t.Run("with a struct", func(t *testing.T) {
		type Thing struct{}

		assert(t,
			func(mt *mockTestingT) bool {
				thing := &Thing{}
				return Nil(mt, thing)
			},
			`thing (-got +want):`,
		)

		assert(t,
			func(mt *mockTestingT) bool {
				var thing *Thing
				return Nil(mt, thing)
			},
			``,
		)
	})
	t.Run("with a string value", func(t *testing.T) {
		assert(t,
			func(mt *mockTestingT) bool {
				str := "foo"
				return Nil(mt, str)
			},
			`str (-got +want):`)
	})
}

func TestAssertNotNil(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		return NotNil(mt, &struct{}{})
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			var thing *struct{}
			return NotNil(mt, thing)
		},
		`thing was not nil`,
	)
}

func TestAssertEmpty(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		return Empty(mt, "")
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			val := "abc"
			return Empty(mt, val)
		},
		`val ("abc") was not empty`,
	)

	assert(t,
		func(mt *mockTestingT) bool {
			val := []int{1, 2, 3}
			return Empty(mt, val)
		},
		`val ([1 2 3]) was not empty`,
	)
}

func TestAssertNotEmpty(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		return NotEmpty(mt, "text")
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			var val string
			return NotEmpty(mt, val)
		},
		`val was empty`,
	)

	assert(t,
		func(mt *mockTestingT) bool {
			var val []int
			return NotEmpty(mt, val)
		},
		`val was empty`,
	)
}

func TestErrorContains(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		err := fmt.Errorf("foo bar")
		return ErrorContains(mt, err, "foo")
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			err := fmt.Errorf("foo")
			return ErrorContains(mt, err, "bar")
		},
		`err ("foo") does not contain "bar"`,
	)

	assert(t,
		func(mt *mockTestingT) bool {
			var err error
			return ErrorContains(mt, err, "foo")
		},
		`err was not nil`,
	)
}

func removeLeadingTabs(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimPrefix(l, "		")
	}
	return strings.Join(lines, "\n")
}

func assert(t *testing.T, fn func(mt *mockTestingT) bool, want string) {
	t.Helper()
	want = removeLeadingTabs(want)
	mt := &mockTestingT{}
	ret := fn(mt)
	if want != "" && !strings.HasPrefix(mt.err, want) {
		t.Errorf("error:\ngot:  %s\nwant prefix: %s", mt.err, want)
	}
	if ret != (want == "") {
		t.Errorf("returned %v, want %v", ret, want == "")
	}
}

type mockTestingT struct {
	err, fatal string
}

func (t *mockTestingT) Helper()                   {}
func (t *mockTestingT) Error(args ...interface{}) { t.err = fmt.Sprint(args...) }
func (t *mockTestingT) Fatal(args ...interface{}) { t.fatal = fmt.Sprint(args...) }

func assertEQ(t *testing.T, got, want interface{}) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRegisterOptions(t *testing.T) {
	compareTrue := cmp.Comparer(func(int, int) bool { return true })
	compareFalse := cmp.Comparer(func(int, int) bool { return false })

	tests := []struct {
		fn            func(t testingT, got, want interface{}, opts ...cmp.Option) bool
		fnGot, fnWant interface{}
		regOpt        cmp.Option
	}{
		{
			fn:     Equal,
			fnGot:  1,
			fnWant: 2,
			regOpt: compareTrue,
		},
		{
			fn:     NotEqual,
			fnGot:  1,
			fnWant: 1,
			regOpt: compareFalse,
		},
		{
			fn:     Contains,
			fnGot:  []int{1},
			fnWant: 2,
			regOpt: compareTrue,
		},
		{
			fn:     ContainsAll,
			fnGot:  []int{1},
			fnWant: []int{2},
			regOpt: compareTrue,
		},
	}
	for _, tt := range tests {
		funcPath := runtime.FuncForPC(reflect.ValueOf(tt.fn).Pointer()).Name()
		funcName := strings.Split(path.Base(funcPath), ".")[1]

		t.Run(funcName, func(t *testing.T) {
			defer resetDefaultOpts()
			RegisterOptions(tt.regOpt)
			mt := &mockTestingT{}
			if !tt.fn(mt, tt.fnGot, tt.fnWant) {
				t.Errorf("%s did not use RegisterOptions", funcName)
			}
		})
	}
}

var resetDefaultOpts = func() func() {
	defaultDefaultOpts := defaultOpts
	return func() { defaultOpts = defaultDefaultOpts }
}()
