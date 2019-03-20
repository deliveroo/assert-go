package assert_test

import (
	"testing"
	"time"

	"github.com/deliveroo/assert-go"
	"github.com/google/go-cmp/cmp"
)

func TestIgnore(t *testing.T) {
	type User struct {
		ID      int
		Name    string
		Created time.Time
	}

	u1 := User{ID: 1, Name: "Alice", Created: time.Now()}
	u2 := User{ID: 1, Name: "Bob", Created: time.Now().Add(5 * time.Minute)}

	assert.False(t, cmp.Equal(u1, u2))
	assert.True(t, cmp.Equal(u1, u2, assert.Ignore("Created", "Name")))
}
