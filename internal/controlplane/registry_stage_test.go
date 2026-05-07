package controlplane

import (
	"testing"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
	"github.com/stretchr/testify/require"
)

type registryStageState struct {
	t          *testing.T
	assertions *require.Assertions
	registry   *Registry
	baseTime   time.Time

	agentAEnroll protocol.EnrollResponse
	enrollErr    error

	heartbeatErr error
	agent        AgentRecord
	agentFound   bool

	peers    protocol.PeersResponse
	peersErr error
	nodes    protocol.NodesResponse
}

type registryGiven struct {
	state *registryStageState
}

type registryWhen struct {
	state *registryStageState
}

type registryThen struct {
	state *registryStageState
}

func NewRegistryStage(t *testing.T) (*registryGiven, *registryWhen, *registryThen) {
	t.Helper()
	state := &registryStageState{
		t:          t,
		assertions: require.New(t),
		registry:   newTestRegistry(t),
		baseTime:   time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
	}
	return &registryGiven{state: state}, &registryWhen{state: state}, &registryThen{state: state}
}

func (g *registryGiven) there_is_an_empty_registry() *registryGiven {
	g.state.t.Helper()
	return g
}

func (g *registryGiven) there_is_an_agent_that_exists_with_hostname_agent_a() *registryGiven {
	g.state.t.Helper()
	g.state.agentAEnroll = enrollAgent(g.state.t, g.state.registry, "agent-a", "agent-a", "pub-a", "172.28.0.11:51820", g.state.baseTime)
	return g
}

func (g *registryGiven) there_is_an_agent_that_exists_with_hostname_agent_b() *registryGiven {
	g.state.t.Helper()
	enrollAgent(g.state.t, g.state.registry, "agent-b", "agent-b", "pub-b", "172.28.0.12:51820", g.state.baseTime.Add(time.Second))
	return g
}

func (g *registryGiven) there_is_an_agent_that_exists_with_hostname_zulu() *registryGiven {
	g.state.t.Helper()
	enrollAgent(g.state.t, g.state.registry, "agent-z", "zulu", "pub-z", "172.28.0.13:51820", g.state.baseTime)
	return g
}

func (g *registryGiven) there_is_an_agent_that_exists_with_hostname_alpha() *registryGiven {
	g.state.t.Helper()
	g.state.agentAEnroll = enrollAgent(g.state.t, g.state.registry, "agent-a", "alpha", "pub-a", "172.28.0.11:51820", g.state.baseTime)
	return g
}

func (g *registryGiven) there_is_an_active_agent_a() *registryGiven {
	g.state.t.Helper()
	enrollAgent(g.state.t, g.state.registry, "agent-a", "alpha", "pub-a", "172.28.0.11:51820", g.state.baseTime)
	return g
}

func (g *registryGiven) there_is_an_active_agent_b() *registryGiven {
	g.state.t.Helper()
	enrollAgent(g.state.t, g.state.registry, "agent-b", "bravo", "pub-b", "172.28.0.12:51820", g.state.baseTime)
	return g
}

func (g *registryGiven) there_is_an_expired_agent_c() *registryGiven {
	g.state.t.Helper()
	enrollAgent(g.state.t, g.state.registry, "agent-c", "charlie", "pub-c", "172.28.0.13:51820", g.state.baseTime.Add(-time.Minute))
	return g
}

func (g *registryGiven) and() *registryGiven {
	g.state.t.Helper()
	return g
}

func (w *registryWhen) agent_a_enrolls() {
	w.state.t.Helper()
	w.state.agentAEnroll, w.state.enrollErr = w.state.registry.Enroll(protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.11:51820",
		EnrollmentToken:    "secret",
	}, w.state.baseTime)
}

func (w *registryWhen) agent_a_reenrolls() {
	w.state.t.Helper()
	w.state.agentAEnroll, w.state.enrollErr = w.state.registry.Enroll(protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha-renamed",
		WireGuardPublicKey: "pub-a",
		Endpoint:           "172.28.0.11:51820",
		EnrollmentToken:    "secret",
	}, w.state.baseTime.Add(time.Second))
}

func (w *registryWhen) agent_a_enrolls_with_an_invalid_token() {
	w.state.t.Helper()
	_, w.state.enrollErr = w.state.registry.Enroll(protocol.EnrollRequest{
		AgentID:         "agent-a",
		EnrollmentToken: "wrong",
	}, w.state.baseTime)
}

func (w *registryWhen) agent_a_sends_a_heartbeat_with_updated_metadata() {
	w.state.t.Helper()
	heartbeatTime := w.state.baseTime.Add(5 * time.Second)
	w.state.heartbeatErr = w.state.registry.Heartbeat(protocol.HeartbeatRequest{
		AgentID:            "agent-a",
		Hostname:           "alpha-new",
		WireGuardPublicKey: "pub-a-new",
		Endpoint:           "172.28.0.111:51820",
		EnrollmentToken:    "secret",
	}, heartbeatTime)
	w.state.agent, w.state.agentFound = w.state.registry.Agent("agent-a")
}

func (w *registryWhen) a_missing_agent_sends_a_heartbeat() {
	w.state.t.Helper()
	w.state.heartbeatErr = w.state.registry.Heartbeat(protocol.HeartbeatRequest{
		AgentID:         "missing",
		EnrollmentToken: "secret",
	}, w.state.baseTime)
}

func (w *registryWhen) agent_a_requests_its_peers() {
	w.state.t.Helper()
	w.state.peers, w.state.peersErr = w.state.registry.Peers("agent-a", w.state.baseTime.Add(15*time.Second))
}

