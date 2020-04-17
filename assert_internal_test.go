package assert

import "testing"

func TestIsEmpty(t *testing.T) {
	type dummy struct{ ID int }
	tests := []struct {
		val   interface{}
		empty bool
	}{
		// Empty
		{nil, true},
		{(*dummy)(nil), true},
		{&dummy{}, true},
		{"", true},
		{0, true},
		{[]int{}, true},
		{([]int)(nil), true},
		{map[string]int{}, true},

		// Not Empty
		{&dummy{ID: 1}, false},
		{"abc", false},
		{1, false},
		{[]int{1, 2, 3}, false},
		{map[string]int{"a": 1}, false},
	}

	for i, tt := range tests {
		got := isEmpty(tt.val)
		if got != tt.empty {
			t.Errorf("%d: got %v, want %v", i, got, tt.empty)
		}
	}
}
