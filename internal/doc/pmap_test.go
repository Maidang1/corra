package doc

import (
	"math/rand/v2"
	"testing"
)

func TestPmapMatchesMap(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	var m *pmap
	ref := map[uint64]uint64{}
	var history []*pmap
	var refs []map[uint64]uint64
	for i := 0; i < 20000; i++ {
		k := uint64(r.IntN(3000))
		if r.IntN(3) == 0 {
			m = m.Delete(k)
			delete(ref, k)
		} else {
			v := r.Uint64()
			m = m.Set(k, v)
			ref[k] = v
		}
		if i%2000 == 0 {
			snap := map[uint64]uint64{}
			for k, v := range ref {
				snap[k] = v
			}
			history = append(history, m)
			refs = append(refs, snap)
		}
	}
	check := func(m *pmap, ref map[uint64]uint64) {
		t.Helper()
		if m.Len() != len(ref) {
			t.Fatalf("len %d, want %d", m.Len(), len(ref))
		}
		for k := uint64(0); k < 3000; k++ {
			v, ok := m.Get(k)
			w, wok := ref[k]
			if ok != wok || v != w {
				t.Fatalf("key %d: got %d,%v want %d,%v", k, v, ok, w, wok)
			}
		}
	}
	check(m, ref)
	// Older versions are unaffected by later changes.
	for i := range history {
		check(history[i], refs[i])
	}
}
