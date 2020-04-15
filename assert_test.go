package assert

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func testGetArg(interface{}) string { return getArg(0)() }

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
		`id (-got +want): {int}:
			-: 1
			+: 2
		`)
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
		`id: should not equal 1`)

	assert(t,
		func(mt *mockTestingT) bool {
			subject := "noun"
			return NotEqual(mt, subject, subject)
		},
		`subject: should not equal "noun"`)

	assert(t,
		func(mt *mockTestingT) bool {
			subject := struct {
				ID int `json:"id"`
			}{1}
			return NotEqual(mt, subject, subject)
		},
		`subject: should not equal struct { ID int "json:\"id\"" }{ID:1}`)
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
		`subject (-got +want): root["id"]:
			-: 1
			+: 2
		`)
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
		`$.id (-got +want): {string}:
			-: "false"
			+: "true"
		`)

	assert(t,
		func(mt *mockTestingT) bool {
			return JSONPath(mt, subject, "nonexistent", 1)
		},
		`key error: nonexistent not found in object`,
	)
}

func TestAssertContains(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		out := "red orange yellow"
		return Contains(mt, out, "yellow")
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			out := "red orange yellow"
			return Contains(mt, out, "blue")
		},
		`out: got "red orange yellow", which doesn't contain "blue"`,
	)
}

func TestAssertTrue(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		enabled := true
		return True(mt, enabled)
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			enabled := false
			return True(mt, enabled)
		},
		`enabled (-got +want): {bool}:
			-: false
			+: true
		`,
	)
}

func TestAssertFalse(t *testing.T) {
	assert(t, func(mt *mockTestingT) bool {
		enabled := false
		return False(mt, enabled)
	}, ``)

	assert(t,
		func(mt *mockTestingT) bool {
			enabled := true
			return False(mt, enabled)
		},
		`enabled (-got +want): {bool}:
			-: true
			+: false
		`,
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
		`log: got "hello, world!", which doesn't match "^goodbye.*$"`,
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
			`thing (-got +want): :
			-: &assert.Thing{}
			+: <non-existent>
		`,
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
			`str (-got +want): :
			-: "foo"
			+: <non-existent>
		`,
		)
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
		`thing: got <nil>, want not nil`,
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
		`val: got "abc", want empty`,
	)

	assert(t,
		func(mt *mockTestingT) bool {
			val := []int{1, 2, 3}
			return Empty(mt, val)
		},
		`val: got [1 2 3], want empty`,
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
		`val: got "", want not empty`,
	)

	assert(t,
		func(mt *mockTestingT) bool {
			var val []int
			return NotEmpty(mt, val)
		},
		`val: got [], want not empty`,
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
		`err: got "foo", which does not contain "bar"`,
	)

	assert(t,
		func(mt *mockTestingT) bool {
			var err error
			return ErrorContains(mt, err, "foo")
		},
		`err: got <nil>, want not nil`,
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
	if mt.err != want {
		t.Errorf("error:\ngot:  %s\nwant: %s", mt.err, want)
	}
	if ret != (want == "") {
		t.Errorf("returned %v, want %v", ret, (want == ""))
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
