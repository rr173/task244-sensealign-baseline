package model

import "testing"

func TestValidStateValues(t *testing.T) {
	valid := []struct {
		name string
		ok   bool
	}{
		{"batch", ValidBatchStatus(string(BatchSealed))},
		{"sense", ValidSenseStatus(string(SenseSplit))},
		{"alignment", ValidAlignRelation(string(AlignRejected))},
		{"version", ValidVersionStatus(string(VersionFrozen))},
	}
	for _, tc := range valid {
		if !tc.ok {
			t.Errorf("%s state was rejected", tc.name)
		}
	}
	if ValidBatchStatus("unknown") || ValidAlignRelation("unknown") {
		t.Fatal("unknown state accepted")
	}
}
