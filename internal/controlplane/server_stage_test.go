package controlplane

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
	"github.com/stretchr/testify/require"
)

type serverStageState struct {
	t          *testing.T
	assertions *require.Assertions
	server     *Server

	response     *httptest.ResponseRecorder
	enroll       protocol.EnrollResponse
	peers        protocol.PeersResponse
	nodes        protocol.NodesResponse
	decodeErr    error
	enrollStatus int
}

type serverGiven struct {
	state *serverStageState
}

type serverWhen struct {
	state *serverStageState
}

type serverThen struct {
	state *serverStageState
}

func NewServerStage(t *testing.T) (*serverGiven, *serverWhen, *serverThen) {
	t.Helper()
	state := &serverStageState{
		t:          t,
		assertions: require.New(t),
		server:     newTestServer(t),
	}
	return &serverGiven{state: state}, &serverWhen{state: state}, &serverThen{state: state}
}

func (g *serverGiven) there_is_a_control_plane() *serverGiven {
	g.state.t.Helper()
	return g
}

func (g *serverGiven) there_is_an_agent_that_exists_with_hostname_agent_a() *serverGiven {
	g.state.t.Helper()
	resp := postJSON(g.state.t, g.state.server, "/v1/agents/enroll", protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "agent-a",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.11:51820",
		EnrollmentToken:    "secret",
	})
	g.state.assertions.Equal(http.StatusOK, resp.Code)
	return g
}

func (g *serverGiven) there_is_an_agent_that_exists_with_hostname_bravo() *serverGiven {
	g.state.t.Helper()
	resp := postJSON(g.state.t, g.state.server, "/v1/agents/enroll", protocol.EnrollRequest{
		AgentID:            "agent-b",
		Hostname:           "bravo",
		WireGuardPublicKey: "pub-b",
		Endpoint:           "172.28.0.12:51820",
		EnrollmentToken:    "secret",
	})
	g.state.assertions.Equal(http.StatusOK, resp.Code)
	return g
}

func (g *serverGiven) and() *serverGiven {
	g.state.t.Helper()
	return g
}

func (w *serverWhen) the_health_endpoint_is_requested() {
	w.state.t.Helper()
	w.get("/healthz")
}

func (w *serverWhen) agent_a_enrolls() {
	w.state.t.Helper()
	resp := postJSON(w.state.t, w.state.server, "/v1/agents/enroll", protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.11:51820",
		EnrollmentToken:    "secret",
	})
	w.state.response = resp
	w.state.enrollStatus = resp.Code
	if resp.Code == http.StatusOK {
		w.state.decodeErr = json.NewDecoder(resp.Body).Decode(&w.state.enroll)
	}
}

func (w *serverWhen) agent_a_enrolls_with_an_invalid_token() {
	w.state.t.Helper()
	w.state.response = postJSON(w.state.t, w.state.server, "/v1/agents/enroll", protocol.EnrollRequest{
		AgentID:         "agent-a",
		EnrollmentToken: "wrong",
	})
}

func (w *serverWhen) malformed_json_is_posted_to_the_enroll_endpoint() {
	w.state.t.Helper()
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/agents/enroll", bytes.NewBufferString("{"))
	req.Header.Set("content-type", "application/json")
	w.state.server.ServeHTTP(resp, req)
	w.state.response = resp
}

func (w *serverWhen) agent_a_sends_a_heartbeat_with_updated_metadata() {
	w.state.t.Helper()
	w.state.response = postJSON(w.state.t, w.state.server, "/v1/agents/heartbeat", protocol.HeartbeatRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha-new",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.111:51820",
		EnrollmentToken:    "secret",
	})
}

func (w *serverWhen) agent_a_requests_its_peers() {
	w.state.t.Helper()
	w.get("/v1/agents/agent-a/peers")
	if w.state.response.Code == http.StatusOK {
		w.state.decodeErr = json.NewDecoder(w.state.response.Body).Decode(&w.state.peers)
	}
}

func (w *serverWhen) peers_are_requested_for_a_missing_agent() {
	w.state.t.Helper()
	w.get("/v1/agents/missing/peers")
}

func (w *serverWhen) the_nodes_endpoint_is_requested() {
	w.state.t.Helper()
	w.get("/v1/nodes")
	if w.state.response.Code == http.StatusOK {
		w.state.decodeErr = json.NewDecoder(w.state.response.Body).Decode(&w.state.nodes)
	}
}

func (w *serverWhen) the_nodes_endpoint_is_posted_to() {
	w.state.t.Helper()
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/nodes", nil)
	w.state.server.ServeHTTP(resp, req)
	w.state.response = resp
}

func (th *serverThen) the_request_succeeds() *serverThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.StatusOK, th.state.response.Code)
	return th
}

func (th *serverThen) agent_a_is_given_the_first_ghostwire_ip() *serverThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.StatusOK, th.state.enrollStatus)
	th.state.assertions.NoError(th.state.decodeErr)
	th.state.assertions.Equal("10.44.0.1", th.state.enroll.PrivateIP)
	return th
}

func (th *serverThen) the_heartbeat_is_accepted() *serverThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.StatusNoContent, th.state.response.Code)
	return th
}

func (th *serverThen) only_agent_b_is_returned_as_a_peer() *serverThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.StatusOK, th.state.response.Code)
	th.state.assertions.NoError(th.state.decodeErr)
	th.state.assertions.Len(th.state.peers.Peers, 1)
	th.state.assertions.Equal("agent-b", th.state.peers.Peers[0].AgentID)
	return th
}

func (th *serverThen) the_request_is_rejected_as_unauthorized() *serverThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.StatusUnauthorized, th.state.response.Code)
	return th
}

func (th *serverThen) the_request_is_rejected_as_bad_request() *serverThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.StatusBadRequest, th.state.response.Code)
	return th
}

func (th *serverThen) the_request_is_rejected_as_not_found() *serverThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.StatusNotFound, th.state.response.Code)
	return th
}

func (th *serverThen) agent_a_is_listed_with_its_node_metadata() *serverThen {
	th.state.t.Helper()
	th.state.assertions.NoError(th.state.decodeErr)
	th.state.assertions.Equal([]protocol.Node{
		{
			NodeID:             "agent-a",
			Hostname:           "agent-a",
			WireGuardPublicKey: "pub-a",
			GhostwireIP:        "10.44.0.1",
			LastSeen:           time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		},
	}, th.state.nodes.Nodes)
	return th
}

func (th *serverThen) the_request_is_rejected_because_the_method_is_not_allowed() *serverThen {
	th.state.t.Helper()
	th.state.assertions.Equal(http.StatusMethodNotAllowed, th.state.response.Code)
	return th
}

func (th *serverThen) and() *serverThen {
	th.state.t.Helper()
	return th
}

func (w *serverWhen) get(path string) {
	w.state.t.Helper()
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w.state.server.ServeHTTP(resp, req)
	w.state.response = resp
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	assertions := require.New(t)
	ipam, err := NewIPAM("10.44.0.0/29")
	assertions.NoError(err)
	registry := NewRegistry(RegistryConfig{
		EnrollmentToken:   "secret",
		NetworkCIDR:       "10.44.0.0/29",
		HeartbeatInterval: time.Second,
		PollInterval:      time.Second,
		AgentTTL:          time.Minute,
	}, ipam)
	return NewServer(registry, func() time.Time { return time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC) })
}

func postJSON(t *testing.T, server http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	buf := bytes.Buffer{}
	require.NoError(t, json.NewEncoder(&buf).Encode(body))
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("content-type", "application/json")
	server.ServeHTTP(resp, req)
	return resp
}
