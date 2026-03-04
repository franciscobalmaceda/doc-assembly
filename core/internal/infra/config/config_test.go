package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCSVList_DedupesAndTrims(t *testing.T) {
	raw := " https://a.example ,https://b.example,https://A.example , ,https://b.example "
	got := parseCSVList(raw)

	assert.Equal(t, []string{"https://a.example", "https://b.example"}, got)
}

func TestApplyServerEnvOverrides_PublicSigningFrameAncestors(t *testing.T) {
	t.Setenv(serverPublicSigningFrameAncestorsEnv, "https://foo.example, https://bar.example/")

	cfg := &ServerConfig{
		PublicSigningFrameAncestors: []string{"https://old.example"},
	}

	applyServerEnvOverrides(cfg)

	assert.Equal(t, []string{"https://foo.example", "https://bar.example/"}, cfg.PublicSigningFrameAncestors)
}

func TestApplyStorageEnvOverrides(t *testing.T) {
	t.Setenv("DOC_ENGINE_STORAGE_PROVIDER", "s3")
	t.Setenv("DOC_ENGINE_STORAGE_LOCAL_DIR", "/tmp/doc-storage")
	t.Setenv("DOC_ENGINE_STORAGE_BUCKET", "bucket-name")
	t.Setenv("DOC_ENGINE_STORAGE_REGION", "us-central1")
	t.Setenv("DOC_ENGINE_STORAGE_ENDPOINT", "https://storage.googleapis.com")
	t.Setenv("DOC_ENGINE_STORAGE_ENABLED", "false")

	cfg := &StorageConfig{
		Enabled:  true,
		Provider: "local",
		LocalDir: "./data/storage",
	}

	applyStorageEnvOverrides(cfg)

	assert.False(t, cfg.Enabled)
	assert.Equal(t, "s3", cfg.Provider)
	assert.Equal(t, "/tmp/doc-storage", cfg.LocalDir)
	assert.Equal(t, "bucket-name", cfg.Bucket)
	assert.Equal(t, "us-central1", cfg.Region)
	assert.Equal(t, "https://storage.googleapis.com", cfg.Endpoint)
}
