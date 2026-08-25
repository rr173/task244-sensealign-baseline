package evidence

import (
	"math"
	"testing"

	"task244-sensealign/internal/model"
)

func TestTokenizeMixedScripts(t *testing.T) {
	got := Tokenize("Bank, 河岸边；BANK!")
	want := []string{"bank", "河", "岸", "边", "bank"}
	if len(got) != len(want) {
		t.Fatalf("Tokenize length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Tokenize[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCoverageIsSymmetricForEquivalentEvidence(t *testing.T) {
	a := &testSense{definition: "financial institution"}
	b := &testSense{definition: "financial institution"}
	score := Coverage(a.model(), nil, b.model(), nil)
	if math.Abs(score-1) > 1e-9 {
		t.Fatalf("equivalent evidence score = %v, want 1", score)
	}
}

func TestRegisterCompatibilityTreatsDisjointTagsAsConflict(t *testing.T) {
	compatible, conflict := RegisterCompatibility([]string{"formal"}, []string{"slang"})
	if compatible || !conflict {
		t.Fatalf("got compatible=%v conflict=%v for disjoint tags", compatible, conflict)
	}
}

type testSense struct{ definition string }

func (s *testSense) model() *model.Sense {
	return &model.Sense{Definition: s.definition}
}
