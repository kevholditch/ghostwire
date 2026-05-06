package controlplane

import (
	"errors"
	"testing"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
)

func TestRegistryEnrollsAgentAndAssignsStablePrivateIP(t *testing.T) {
	registry := newTestRegistry(t)
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)

	resp, err := registry.Enroll(protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.11:51820",
		EnrollmentToken:    "secret",
	}, now)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}

	if resp.PrivateIP != "10.44.0.1" {
		t.Fatalf("private ip = %q, want 10.44.0.1", resp.PrivateIP)
	}

	again, err := registry.Enroll(protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha-renamed",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.11:51820",
		EnrollmentToken:    "secret",
	}, now.Add(time.Second))
	if err != nil {
		t.Fatalf("re-enroll: %v", err)
	}
	if again.PrivateIP != resp.PrivateIP {
		t.Fatalf("stable ip = %q then %q", resp.PrivateIP, again.PrivateIP)
	}
}

func TestRegistryRejectsInvalidEnrollmentToken(t *testing.T) {
	registry := newTestRegistry(t)

	_, err := registry.Enroll(protocol.EnrollRequest{
		AgentID:         "agent-a",
		EnrollmentToken: "wrong",
	}, time.Now())
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v, want ErrUnauthorized", err)
	}
}

func TestRegistryHeartbeatRefreshesAgent(t *testing.T) {
	registry := newTestRegistry(t)
	enrollTime := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	_, err := registry.Enroll(protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.11:51820",
		EnrollmentToken:    "secret",
	}, enrollTime)
	if err != nil {
		t.Fatalf("enroll: %v", err)
	}

	heartbeatTime := enrollTime.Add(5 * time.Second)
	err = registry.Heartbeat(protocol.HeartbeatRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha-new",
		WireGuardPublicKey: "pub-a-new",
		Endpoint:           "172.28.0.111:51820",
		EnrollmentToken:    "secret",
	}, heartbeatTime)
	if err != nil {
		t.Fatalf("heartbeat: %v", err)
	}

	agent, ok := registry.Agent("agent-a")
	if !ok {
		t.Fatal("agent not found")
	}
	if agent.Hostname != "alpha-new" || agent.WireGuardPublicKey != "pub-a-new" || agent.Endpoint != "172.28.0.111:51820" {
		t.Fatalf("agent not refreshed: %+v", agent)
	}
	if !agent.LastSeen.Equal(heartbeatTime) {
		t.Fatalf("last seen = %s, want %s", agent.LastSeen, heartbeatTime)
	}
}

func TestRegistryHeartbeatUnknownAgent(t *testing.T) {
	registry := newTestRegistry(t)

	err := registry.Heartbeat(protocol.HeartbeatRequest{
		AgentID:         "missing",
		EnrollmentToken: "secret",
	}, time.Now())
	if !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("err = %v, want ErrAgentNotFound", err)
	}
}

func TestRegistryPeersExcludeRequesterAndExpiredAgents(t *testing.T) {
	registry := newTestRegistry(t)
	base := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	enrollAgent(t, registry, "agent-a", "alpha", "pub-a", "172.28.0.11:51820", base)
	enrollAgent(t, registry, "agent-b", "bravo", "pub-b", "172.28.0.12:51820", base)
	enrollAgent(t, registry, "agent-c", "charlie", "pub-c", "172.28.0.13:51820", base.Add(-time.Minute))

	peers, err := registry.Peers("agent-a", base.Add(15*time.Second))
	if err != nil {
		t.Fatalf("peers: %v", err)
	}
	if len(peers.Peers) != 1 {
		t.Fatalf("peers len = %d, want 1: %+v", len(peers.Peers), peers.Peers)
	}
	if peers.Peers[0].AgentID != "agent-b" {
		t.Fatalf("peer = %q, want agent-b", peers.Peers[0].AgentID)
	}
	if peers.Peers[0].PrivateIP != "10.44.0.2" {
		t.Fatalf("peer private ip = %q, want 10.44.0.2", peers.Peers[0].PrivateIP)
	}
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	ipam, err := NewIPAM("10.44.0.0/29")
	if err != nil {
		t.Fatalf("new ipam: %v", err)
	}
	return NewRegistry(RegistryConfig{
		EnrollmentToken:   "secret",
		NetworkCIDR:       "10.44.0.0/29",
		HeartbeatInterval: time.Second,
		PollInterval:      time.Second,
		AgentTTL:          30 * time.Second,
	}, ipam)
}

func enrollAgent(t *testing.T, registry *Registry, id, hostname, publicKey, endpoint string, now time.Time) {
	t.Helper()
	_, err := registry.Enroll(protocol.EnrollRequest{
		AgentID:            id,
		Hostname:           hostname,
		WireGuardPublicKey: publicKey,
		Endpoint:           endpoint,
		EnrollmentToken:    "secret",
	}, now)
	if err != nil {
		t.Fatalf("enroll %s: %v", id, err)
	}
}
