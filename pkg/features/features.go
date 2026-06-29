package features

import (
	"fmt"
	"sort"
	"strings"
)

type FeatureRef struct {
	Registry  string
	Namespace string
	ID        string
	Version   string
}

type Feature struct {
	ID            string
	DependsOn     []string
	InstallsAfter []string
}

func ParseFeatureRef(ref string) (FeatureRef, error) {
	// Format should be registry/namespace/id:version
	firstSlash := strings.Index(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	lastSlash := strings.LastIndex(ref, "/")

	if firstSlash == -1 || lastColon == -1 || lastSlash == -1 || lastSlash <= firstSlash || lastColon <= lastSlash {
		return FeatureRef{}, fmt.Errorf("invalid feature reference format: %s", ref)
	}

	registry := ref[:firstSlash]
	version := ref[lastColon+1:]
	id := ref[lastSlash+1 : lastColon]
	namespace := ref[firstSlash+1 : lastSlash]

	return FeatureRef{
		Registry:  registry,
		Namespace: namespace,
		ID:        id,
		Version:   version,
	}, nil
}

func SortFeatures(features []Feature) ([]Feature, error) {
	// Create a map of features in the input list for quick lookup
	inInputList := make(map[string]bool)
	for _, f := range features {
		inInputList[f.ID] = true
	}

	installed := make(map[string]bool)
	var sorted []Feature

	// Loop until all features are sorted/installed
	for len(sorted) < len(features) {
		var candidates []Feature

		for _, f := range features {
			if installed[f.ID] {
				continue
			}

			// Check dependsOn dependencies
			depsSatisfied := true
			for _, dep := range f.DependsOn {
				if !installed[dep] {
					depsSatisfied = false
					break
				}
			}

			if !depsSatisfied {
				continue
			}

			// Check installsAfter soft dependencies
			softDepsSatisfied := true
			for _, softDep := range f.InstallsAfter {
				// Only block if the soft dependency is in the input list and not installed yet
				if inInputList[softDep] && !installed[softDep] {
					softDepsSatisfied = false
					break
				}
			}

			if softDepsSatisfied {
				candidates = append(candidates, f)
			}
		}

		if len(candidates) == 0 {
			return nil, fmt.Errorf("circular dependency detected among features")
		}

		// Lexicographically sort candidate IDs to ensure deterministic sorting order
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].ID < candidates[j].ID
		})

		// Add candidates to sorted list and mark as installed
		for _, c := range candidates {
			sorted = append(sorted, c)
			installed[c.ID] = true
		}
	}

	return sorted, nil
}
