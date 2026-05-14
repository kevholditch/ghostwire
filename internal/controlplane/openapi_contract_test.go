package controlplane

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAPIV1ContractDocumentsOperatorAndAgentAuth(t *testing.T) {
	raw, err := os.ReadFile("../../docs/openapi/v1.json")
	require.NoError(t, err)

	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))

	paths := doc["paths"].(map[string]any)
	for _, path := range []string{
		"/v1/agents/enroll",
		"/v1/agents/heartbeat",
		"/v1/agents/{agent_id}/peers",
		"/v1/nodes",
		"/v1/nodes/{node_id}",
		"/v1/nodes/{node_id}/peers",
	} {
		require.Contains(t, paths, path)
	}

	components := doc["components"].(map[string]any)
	securitySchemes := components["securitySchemes"].(map[string]any)
	require.Contains(t, securitySchemes, "apiToken")

	schemas := components["schemas"].(map[string]any)
	errorResponse := schemas["ErrorResponse"].(map[string]any)
	properties := errorResponse["properties"].(map[string]any)
	require.Contains(t, properties, "code")
	require.Contains(t, properties, "message")
}
