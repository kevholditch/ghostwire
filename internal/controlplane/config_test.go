package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnvRequiresAPIToken(t *testing.T) {
	t.Setenv("GHOSTWIRE_ENROLLMENT_TOKEN", "old-secret")

	_, err := ConfigFromEnv()

	require.ErrorContains(t, err, "GHOSTWIRE_API_TOKEN is required")
}

func TestConfigFromEnvLoadsAPIToken(t *testing.T) {
	t.Setenv("GHOSTWIRE_API_TOKEN", "api-secret")

	cfg, err := ConfigFromEnv()

	require.NoError(t, err)
	require.Equal(t, "api-secret", cfg.APIToken)
}
