package waveletmatrix

import (
	"testing"
)

func TestBuild(t *testing.T) {
	builder := NewWMBuilder()
	src := []uint64{1,2}
	wm, _ := builder.Build(src)
	_ = wm
}
