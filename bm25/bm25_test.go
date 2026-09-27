package bm25

import (
	"math"
	"testing"
)

func TestScore(t *testing.T) {
	if got := Score(1, 1, 2, 1, 1, K1, B); math.Abs(got-math.Ln2) > 1e-12 {
		t.Errorf("Score(1, 1, 2, 1, 1) = %v, want ln 2", got)
	}
	if got := Score(0, 1, 2, 1, 1, K1, B); got != 0 {
		t.Errorf("Score with tf 0 = %v, want 0", got)
	}
	short, long := Score(3, 5, 100, 10, 20, K1, B), Score(3, 5, 100, 40, 20, K1, B)
	if long >= short {
		t.Errorf("b=%v: long doc %v, short doc %v; want long < short", B, long, short)
	}
	short, long = Score(3, 5, 100, 10, 20, K1, 0), Score(3, 5, 100, 40, 20, K1, 0)
	if long != short {
		t.Errorf("b=0: long doc %v, short doc %v; want equal", long, short)
	}
}

func TestQueryTermWeight(t *testing.T) {
	if got := QueryTermWeight(1, 8); got != 1 {
		t.Errorf("QueryTermWeight(1, 8) = %v, want 1", got)
	}
	if got := QueryTermWeight(2, 8); math.Abs(got-1.8) > 1e-12 {
		t.Errorf("QueryTermWeight(2, 8) = %v, want 1.8", got)
	}
	if got, want := QueryTermWeight(2, 0), QueryTermWeight(2, K3); got != want {
		t.Errorf("QueryTermWeight(2, 0) = %v, want %v", got, want)
	}
}
