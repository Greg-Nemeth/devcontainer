package docker

import (
	"fmt"
	"regexp"
	"strings"
)

type FeatureInstallOptions struct {
	ID          string
	UnpackedDir string
	Options     map[string]interface{}
}

type BuildOptions struct {
	BaseImage     string
	TargetImage   string
	Features      []FeatureInstallOptions
	MetadataLabel string
}

func GetSafeID(str string) string {
	// Replaces [^\w_] with _
	reg1 := regexp.MustCompile(`[^\w_]`)
	res := reg1.ReplaceAllString(str, "_")

	// Replaces leading digits/underscores with _
	reg2 := regexp.MustCompile(`^[\d_]+`)
	res = reg2.ReplaceAllString(res, "_")

	return strings.ToUpper(res)
}

func GenerateDockerfile(opts BuildOptions) (string, error) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("FROM %s\n\n", opts.BaseImage))

	// For each feature:
	// 1. Copy its unpacked directory to a temporary build container path
	// 2. Set environment variables from options
	// 3. Execute install.sh
	for _, feat := range opts.Features {
		safeID := GetSafeID(feat.ID)
		destPath := fmt.Sprintf("/tmp/features/%s", safeID)

		builder.WriteString(fmt.Sprintf("COPY features/%s %s\n", safeID, destPath))

		// Inject options as env variables
		var envLines []string
		for k, v := range feat.Options {
			// Convert options keys to uppercase safe environment variables
			envKey := strings.ToUpper(k)
			envVal := fmt.Sprintf("%v", v)
			envLines = append(envLines, fmt.Sprintf("%s=%q", envKey, envVal))
		}

		envStr := ""
		if len(envLines) > 0 {
			envStr = strings.Join(envLines, " ") + " "
		}

		builder.WriteString(fmt.Sprintf("RUN cd %s && chmod +x install.sh && %s./install.sh\n\n", destPath, envStr))
	}

	if opts.MetadataLabel != "" {
		escapedMetadata := strings.ReplaceAll(opts.MetadataLabel, `"`, `\"`)
		builder.WriteString(fmt.Sprintf("LABEL devcontainer.metadata=\"%s\"\n", escapedMetadata))
	}

	return builder.String(), nil
}

func (c *CLI) BuildImage(dockerfilePath, contextDir string, opts BuildOptions) error {
	args := []string{"build", "-f", dockerfilePath, "-t", opts.TargetImage, contextDir}
	output, err := c.runner(c.CLIPath, args...)
	if err != nil {
		return fmt.Errorf("build failed: %w (output: %s)", err, string(output))
	}
	return nil
}
