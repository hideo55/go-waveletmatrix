package waveletmatrix

import (
	"testing"
)

func TestBuildAndAccess(t *testing.T) {
	builder := NewWMBuilder()
	src := []uint64{5, 1, 0, 4, 2, 2, 0, 3}
	wm, _ := builder.Build(src)
	if wm.Size() != uint64(len(src)) {
		t.Error("Exprected", len(src), "Got", wm.Size())
	}
	for i := 0; i < len(src); i++ {
		v, found := wm.Lookup(uint64(i))
		if !found {
			t.Error("Not Found:", i)
		}
		if v != src[i] {
			t.Error("Exprected", src[i], "Got", v)
		}
	}
	if r, _ := wm.Rank(uint64(3), uint64(6)); r != uint64(0) {
		t.Error("Expected", 0, "Got", r)
	}
}
