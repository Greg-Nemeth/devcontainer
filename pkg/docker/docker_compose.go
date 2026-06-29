package docker

import (
	"bufio"
	"strings"
)

type ServiceBuildInfo struct {
	Image      string
	Context    string
	Dockerfile string
	Args       map[string]string
}

func ParseComposeConfig(yamlContent string, targetService string) (ServiceBuildInfo, error) {
	var info ServiceBuildInfo
	info.Args = make(map[string]string)

	scanner := bufio.NewScanner(strings.NewReader(yamlContent))

	servicesIndent := -1
	serviceIndent := -1
	buildIndent := -1
	argsIndent := -1

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Calculate indentation level (leading spaces)
		indent := len(line) - len(strings.TrimLeft(line, " "))

		// Pop states depending on indentation level
		if argsIndent != -1 && indent <= argsIndent {
			argsIndent = -1
		}
		if buildIndent != -1 && indent <= buildIndent {
			buildIndent = -1
		}
		if serviceIndent != -1 && indent <= serviceIndent {
			serviceIndent = -1
		}
		if servicesIndent != -1 && indent <= servicesIndent {
			servicesIndent = -1
		}

		// Process blocks
		if argsIndent != -1 {
			// Parse argument
			if strings.HasPrefix(trimmed, "- ") {
				// List form: - KEY=VALUE or - KEY
				val := strings.TrimPrefix(trimmed, "- ")
				idx := strings.Index(val, "=")
				if idx != -1 {
					k := strings.TrimSpace(val[:idx])
					v := strings.Trim(strings.TrimSpace(val[idx+1:]), `"'`)
					info.Args[k] = v
				} else {
					info.Args[strings.TrimSpace(val)] = ""
				}
			} else {
				// Map form: KEY: VALUE
				idx := strings.Index(trimmed, ":")
				if idx != -1 {
					k := strings.TrimSpace(trimmed[:idx])
					v := strings.Trim(strings.TrimSpace(trimmed[idx+1:]), `"'`)
					info.Args[k] = v
				}
			}
		} else if buildIndent != -1 {
			if strings.HasPrefix(trimmed, "context:") {
				info.Context = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "context:")), `"'`)
			} else if strings.HasPrefix(trimmed, "dockerfile:") {
				info.Dockerfile = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "dockerfile:")), `"'`)
			} else if strings.HasPrefix(trimmed, "args:") {
				argsIndent = indent
			}
		} else if serviceIndent != -1 {
			if strings.HasPrefix(trimmed, "image:") {
				info.Image = strings.Trim(strings.TrimSpace(strings.TrimPrefix(trimmed, "image:")), `"'`)
			} else if strings.HasPrefix(trimmed, "build:") {
				buildIndent = indent
				// Handle short-form build context if build value is a string (e.g. build: ./context)
				buildVal := strings.TrimSpace(strings.TrimPrefix(trimmed, "build:"))
				if buildVal != "" {
					info.Context = strings.Trim(buildVal, `"'`)
				}
			}
		} else if servicesIndent != -1 {
			if strings.HasPrefix(trimmed, targetService+":") {
				serviceIndent = indent
			}
		} else {
			if strings.HasPrefix(trimmed, "services:") {
				servicesIndent = indent
			}
		}
	}

	return info, nil
}
