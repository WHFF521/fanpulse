package lottery

import (
	"errors"
	"reflect"
	"testing"
)

func TestDraw_IsDeterministicAndUnique(t *testing.T) {
	candidates := []string{"u1", "u2", "u3", "u4", "u5"}
	first, err := Draw(candidates, 3, 42)
	if err != nil {
		t.Fatalf("Draw() error = %v", err)
	}
	second, _ := Draw(candidates, 3, 42)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("draws differ: %v vs %v", first, second)
	}
	seen := map[string]bool{}
	for _, id := range first {
		if seen[id] {
			t.Fatalf("duplicate winner %q", id)
		}
		seen[id] = true
	}
	if len(first) != 3 {
		t.Fatalf("winner count = %d", len(first))
	}
	if !reflect.DeepEqual(candidates, []string{"u1", "u2", "u3", "u4", "u5"}) {
		t.Fatal("input mutated")
	}
}

func TestDraw_RejectsInvalidCountAndDuplicateCandidates(t *testing.T) {
	tests := []struct {
		candidates []string
		count      int
	}{
		{[]string{"u1"}, -1},
		{[]string{"u1"}, 2},
		{[]string{"u1", "u1"}, 1},
	}
	for _, tt := range tests {
		_, err := Draw(tt.candidates, tt.count, 1)
		if !errors.Is(err, ErrInvalidDraw) {
			t.Fatalf("Draw(%v,%d) error=%v", tt.candidates, tt.count, err)
		}
	}
}
