package entity

import (
	"fmt"
	"strings"
	"sync"
)

// Environment indicates whether the request targets a dev (sandbox) or prod workspace.
type Environment string

const (
	// EnvironmentProd targets the production workspace.
	EnvironmentProd Environment = "prod"
	// EnvironmentDev targets the sandbox (dev) workspace.
	EnvironmentDev Environment = "dev"
)

// defaultEnvironmentAliases are used when no config is provided.
var defaultEnvironmentAliases = map[string][]string{
	"dev":  {"dev", "develop", "development", "staging", "uat", "qa", "local", "sandbox"},
	"prod": {"prod", "production"},
}

var (
	envAliasMap map[string]Environment
	envMu       sync.RWMutex
)

func init() {
	envAliasMap = buildAliasMap(defaultEnvironmentAliases)
}

// InitEnvironmentAliases replaces the alias registry from configuration.
// Keys must be "dev" or "prod"; values are the accepted aliases for each.
func InitEnvironmentAliases(aliases map[string][]string) {
	if len(aliases) == 0 {
		return
	}
	envMu.Lock()
	defer envMu.Unlock()
	envAliasMap = buildAliasMap(aliases)
}

func buildAliasMap(aliases map[string][]string) map[string]Environment {
	m := make(map[string]Environment)
	targets := map[string]Environment{
		"dev":  EnvironmentDev,
		"prod": EnvironmentProd,
	}
	for key, names := range aliases {
		env, ok := targets[strings.ToLower(strings.TrimSpace(key))]
		if !ok {
			continue
		}
		for _, name := range names {
			n := strings.ToLower(strings.TrimSpace(name))
			if n != "" {
				m[n] = env
			}
		}
	}
	return m
}

// ParseEnvironment parses a string into an Environment value using the alias registry.
func ParseEnvironment(s string) (Environment, error) {
	key := strings.ToLower(strings.TrimSpace(s))
	envMu.RLock()
	env, ok := envAliasMap[key]
	envMu.RUnlock()
	if ok {
		return env, nil
	}
	return "", fmt.Errorf("invalid environment value: %q (must be %q or %q)", s, EnvironmentDev, EnvironmentProd)
}

// EnvironmentFromSandbox derives the Environment from the workspace sandbox flag.
func EnvironmentFromSandbox(isSandbox bool) Environment {
	if isSandbox {
		return EnvironmentDev
	}
	return EnvironmentProd
}
