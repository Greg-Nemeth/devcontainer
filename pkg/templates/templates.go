package templates

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type TemplateRef struct {
	Registry  string
	Namespace string
	ID        string
	Version   string
}

func ParseTemplateRef(ref string) (TemplateRef, error) {
	// Format should be registry/namespace/id:version
	firstSlash := strings.Index(ref, "/")
	lastColon := strings.LastIndex(ref, ":")
	lastSlash := strings.LastIndex(ref, "/")

	if firstSlash == -1 || lastColon == -1 || lastSlash == -1 || lastSlash <= firstSlash || lastColon <= lastSlash {
		return TemplateRef{}, fmt.Errorf("invalid template reference format: %s", ref)
	}

	registry := ref[:firstSlash]
	version := ref[lastColon+1:]
	id := ref[lastSlash+1 : lastColon]
	namespace := ref[firstSlash+1 : lastSlash]

	return TemplateRef{
		Registry:  registry,
		Namespace: namespace,
		ID:        id,
		Version:   version,
	}, nil
}

func ApplyTemplate(srcDir, dstDir string, options map[string]string) error {
	re := regexp.MustCompile(`\$\{templateOption:([^}]+)\}`)

	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		if rel == "." {
			return nil
		}

		target := filepath.Join(dstDir, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		// Read file content
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Perform variable substitution
		content := string(data)
		substituted := re.ReplaceAllStringFunc(content, func(match string) string {
			m := re.FindStringSubmatch(match)
			if len(m) > 1 {
				optName := m[1]
				if val, ok := options[optName]; ok {
					return val
				}
			}
			return ""
		})

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		return os.WriteFile(target, []byte(substituted), info.Mode())
	})

	return err
}
