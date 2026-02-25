package entity

import "fmt"

// Environment indicates whether the request targets a dev (sandbox) or prod workspace.
type Environment string

const (
	// EnvironmentProd targets the production workspace.
	EnvironmentProd Environment = "prod"
	// EnvironmentDev targets the sandbox (dev) workspace.
	EnvironmentDev Environment = "dev"
)

// ParseEnvironment parses a string into an Environment value.
// Returns an error if the value is not "dev" or "prod".
func ParseEnvironment(s string) (Environment, error) {
	switch Environment(s) {
	case EnvironmentProd:
		return EnvironmentProd, nil
	case EnvironmentDev:
		return EnvironmentDev, nil
	default:
		return "", fmt.Errorf("invalid environment value: %q (must be %q or %q)", s, EnvironmentDev, EnvironmentProd)
	}
}

// EnvironmentFromSandbox derives the Environment from the workspace sandbox flag.
func EnvironmentFromSandbox(isSandbox bool) Environment {
	if isSandbox {
		return EnvironmentDev
	}
	return EnvironmentProd
}
