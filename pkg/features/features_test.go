package features

import (
	"reflect"
	"testing"
)

func TestParseFeatureRef(t *testing.T) {
	tests := []struct {
		input    string
		expected FeatureRef
		wantErr  bool
	}{
		{
			input: "ghcr.io/devcontainers/features/git:1",
			expected: FeatureRef{
				Registry:  "ghcr.io",
				Namespace: "devcontainers/features",
				ID:        "git",
				Version:   "1",
			},
			wantErr: false,
		},
		{
			input: "ghcr.io/owner/repo/node:latest",
			expected: FeatureRef{
				Registry:  "ghcr.io",
				Namespace: "owner/repo",
				ID:        "node",
				Version:   "latest",
			},
			wantErr: false,
		},
		{
			input:    "invalid-ref-format",
			expected: FeatureRef{},
			wantErr:  true,
		},
	}

	for _, tc := range tests {
		got, err := ParseFeatureRef(tc.input)
		if (err != nil) != tc.wantErr {
			t.Errorf("ParseFeatureRef(%q) err = %v; wantErr = %v", tc.input, err, tc.wantErr)
			continue
		}
		if !tc.wantErr && !reflect.DeepEqual(got, tc.expected) {
			t.Errorf("ParseFeatureRef(%q) = %+v; want %+v", tc.input, got, tc.expected)
		}
	}
}

func TestSortFeatures(t *testing.T) {
	// Feature list:
	// A depends on B
	// B installsAfter C (soft dependency)
	// C is standalone
	feats := []Feature{
		{
			ID:        "feat-A",
			DependsOn: []string{"feat-B"},
		},
		{
			ID:            "feat-B",
			InstallsAfter: []string{"feat-C"},
		},
		{
			ID: "feat-C",
		},
	}

	sorted, err := SortFeatures(feats)
	if err != nil {
		t.Fatalf("SortFeatures returned unexpected error: %v", err)
	}

	// Correct order: feat-C, feat-B, feat-A (since B installsAfter C, A dependsOn B)
	expectedOrder := []string{"feat-C", "feat-B", "feat-A"}
	var gotOrder []string
	for _, f := range sorted {
		gotOrder = append(gotOrder, f.ID)
	}

	if !reflect.DeepEqual(gotOrder, expectedOrder) {
		t.Errorf("SortFeatures ordered incorrectly.\nGot: %v\nWant: %v", gotOrder, expectedOrder)
	}
}

func TestSortFeaturesCircular(t *testing.T) {
	// A depends on B
	// B depends on A
	feats := []Feature{
		{
			ID:        "feat-A",
			DependsOn: []string{"feat-B"},
		},
		{
			ID:        "feat-B",
			DependsOn: []string{"feat-A"},
		},
	}

	_, err := SortFeatures(feats)
	if err == nil {
		t.Fatal("Expected error due to circular dependency, got nil")
	}
}
