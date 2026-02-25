package entity

import (
	"testing"
)

func TestParseEnvironment_Valid(t *testing.T) {
	tests := []struct {
		input string
		want  Environment
	}{
		{"dev", EnvironmentDev},
		{"prod", EnvironmentProd},
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
	invalid := []string{"", "staging", "DEV", "PROD", "development", "production"}
	for _, s := range invalid {
		_, err := ParseEnvironment(s)
		if err == nil {
			t.Errorf("ParseEnvironment(%q) expected error, got nil", s)
		}
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