func (w *registryWhen) the_control_plane_lists_registered_nodes() {
	w.state.t.Helper()
	w.state.nodes = w.state.registry.Nodes(w.state.baseTime.Add(2 * time.Second))
}

func (w *registryWhen) agent_a_reenrolls_with_new_metadata() {
	w.state.t.Helper()
	w.state.agentAEnroll, w.state.enrollErr = w.state.registry.Enroll(protocol.EnrollRequest{
		AgentID:            "agent-a",
		Hostname:           "agent-a-renamed",
		WireGuardPublicKey: "pub-a-new",
		Endpoint:           "172.28.0.111:51820",
		EnrollmentToken:    "secret",
	}, w.state.baseTime.Add(3*time.Second))
	w.state.nodes = w.state.registry.Nodes(w.state.baseTime.Add(3 * time.Second))
}

func (th *registryThen) agent_a_is_given_the_first_ghostwire_ip() *registryThen {
	th.state.t.Helper()
	th.state.assertions.NoError(th.state.enrollErr)
	th.state.assertions.Equal("10.44.0.1", th.state.agentAEnroll.PrivateIP)
	return th
}

func (th *registryThen) the_agent_is_rejected_as_unauthorized() *registryThen {
	th.state.t.Helper()
	th.state.assertions.ErrorIs(th.state.enrollErr, ErrUnauthorized)
	return th
}

func (th *registryThen) agent_a_metadata_is_refreshed() *registryThen {
	th.state.t.Helper()
	th.state.assertions.NoError(th.state.heartbeatErr)
	th.state.assertions.True(th.state.agentFound)
	th.state.assertions.Equal("alpha-new", th.state.agent.Hostname)
	th.state.assertions.Equal("pub-a-new", th.state.agent.WireGuardPublicKey)
	th.state.assertions.Equal("172.28.0.111:51820", th.state.agent.Endpoint)
	th.state.assertions.Equal(th.state.baseTime.Add(5*time.Second), th.state.agent.LastSeen)
	return th
}

func (th *registryThen) the_agent_is_rejected_as_not_found() *registryThen {
	th.state.t.Helper()
	th.state.assertions.ErrorIs(th.state.heartbeatErr, ErrAgentNotFound)
	return th
}

func (th *registryThen) only_agent_b_is_returned_as_a_peer() *registryThen {
	th.state.t.Helper()
	th.state.assertions.NoError(th.state.peersErr)
	th.state.assertions.Equal([]protocol.Peer{
		{
			AgentID:            "agent-b",
			Hostname:           "bravo",
			WireGuardPublicKey: "pub-b",
			PrivateIP:          "10.44.0.2",
			Endpoint:           "172.28.0.12:51820",
			LastSeen:           th.state.baseTime,
		},
	}, th.state.peers.Peers)
	return th
}

func (th *registryThen) agent_a_is_listed_with_its_node_metadata() *registryThen {
	th.state.t.Helper()
	node := th.findNode("agent-a")
	th.state.assertions.Equal(protocol.Node{
		NodeID:             "agent-a",
		Hostname:           "agent-a",
		WireGuardPublicKey: "pub-a",
		GhostwireIP:        "10.44.0.1",
		LastSeen:           th.state.baseTime,
	}, node)
	return th
}

func (th *registryThen) agent_a_keeps_its_ghostwire_ip() *registryThen {
	th.state.t.Helper()
	th.state.assertions.NoError(th.state.enrollErr)
	th.state.assertions.Equal("10.44.0.1", th.state.agentAEnroll.PrivateIP)
	return th
}

func (th *registryThen) agent_a_metadata_is_updated() *registryThen {
	th.state.t.Helper()
	node := th.findNode("agent-a")
	th.state.assertions.Equal("agent-a-renamed", node.Hostname)
	th.state.assertions.Equal("pub-a-new", node.WireGuardPublicKey)
	th.state.assertions.Equal("10.44.0.1", node.GhostwireIP)
	return th
}

func (th *registryThen) agent_a_is_listed_before_agent_z() *registryThen {
	th.state.t.Helper()
	th.state.assertions.Len(th.state.nodes.Nodes, 2)
	th.state.assertions.Equal("agent-a", th.state.nodes.Nodes[0].NodeID)
	th.state.assertions.Equal("agent-z", th.state.nodes.Nodes[1].NodeID)
	return th
}

func (th *registryThen) and() *registryThen {
	th.state.t.Helper()
	return th
}

func (th *registryThen) findNode(nodeID string) protocol.Node {
	th.state.t.Helper()
	for _, node := range th.state.nodes.Nodes {
		if node.NodeID == nodeID {
			return node
		}
	}
	th.state.assertions.FailNow("node not found", "node_id=%s nodes=%+v", nodeID, th.state.nodes.Nodes)
	return protocol.Node{}
}

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	assertions := require.New(t)
	ipam, err := NewIPAM("10.44.0.0/29")
	assertions.NoError(err)
	return NewRegistry(RegistryConfig{
		EnrollmentToken:   "secret",
		NetworkCIDR:       "10.44.0.0/29",
		HeartbeatInterval: time.Second,
		PollInterval:      time.Second,
		AgentTTL:          30 * time.Second,
	}, ipam)
}

func enrollAgent(t *testing.T, registry *Registry, id, hostname, publicKey, endpoint string, now time.Time) protocol.EnrollResponse {
	t.Helper()
	resp, err := registry.Enroll(protocol.EnrollRequest{
		AgentID:            id,
		Hostname:           hostname,
		WireGuardPublicKey: publicKey,
		Endpoint:           endpoint,
		EnrollmentToken:    "secret",
	}, now)
	require.NoError(t, err)
	return resp
}
