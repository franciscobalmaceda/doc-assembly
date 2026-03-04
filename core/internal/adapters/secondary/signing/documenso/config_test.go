package documenso

import "testing"

func TestNormalizeDocumensoAPIBaseURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "adds api v2 when base url has no api segment",
			input: "https://sign.tether.education",
			want:  "https://sign.tether.education/api/v2",
		},
		{
			name:  "keeps api v2 when already present",
			input: "https://sign.tether.education/api/v2",
			want:  "https://sign.tether.education/api/v2",
		},
		{
			name:  "keeps api v1 when already present",
			input: "https://sign.tether.education/api/v1",
			want:  "https://sign.tether.education/api/v1",
		},
		{
			name:  "adds api v2 when custom path has no api segment",
			input: "https://sign.tether.education/documenso",
			want:  "https://sign.tether.education/documenso/api/v2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeDocumensoAPIBaseURL(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeDocumensoAPIBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConfigValidate_NormalizesBaseAndSigningURL(t *testing.T) {
	cfg := &Config{
		APIKey:  "api_test_key",
		BaseURL: "https://sign.tether.education",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if cfg.BaseURL != "https://sign.tether.education/api/v2" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "https://sign.tether.education/api/v2")
	}

	if cfg.SigningBaseURL != "https://sign.tether.education" {
		t.Fatalf("SigningBaseURL = %q, want %q", cfg.SigningBaseURL, "https://sign.tether.education")
	}
}
