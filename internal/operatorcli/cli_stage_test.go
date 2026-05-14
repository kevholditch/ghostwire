package operatorcli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
	"github.com/stretchr/testify/require"
)

type cliStageState struct {
	t          *testing.T
	assertions *require.Assertions

	server *httptest.Server
	stdout strings.Builder
	stderr strings.Builder
	code   int
}

type cliGiven struct {
	state *cliStageState
}

type cliWhen struct {
	state *cliStageState
}

type cliThen struct {
	state *cliStageState
}

func NewCLIStage(t *testing.T) (*cliGiven, *cliWhen, *cliThen) {
	t.Helper()
	state := &cliStageState{
		t:          t,
		assertions: require.New(t),
	}
	t.Cleanup(func() {
		if state.server != nil {
			state.server.Close()
		}
	})
	return &cliGiven{state: state}, &cliWhen{state: state}, &cliThen{state: state}
}

func (g *cliGiven) there_is_a_control_plane_with_operator_nodes() *cliGiven {
	g.state.t.Helper()
	g.state.server = operatorCLITestServer(g.state.t)
	return g
}

func (g *cliGiven) and() *cliGiven {
	g.state.t.Helper()
	return g
}

func (w *cliWhen) nodes_are_listed_using_environment_config() {
	w.state.t.Helper()
	w.run([]string{"nodes", "list"}, envFor(w.state.server.URL, "api-secret"))
}

func (w *cliWhen) node_agent_a_is_requested_as_json_using_flags() {
	w.state.t.Helper()
	w.run([]string{"--control-url", w.state.server.URL, "--api-token", "api-secret", "--output", "json", "nodes", "get", "agent-a"}, envFor("", ""))
}

func (w *cliWhen) agent_a_peers_are_listed_using_environment_config() {
	w.state.t.Helper()
	w.run([]string{"nodes", "peers", "agent-a"}, envFor(w.state.server.URL, "api-secret"))
}

func (w *cliWhen) nodes_are_listed_with_an_invalid_api_token() {
	w.state.t.Helper()
	w.run([]string{"nodes", "list"}, envFor(w.state.server.URL, "wrong"))
}

func (w *cliWhen) run(args []string, getenv func(string) string) {
	w.state.t.Helper()
	w.state.code = Run(context.Background(), args, getenv, &w.state.stdout, &w.state.stderr)
}

func (th *cliThen) the_command_succeeds() *cliThen {
	th.state.t.Helper()
	th.state.assertions.Equal(0, th.state.code)
	th.state.assertions.Empty(th.state.stderr.String())
	return th
}

func (th *cliThen) the_command_fails() *cliThen {
	th.state.t.Helper()
	th.state.assertions.Equal(1, th.state.code)
	th.state.assertions.Empty(th.state.stdout.String())
	return th
}

func (th *cliThen) the_nodes_table_is_rendered() *cliThen {
	th.state.t.Helper()
	out := th.state.stdout.String()
	th.state.assertions.Contains(out, "NODE ID")
	th.state.assertions.Contains(out, "agent-a")
	th.state.assertions.Contains(out, "10.44.0.1")
	th.state.assertions.Contains(out, "online")
	th.state.assertions.Contains(out, "ago")
	return th
}

func (th *cliThen) agent_a_json_is_rendered() *cliThen {
	th.state.t.Helper()
	var node protocol.Node
	th.state.assertions.NoError(json.Unmarshal([]byte(th.state.stdout.String()), &node))
	th.state.assertions.Equal("agent-a", node.NodeID)
	th.state.assertions.Equal("10.44.0.1", node.GhostwireIP)
	return th
}

func (th *cliThen) only_agent_b_is_rendered_as_a_peer() *cliThen {
	th.state.t.Helper()
	out := th.state.stdout.String()
	th.state.assertions.Contains(out, "agent-b")
	th.state.assertions.NotContains(out, "agent-a\t")
	return th
}

func (th *cliThen) the_unauthorized_error_is_rendered() *cliThen {
	th.state.t.Helper()
	th.state.assertions.Contains(th.state.stderr.String(), "unauthorized")
	return th
}

func (th *cliThen) and() *cliThen {
	th.state.t.Helper()
	return th
}

func operatorCLITestServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeOperatorCLITest(t, w, r) {
			return
		}
		writeOperatorCLITestJSON(t, w, protocol.NodesResponse{Nodes: []protocol.Node{
			{
				NodeID:             "agent-a",
				Hostname:           "agent-a",
				WireGuardPublicKey: "pub-a",
				GhostwireIP:        "10.44.0.1",
				Endpoint:           "172.28.0.11:51820",
				LastSeen:           time.Now().Add(-3 * time.Second).UTC(),
				Status:             protocol.NodeStatusOnline,
			},
		}})
	})
	mux.HandleFunc("/v1/nodes/agent-a", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeOperatorCLITest(t, w, r) {
			return
		}
		writeOperatorCLITestJSON(t, w, protocol.Node{
			NodeID:             "agent-a",
			Hostname:           "agent-a",
			WireGuardPublicKey: "pub-a",
			GhostwireIP:        "10.44.0.1",
			Endpoint:           "172.28.0.11:51820",
			LastSeen:           time.Now().Add(-3 * time.Second).UTC(),
			Status:             protocol.NodeStatusOnline,
		})
	})
	mux.HandleFunc("/v1/nodes/agent-a/peers", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeOperatorCLITest(t, w, r) {
			return
		}
		writeOperatorCLITestJSON(t, w, protocol.NodesResponse{Nodes: []protocol.Node{
			{
				NodeID:             "agent-b",
				Hostname:           "agent-b",
				WireGuardPublicKey: "pub-b",
				GhostwireIP:        "10.44.0.2",
				Endpoint:           "172.28.0.12:51820",
				LastSeen:           time.Now().Add(-5 * time.Second).UTC(),
				Status:             protocol.NodeStatusOnline,
			},
		}})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		writeOperatorCLITestJSON(t, w, protocol.ErrorResponse{Code: "unauthorized", Message: "missing or invalid API token"})
	})
	return httptest.NewServer(mux)
}

func authorizeOperatorCLITest(t *testing.T, w http.ResponseWriter, r *http.Request) bool {
	t.Helper()
	if r.Header.Get("authorization") == "Bearer api-secret" {
		return true
	}
	w.WriteHeader(http.StatusUnauthorized)
	writeOperatorCLITestJSON(t, w, protocol.ErrorResponse{Code: "unauthorized", Message: "missing or invalid API token"})
	return false
}

func writeOperatorCLITestJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("content-type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}

func envFor(controlURL, apiToken string) func(string) string {
	return func(key string) string {
		switch key {
		case "GHOSTWIRE_CONTROL_URL":
			return controlURL
		case "GHOSTWIRE_API_TOKEN":
			return apiToken
		default:
			return ""
		}
	}
}
