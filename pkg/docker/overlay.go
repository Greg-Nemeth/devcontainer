package docker

import (
	"fmt"
	"sort"
	"strings"
)

type ComposeOverrideOptions struct {
	Service string
	Image   string
	Mounts  []string
	Env     map[string]string
}

func GenerateComposeOverride(opts ComposeOverrideOptions) (string, error) {
	var sb strings.Builder

	sb.WriteString("version: '3'\n")
	sb.WriteString("services:\n")
	sb.WriteString(fmt.Sprintf("  %s:\n", opts.Service))

	if opts.Image != "" {
		sb.WriteString(fmt.Sprintf("    image: %s\n", opts.Image))
	}

	if len(opts.Mounts) > 0 {
		sb.WriteString("    volumes:\n")
		for _, m := range opts.Mounts {
			sb.WriteString(fmt.Sprintf("      - %s\n", m))
		}
	}

	if len(opts.Env) > 0 {
		sb.WriteString("    environment:\n")

		// Sort keys for deterministic output
		var keys []string
		for k := range opts.Env {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("      - %s=%s\n", k, opts.Env[k]))
		}
	}

	return sb.String(), nil
}
