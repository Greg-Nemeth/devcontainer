package varsub

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type SubstitutionContext struct {
	Platform                 string
	ConfigFile               string
	LocalWorkspaceFolder     string
	ContainerWorkspaceFolder string
	Env                      map[string]string
}

var varRegex = regexp.MustCompile(`\$\{(.*?)\}`)

func Substitute(context SubstitutionContext, value interface{}) interface{} {
	isWindows := context.Platform == "win32"
	normalizedEnv := normalizeEnv(isWindows, context.Env)

	// Pre-resolve containerWorkspaceFolder if it has variables
	if context.ContainerWorkspaceFolder != "" {
		tempContext := context
		tempContext.ContainerWorkspaceFolder = ""
		context.ContainerWorkspaceFolder = substituteString(func(match string, variable string, args []string) string {
			return replaceWithContext(isWindows, tempContext, match, variable, args)
		}, context.ContainerWorkspaceFolder)
	}

	replaceFunc := func(match string, variable string, args []string) string {
		ctx := context
		ctx.Env = normalizedEnv
		return replaceWithContext(isWindows, ctx, match, variable, args)
	}

	return substituteRecursive(replaceFunc, value)
}

func ContainerSubstitute(platform string, configFile string, containerEnv map[string]string, value interface{}) interface{} {
	isWindows := platform == "win32"
	normalizedEnv := normalizeEnv(isWindows, containerEnv)

	replaceFunc := func(match string, variable string, args []string) string {
		if variable == "containerEnv" {
			return lookupValue(isWindows, normalizedEnv, args, match, configFile)
		}
		return match
	}

	return substituteRecursive(replaceFunc, value)
}

func BeforeContainerSubstitute(idLabels map[string]string, value interface{}) interface{} {
	var devcontainerId string
	replaceFunc := func(match string, variable string, args []string) string {
		if variable == "devcontainerId" {
			if devcontainerId == "" {
				devcontainerId = devcontainerIdForLabels(idLabels)
			}
			return devcontainerId
		}
		return match
	}

	return substituteRecursive(replaceFunc, value)
}

func normalizeEnv(isWindows bool, originalEnv map[string]string) map[string]string {
	if isWindows {
		env := make(map[string]string)
		for k, v := range originalEnv {
			env[strings.ToLower(k)] = v
		}
		return env
	}
	return originalEnv
}

func substituteRecursive(replace func(string, string, []string) string, value interface{}) interface{} {
	switch val := value.(type) {
	case string:
		return substituteString(replace, val)
	case []interface{}:
		res := make([]interface{}, len(val))
		for i, v := range val {
			res[i] = substituteRecursive(replace, v)
		}
		return res
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, v := range val {
			res[k] = substituteRecursive(replace, v)
		}
		return res
	default:
		return val
	}
}

func substituteString(replace func(string, string, []string) string, value string) string {
	// Replaces all matches in the string
	res := varRegex.ReplaceAllStringFunc(value, func(match string) string {
		inner := match[2 : len(match)-1]
		parts := strings.Split(inner, ":")
		variable := parts[0]
		var args []string
		if len(parts) > 1 {
			args = parts[1:]
		}
		return replace(match, variable, args)
	})
	return res
}

func getBasename(path string, isWindows bool) string {
	if isWindows {
		// split by both \ and / for safety
		path = strings.ReplaceAll(path, "/", "\\")
		idx := strings.LastIndex(path, "\\")
		if idx == -1 {
			return path
		}
		return path[idx+1:]
	}
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

func replaceWithContext(isWindows bool, context SubstitutionContext, match string, variable string, args []string) string {
	switch variable {
	case "env", "localEnv":
		return lookupValue(isWindows, context.Env, args, match, context.ConfigFile)
	case "localWorkspaceFolder":
		if context.LocalWorkspaceFolder != "" {
			return context.LocalWorkspaceFolder
		}
		return match
	case "localWorkspaceFolderBasename":
		if context.LocalWorkspaceFolder != "" {
			return getBasename(context.LocalWorkspaceFolder, isWindows)
		}
		return match
	case "containerWorkspaceFolder":
		if context.ContainerWorkspaceFolder != "" {
			return context.ContainerWorkspaceFolder
		}
		return match
	case "containerWorkspaceFolderBasename":
		if context.ContainerWorkspaceFolder != "" {
			return getBasename(context.ContainerWorkspaceFolder, false) // Container is posix
		}
		return match
	default:
		return match
	}
}

func lookupValue(isWindows bool, envObj map[string]string, args []string, match string, configFile string) string {
	if len(args) > 0 {
		name := args[0]
		if isWindows {
			name = strings.ToLower(name)
		}
		val, ok := envObj[name]
		if ok {
			return val
		}
		if len(args) > 1 {
			return args[1]
		}
		return ""
	}
	// Mimics ContainerError on missing argument for env lookup
	panic(fmt.Sprintf("'%s'%s can not be resolved because no environment variable name is given.", match, func() string {
		if configFile != "" {
			return " in " + filepath.Base(configFile)
		}
		return ""
	}()))
}

func devcontainerIdForLabels(idLabels map[string]string) string {
	if idLabels == nil {
		idLabels = make(map[string]string)
	}
	
	// Double check sorting to be absolutely sure
	keys := make([]string, 0, len(idLabels))
	for k := range idLabels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	// Build sorted json
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%q:%q", k, idLabels[k]))
	}
	stringInput := "{" + strings.Join(parts, ",") + "}"
	
	hash := sha256.Sum256([]byte(stringInput))
	val := new(big.Int).SetBytes(hash[:])
	id := val.Text(32)
	if len(id) < 52 {
		id = strings.Repeat("0", 52-len(id)) + id
	}
	return id
}
