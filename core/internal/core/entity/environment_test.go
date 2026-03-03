package entity

import (
	"testing"
)

func TestParseEnvironment_DefaultAliases(t *testing.T) {
	tests := []struct {
		input string
		want  Environment
	}{
		{"dev", EnvironmentDev},
		{"prod", EnvironmentProd},
		{"DEV", EnvironmentDev},
		{"PROD", EnvironmentProd},
		{"development", EnvironmentDev},
		{"production", EnvironmentProd},
		{"staging", EnvironmentDev},
		{"uat", EnvironmentDev},
		{"qa", EnvironmentDev},
		{"local", EnvironmentDev},
		{"sandbox", EnvironmentDev},
		{"develop", EnvironmentDev},
	}
	for _, tt := range tests {
		got, err := ParseEnvironment(tt.input)
		if err != nil {
			t.Errorf("ParseEnvironment(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("ParseEnvironment(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseEnvironment_Invalid(t *testing.T) {
	invalid := []string{"", "test", "unknown"}
	for _, s := range invalid {
		_, err := ParseEnvironment(s)
		if err == nil {
			t.Errorf("ParseEnvironment(%q) expected error, got nil", s)
		}
	}
}

func TestInitEnvironmentAliases_Custom(t *testing.T) {
	// Save original and restore after test
	envMu.RLock()
	original := envAliasMap
	envMu.RUnlock()
	defer func() {
		envMu.Lock()
		envAliasMap = original
		envMu.Unlock()
	}()

	InitEnvironmentAliases(map[string][]string{
		"dev":  {"dev", "custom-dev"},
		"prod": {"prod", "live"},
	})

	if env, err := ParseEnvironment("custom-dev"); err != nil || env != EnvironmentDev {
		t.Errorf("ParseEnvironment(\"custom-dev\") = %q, %v; want dev, nil", env, err)
	}
	if env, err := ParseEnvironment("live"); err != nil || env != EnvironmentProd {
		t.Errorf("ParseEnvironment(\"live\") = %q, %v; want prod, nil", env, err)
	}
	// "staging" should no longer be valid after custom init
	if _, err := ParseEnvironment("staging"); err == nil {
		t.Error("ParseEnvironment(\"staging\") expected error after custom init, got nil")
	}
}

func TestInitEnvironmentAliases_Empty(t *testing.T) {
	// Empty map should not change aliases
	envMu.RLock()
	original := envAliasMap
	envMu.RUnlock()

	InitEnvironmentAliases(nil)
	InitEnvironmentAliases(map[string][]string{})

	envMu.RLock()
	current := envAliasMap
	envMu.RUnlock()

	if len(current) != len(original) {
		t.Error("InitEnvironmentAliases with empty/nil should not change aliases")
	}
}

func TestEnvironmentFromSandbox(t *testing.T) {
	if got := EnvironmentFromSandbox(true); got != EnvironmentDev {
		t.Errorf("EnvironmentFromSandbox(true) = %q, want %q", got, EnvironmentDev)
	}
	if got := EnvironmentFromSandbox(false); got != EnvironmentProd {
		t.Errorf("EnvironmentFromSandbox(false) = %q, want %q", got, EnvironmentProd)
	}
}
