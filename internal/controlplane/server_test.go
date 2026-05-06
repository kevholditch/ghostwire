package controlplane

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
)

func TestServerHealthz(t *testing.T) {
	server := newTestServer(t)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.Code)
	}
}

func TestServerEnrollHeartbeatAndPeers(t *testing.T) {
	server := newTestServer(t)

	enrollA := postJSON(t, server, "/v1/agents/enroll", protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.11:51820",
		EnrollmentToken:    "secret",
	})
	if enrollA.Code != http.StatusOK {
		t.Fatalf("enroll agent-a status = %d body=%s", enrollA.Code, enrollA.Body.String())
	}
	var enrollResp protocol.EnrollResponse
	if err := json.NewDecoder(enrollA.Body).Decode(&enrollResp); err != nil {
		t.Fatalf("decode enroll: %v", err)
	}
	if enrollResp.PrivateIP != "10.44.0.1" {
		t.Fatalf("private ip = %q, want 10.44.0.1", enrollResp.PrivateIP)
	}

	_ = postJSON(t, server, "/v1/agents/enroll", protocol.EnrollRequest{
		AgentID:            "agent-b",
		Hostname:           "bravo",
		WireGuardPublicKey: "pub-b",
		Endpoint:           "172.28.0.12:51820",
		EnrollmentToken:    "secret",
	})

	heartbeat := postJSON(t, server, "/v1/agents/heartbeat", protocol.HeartbeatRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha-new",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.111:51820",
		EnrollmentToken:    "secret",
	})
	if heartbeat.Code != http.StatusNoContent {
		t.Fatalf("heartbeat status = %d body=%s", heartbeat.Code, heartbeat.Body.String())
	}

	peers := httptest.NewRecorder()
	server.ServeHTTP(peers, httptest.NewRequest(http.MethodGet, "/v1/agents/agent-a/peers", nil))
	if peers.Code != http.StatusOK {
		t.Fatalf("peers status = %d body=%s", peers.Code, peers.Body.String())
	}
	var peerResp protocol.PeersResponse
	if err := json.NewDecoder(peers.Body).Decode(&peerResp); err != nil {
		t.Fatalf("decode peers: %v", err)
	}
	if len(peerResp.Peers) != 1 || peerResp.Peers[0].AgentID != "agent-b" {
		t.Fatalf("peers = %+v, want only agent-b", peerResp.Peers)
	}
}

func TestServerRejectsUnauthorizedEnrollment(t *testing.T) {
	server := newTestServer(t)

	resp := postJSON(t, server, "/v1/agents/enroll", protocol.EnrollRequest{
		AgentID:         "agent-a",
		EnrollmentToken: "wrong",
	})

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.Code)
	}
}

func TestServerRejectsMalformedJSON(t *testing.T) {
	server := newTestServer(t)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/agents/enroll", bytes.NewBufferString("{"))
	req.Header.Set("content-type", "application/json")

	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.Code)
	}
}

func TestServerUnknownAgent(t *testing.T) {
	server := newTestServer(t)
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/agents/missing/peers", nil)

	server.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.Code)
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	ipam, err := NewIPAM("10.44.0.0/29")
	if err != nil {
		t.Fatalf("new ipam: %v", err)
	}
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
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("encode: %v", err)
	}
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("content-type", "application/json")
	server.ServeHTTP(resp, req)
	return resp
}
