package controlapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
	"github.com/stretchr/testify/require"
)

type clientStageState struct {
	t          *testing.T
	assertions *require.Assertions

	server *httptest.Server
	client *Client

	nodes protocol.NodesResponse
	err   error

	requestMethod string
	requestPath   string
	requestAuth   string
}

type clientGiven struct {
	state *clientStageState
}

type clientWhen struct {
	state *clientStageState
}

type clientThen struct {
	state *clientStageState
}

func NewClientStage(t *testing.T) (*clientGiven, *clientWhen, *clientThen) {
	t.Helper()
	state := &clientStageState{
		t:          t,
		assertions: require.New(t),
	}
	t.Cleanup(func() {
		if state.server != nil {
			state.server.Close()
		}
	})
	return &clientGiven{state: state}, &clientWhen{state: state}, &clientThen{state: state}
}

func (g *clientGiven) there_is_a_control_plane_with_operator_nodes() *clientGiven {
	g.state.t.Helper()
	g.state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.recordRequest(r)
		writeClientStageJSON(g.state.t, w, http.StatusOK, protocol.NodesResponse{Nodes: []protocol.Node{
			{
				NodeID:             "agent-a",
				Hostname:           "agent-a",
				WireGuardPublicKey: "pub-a",
				GhostwireIP:        "10.44.0.1",
				Endpoint:           "172.28.0.11:51820",
				LastSeen:           time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
				Status:             protocol.NodeStatusOnline,
			},
		}})
	}))
	g.state.client = NewClient(g.state.server.URL, "api-secret")
	return g
}

func (g *clientGiven) there_is_a_control_plane_that_rejects_requests_as_unauthorized() *clientGiven {
	g.state.t.Helper()
	g.state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.recordRequest(r)
		writeClientStageJSON(g.state.t, w, http.StatusUnauthorized, protocol.ErrorResponse{
			Code:    "unauthorized",
			Message: "missing or invalid API token",
		})
	}))
	g.state.client = NewClient(g.state.server.URL, "wrong")
	return g
}

func (g *clientGiven) there_is_a_control_plane_that_records_peer_requests() *clientGiven {
	g.state.t.Helper()
	g.state.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.recordRequest(r)
		writeClientStageJSON(g.state.t, w, http.StatusOK, protocol.NodesResponse{})
	}))
	g.state.client = NewClient(g.state.server.URL, "api-secret")
	return g
}

func (g *clientGiven) recordRequest(r *http.Request) {
	g.state.t.Helper()
	g.state.requestMethod = r.Method
	g.state.requestPath = r.URL.EscapedPath()
	g.state.requestAuth = r.Header.Get("authorization")
}

func (g *clientGiven) and() *clientGiven {
	g.state.t.Helper()
	return g
}

func (w *clientWhen) nodes_are_listed() {
	w.state.t.Helper()
	w.state.nodes, w.state.err = w.state.client.ListNodes(context.Background())
}

func (w *clientWhen) node_agent_a_is_requested() {
	w.state.t.Helper()
	_, w.state.err = w.state.client.GetNode(context.Background(), "agent-a")
}

func (w *clientWhen) peers_are_listed_for_node_id_containing_a_slash() {
	w.state.t.Helper()
	w.state.nodes, w.state.err = w.state.client.ListNodePeers(context.Background(), "agent/a")
}

func (th *clientThen) the_request_succeeds() *clientThen {
	th.state.t.Helper()
	th.state.assertions.NoError(th.state.err)
	return th
}

func (th *clientThen) the_bearer_token_was_sent() *clientThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.MethodGet, th.state.requestMethod)
	th.state.assertions.Equal("/v1/nodes", th.state.requestPath)
	th.state.assertions.Equal("Bearer api-secret", th.state.requestAuth)
	return th
}

func (th *clientThen) agent_a_is_returned() *clientThen {
	th.state.t.Helper()
	th.state.assertions.Len(th.state.nodes.Nodes, 1)
	th.state.assertions.Equal("agent-a", th.state.nodes.Nodes[0].NodeID)
	return th
}

func (th *clientThen) the_request_fails_with_unauthorized_api_error() *clientThen {
	th.state.t.Helper()
	var apiErr *APIError
	th.state.assertions.True(errors.As(th.state.err, &apiErr))
	th.state.assertions.Equal(http.StatusUnauthorized, apiErr.StatusCode)
	th.state.assertions.Equal("unauthorized", apiErr.Code)
	th.state.assertions.Equal("missing or invalid API token", apiErr.Message)
	th.state.assertions.Equal("unauthorized: missing or invalid API token", th.state.err.Error())
	return th
}

func (th *clientThen) the_node_id_path_segment_was_escaped() *clientThen {
	th.state.t.Helper()
	th.state.assertions.Equal("/v1/nodes/agent%2Fa/peers", th.state.requestPath)
	return th
}

func (th *clientThen) and() *clientThen {
	th.state.t.Helper()
	return th
}

func writeClientStageJSON(t *testing.T, w http.ResponseWriter, status int, value any) {
	t.Helper()
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	require.NoError(t, json.NewEncoder(w).Encode(value))
}
