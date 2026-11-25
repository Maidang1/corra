package doc_test

import (
	"testing"

	"github.com/Maidang1/corra/internal/doc/doctest"
)

func TestRandomCommands(t *testing.T) {
	n := doctest.Iterations(40)
	for seed := uint64(1); seed <= 5; seed++ {
		doctest.Fuzz(t, seed, n, 30)
	}
}
