package doctest

import (
	"testing"

	"github.com/Maidang1/corra/internal/doc"
)

// CheckLog checks a commit's op log: replayed on the document before, it
// gives the document after, IDs included; inverted and replayed backwards
// on the document after, it gives the document before.
func CheckLog(t testing.TB, c doc.Commit) {
	t.Helper()
	tx := c.Before.Begin()
	for _, o := range c.Ops {
		tx.Apply(o)
	}
	if got := tx.State().Doc; !doc.Equal(got, c.After.Doc) {
		t.Fatalf("forward replay differs:\n got  %s\n want %s\n ops %v", DumpIDs(got), DumpIDs(c.After.Doc), c.Ops)
	}
	tx = c.After.Begin()
	for i := len(c.Ops) - 1; i >= 0; i-- {
		tx.Apply(c.Ops[i].Invert())
	}
	if got := tx.State().Doc; !doc.Equal(got, c.Before.Doc) {
		t.Fatalf("inverse replay differs:\n got  %s\n want %s\n ops %v", DumpIDs(got), DumpIDs(c.Before.Doc), c.Ops)
	}
}
