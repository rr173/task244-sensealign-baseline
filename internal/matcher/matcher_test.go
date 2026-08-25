package matcher

import (
	"testing"

	"task244-sensealign/internal/model"
)

func TestDetectOneToManyUsesThresholdAndSourceGrouping(t *testing.T) {
	candidates := []model.Candidate{
		{SourceSenseID: "s1", TargetSenseID: "t1", CovScore: 0.8},
		{SourceSenseID: "s1", TargetSenseID: "t2", CovScore: 0.5},
		{SourceSenseID: "s1", TargetSenseID: "t3", CovScore: 0.49},
		{SourceSenseID: "s2", TargetSenseID: "t4", CovScore: 0.9},
	}
	got := DetectOneToMany(candidates, 0.5)
	if len(got) != 1 || len(got["s1"]) != 2 {
		t.Fatalf("unexpected one-to-many report: %#v", got)
	}
	if got["s1"][0] != "t1" || got["s1"][1] != "t2" {
		t.Fatalf("unexpected target order: %#v", got["s1"])
	}
}

func TestTokenSetOfNormalizesLatinWords(t *testing.T) {
	got := TokenSetOf("River BANK")
	if len(got) != 2 {
		t.Fatalf("token count = %d, want 2: %#v", len(got), got)
	}
	if _, ok := got["bank"]; !ok {
		t.Fatalf("normalized token bank missing: %#v", got)
	}
}
