package regression

import "testing"
import "runtime"
import "sync"

// randSeq generates the candidate sequence a worker would produce from a given
// worker-specific seed, using an independent Randomizer (mirrors Minimize).
func randSeq(seed [2]uint64, length, iters uint) []string {
	r := NewRandomizer(seed)
	out := make([]string, 0, iters)
	for i := uint(0); i < iters; i++ {
		out = append(out, string(r.New(length)))
	}
	return out
}

func equalSeq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestRandomizerDeterministic verifies that candidate generation from a given
// seed is deterministic: two independent Randomizer instances seeded identically
// produce identical candidate sequences (issue 5: reproducible seeded runs).
func TestRandomizerDeterministic(t *testing.T) {
	seed := [2]uint64{42, 7}
	length := uint(10)
	iters := uint(100)
	if !equalSeq(randSeq(seed, length, iters), randSeq(seed, length, iters)) {
		t.Fatalf("same-seed sequences differ")
	}
}

// TestRandomizerReproducible verifies that a worker-specific candidate sequence
// is reproducible across separate Randomizer instances created from the same seed.
func TestRandomizerReproducible(t *testing.T) {
	seed := [2]uint64{42, 7}
	length := uint(10)
	iters := uint(100)
	if !equalSeq(randSeq(seed, length, iters), randSeq(seed, length, iters)) {
		t.Fatalf("worker-specific sequence not reproducible")
	}
}

// TestRandomizerWorkerIndependence verifies that the worker-specific seeds used
// by MinimizeThreaded ({seed[0], seed[1]+i}) yield distinct candidate sequences,
// so workers do not share a generator (issue 5: independent worker RNGs).
func TestRandomizerWorkerIndependence(t *testing.T) {
	s0, s1 := uint64(42), uint64(7)
	length := uint(10)
	iters := uint(100)
	prev := randSeq([2]uint64{s0, s1}, length, iters)
	for i := uint64(1); i < uint64(runtime.NumCPU()); i++ {
		cur := randSeq([2]uint64{s0, s1 + i}, length, iters)
		if equalSeq(prev, cur) {
			t.Fatalf("worker %d sequence equals worker %d", i, i-1)
		}
		prev = cur
	}
}

// TestRandomizerConcurrent verifies that candidate generation is race-free and
// deterministic under concurrent use (issue 5 acceptance criterion). Run with
// go test -race.
func TestRandomizerConcurrent(t *testing.T) {
	seed := [2]uint64{42, 7}
	length := uint(10)
	iters := uint(100)
	want := randSeq(seed, length, iters)
	procs := runtime.NumCPU()
	var wg sync.WaitGroup
	var mu sync.Mutex
	var fail bool
	for i := 0; i < procs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if !equalSeq(want, randSeq(seed, length, iters)) {
				mu.Lock()
				fail = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if fail {
		t.Fatalf("concurrent generation produced divergent sequences")
	}
}
