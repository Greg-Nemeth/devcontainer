package varsub

import (
	"regexp"
	"testing"
)

func TestSubstitute(t *testing.T) {
	ctx := SubstitutionContext{
		Platform:                 "linux",
		LocalWorkspaceFolder:     "/foo/bar",
		ContainerWorkspaceFolder: "/baz/blue",
		ConfigFile:               "/foo/bar/baz.json",
		Env: map[string]string{
			"baz": "somevalue",
		},
	}

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "environment variables",
			input:    map[string]interface{}{"foo": "bar${env:baz}bar"},
			expected: map[string]interface{}{"foo": "barsomevaluebar"},
		},
		{
			name:     "localWorkspaceFolder",
			input:    map[string]interface{}{"foo": "bar${localWorkspaceFolder}bar"},
			expected: map[string]interface{}{"foo": "bar/foo/barbar"},
		},
		{
			name:     "containerWorkspaceFolder",
			input:    map[string]interface{}{"foo": "bar${containerWorkspaceFolder}bar"},
			expected: map[string]interface{}{"foo": "bar/baz/bluebar"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := Substitute(ctx, tc.input)
			resMap, ok1 := res.(map[string]interface{})
			expMap, ok2 := tc.expected.(map[string]interface{})
			if ok1 && ok2 {
				for k, v := range expMap {
					if resMap[k] != v {
						t.Errorf("Expected %q, got %q", v, resMap[k])
					}
				}
			} else {
				t.Errorf("Type mismatch or parsing error")
			}
		})
	}
}

func TestLocalWorkspaceFolderBasename(t *testing.T) {
	ctx := SubstitutionContext{
		Platform:                 "linux",
		LocalWorkspaceFolder:     "/foo/red",
		ContainerWorkspaceFolder: "/baz/${localWorkspaceFolderBasename}",
		ConfigFile:               "/foo/bar/baz.json",
		Env:                      map[string]string{"baz": "somevalue"},
	}

	raw := map[string]interface{}{
		"foo": "bar${containerWorkspaceFolder}bar",
	}

	res := Substitute(ctx, raw)
	resMap := res.(map[string]interface{})
	if resMap["foo"] != "bar/baz/redbar" {
		t.Errorf("Expected bar/baz/redbar, got %q", resMap["foo"])
	}
}

func TestEnvWithDefaultValue(t *testing.T) {
	ctxNoEnv := SubstitutionContext{
		Platform:                 "linux",
		LocalWorkspaceFolder:     "/foo/bar",
		ContainerWorkspaceFolder: "/baz/blue",
		ConfigFile:               "/foo/bar/baz.json",
		Env:                      map[string]string{},
	}

	res1 := Substitute(ctxNoEnv, map[string]interface{}{"foo": "bar${localEnv:baz:default}bar"})
	if res1.(map[string]interface{})["foo"] != "bardefaultbar" {
		t.Errorf("Expected bardefaultbar, got %q", res1.(map[string]interface{})["foo"])
	}

	res2 := Substitute(ctxNoEnv, map[string]interface{}{"foo": "bar${localEnv:baz}bar"})
	if res2.(map[string]interface{})["foo"] != "barbar" {
		t.Errorf("Expected barbar, got %q", res2.(map[string]interface{})["foo"])
	}

	ctxWithEnv := SubstitutionContext{
		Platform:                 "linux",
		LocalWorkspaceFolder:     "/foo/bar",
		ContainerWorkspaceFolder: "/baz/blue",
		ConfigFile:               "/foo/bar/baz.json",
		Env:                      map[string]string{"baz": "somevalue"},
	}
	res3 := Substitute(ctxWithEnv, map[string]interface{}{"foo": "bar${localEnv:baz:default}bar"})
	if res3.(map[string]interface{})["foo"] != "barsomevaluebar" {
		t.Errorf("Expected barsomevaluebar, got %q", res3.(map[string]interface{})["foo"])
	}

	res4 := Substitute(ctxNoEnv, map[string]interface{}{"foo": "bar${localEnv:baz:default:a:b:c}bar"})
	if res4.(map[string]interface{})["foo"] != "bardefaultbar" {
		t.Errorf("Expected bardefaultbar, got %q", res4.(map[string]interface{})["foo"])
	}
}

func TestContainerEnv(t *testing.T) {
	raw := map[string]interface{}{
		"foo": "bar${containerEnv:baz:default}bar",
	}
	res := ContainerSubstitute("linux", "/foo/bar/baz.json", map[string]string{}, raw)
	if res.(map[string]interface{})["foo"] != "bardefaultbar" {
		t.Errorf("Expected bardefaultbar, got %q", res.(map[string]interface{})["foo"])
	}
}

func TestDevContainerId(t *testing.T) {
	raw := map[string]interface{}{
		"test": "${devcontainerId}",
	}

	res1 := BeforeContainerSubstitute(map[string]string{"a": "b"}, raw)
	res1Map := res1.(map[string]interface{})
	id1 := res1Map["test"].(string)

	matched, err := regexp.MatchString(`^[0-9a-v]{52}$`, id1)
	if err != nil || !matched {
		t.Errorf("Expected 52-char base32 string, got %q", id1)
	}

	res2 := BeforeContainerSubstitute(map[string]string{"a": "b", "c": "d"}, raw)
	res2Map := res2.(map[string]interface{})
	id2 := res2Map["test"].(string)

	if id1 == id2 {
		t.Errorf("Expected different IDs for different labels, got identical: %q", id1)
	}

	res3 := BeforeContainerSubstitute(map[string]string{"c": "d", "a": "b"}, raw)
	res3Map := res3.(map[string]interface{})
	id3 := res3Map["test"].(string)

	if id2 != id3 {
		t.Errorf("Expected same ID regardless of label order, got %q and %q", id2, id3)
	}
}
