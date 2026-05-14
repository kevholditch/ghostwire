package controlplane

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
)

var (
	ErrUnauthorized  = errors.New("unauthorized")
	ErrAgentNotFound = errors.New("agent not found")
	ErrInvalidAgent  = errors.New("invalid agent")
)

type RegistryConfig struct {
	NetworkCIDR       string
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	AgentTTL          time.Duration
}

type AgentRecord struct {
	AgentID            string
	Hostname           string
	WireGuardPublicKey string
	PrivateIP          string
	Endpoint           string
	LastSeen           time.Time
}

type Registry struct {
	mu     sync.Mutex
	cfg    RegistryConfig
	ipam   *IPAM
	agents map[string]AgentRecord
}

func NewRegistry(cfg RegistryConfig, ipam *IPAM) *Registry {
	return &Registry{
		cfg:    cfg,
		ipam:   ipam,
		agents: map[string]AgentRecord{},
	}
}

func (r *Registry) Enroll(req protocol.EnrollRequest, now time.Time) (protocol.EnrollResponse, error) {
	if req.AgentID == "" || req.WireGuardPublicKey == "" {
		return protocol.EnrollResponse{}, ErrInvalidAgent
	}

	privateIP, err := r.ipam.Allocate(req.AgentID)
	if err != nil {
		return protocol.EnrollResponse{}, err
	}

	r.mu.Lock()
	r.agents[req.AgentID] = AgentRecord{
		AgentID:            req.AgentID,
		Hostname:           req.Hostname,
		WireGuardPublicKey: req.WireGuardPublicKey,
		PrivateIP:          privateIP,
		Endpoint:           req.Endpoint,
		LastSeen:           now,
	}
	r.mu.Unlock()

	return protocol.EnrollResponse{
		AgentID:           req.AgentID,
		PrivateIP:         privateIP,
		NetworkCIDR:       r.cfg.NetworkCIDR,
		HeartbeatInterval: r.cfg.HeartbeatInterval,
		PollInterval:      r.cfg.PollInterval,
	}, nil
}

func (r *Registry) Heartbeat(req protocol.HeartbeatRequest, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[req.AgentID]
	if !ok {
		return ErrAgentNotFound
	}
	if req.Hostname != "" {
		agent.Hostname = req.Hostname
	}
	if req.WireGuardPublicKey != "" {
		agent.WireGuardPublicKey = req.WireGuardPublicKey
	}
	if req.Endpoint != "" {
		agent.Endpoint = req.Endpoint
	}
	agent.LastSeen = now
	r.agents[req.AgentID] = agent
	return nil
}

func (r *Registry) Peers(agentID string, now time.Time) (protocol.PeersResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.agents[agentID]; !ok {
		return protocol.PeersResponse{}, ErrAgentNotFound
	}

	peers := make([]protocol.Peer, 0, len(r.agents))
	for id, agent := range r.agents {
		if id == agentID || r.isExpired(agent, now) {
			continue
		}
		peers = append(peers, protocol.Peer{
			AgentID:            agent.AgentID,
			Hostname:           agent.Hostname,
			WireGuardPublicKey: agent.WireGuardPublicKey,
			PrivateIP:          agent.PrivateIP,
			Endpoint:           agent.Endpoint,
			LastSeen:           agent.LastSeen,
		})
	}
	sort.Slice(peers, func(i, j int) bool { return peers[i].AgentID < peers[j].AgentID })
	return protocol.PeersResponse{Peers: peers}, nil
}

func (r *Registry) Nodes(now time.Time) protocol.NodesResponse {
	r.mu.Lock()
	defer r.mu.Unlock()

	nodes := make([]protocol.Node, 0, len(r.agents))
	for _, agent := range r.agents {
		nodes = append(nodes, r.nodeFromAgent(agent, now))
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].NodeID < nodes[j].NodeID })
	return protocol.NodesResponse{Nodes: nodes}
}

func (r *Registry) Node(nodeID string, now time.Time) (protocol.Node, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[nodeID]
	if !ok {
		return protocol.Node{}, ErrAgentNotFound
	}
	return r.nodeFromAgent(agent, now), nil
}

func (r *Registry) NodePeers(nodeID string, now time.Time) (protocol.NodesResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.agents[nodeID]; !ok {
		return protocol.NodesResponse{}, ErrAgentNotFound
	}

	nodes := make([]protocol.Node, 0, len(r.agents))
	for id, agent := range r.agents {
		if id == nodeID || r.isExpired(agent, now) {
			continue
		}
		nodes = append(nodes, r.nodeFromAgent(agent, now))
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].NodeID < nodes[j].NodeID })
	return protocol.NodesResponse{Nodes: nodes}, nil
}

func (r *Registry) Agent(agentID string) (AgentRecord, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	agent, ok := r.agents[agentID]
	return agent, ok
}

func (r *Registry) isExpired(agent AgentRecord, now time.Time) bool {
	if r.cfg.AgentTTL <= 0 {
		return false
	}
	return now.Sub(agent.LastSeen) > r.cfg.AgentTTL
}

func (r *Registry) nodeFromAgent(agent AgentRecord, now time.Time) protocol.Node {
	status := protocol.NodeStatusOnline
	if r.isExpired(agent, now) {
		status = protocol.NodeStatusStale
	}
	return protocol.Node{
		NodeID:             agent.AgentID,
		Hostname:           agent.Hostname,
		WireGuardPublicKey: agent.WireGuardPublicKey,
		GhostwireIP:        agent.PrivateIP,
		Endpoint:           agent.Endpoint,
		LastSeen:           agent.LastSeen,
		Status:             status,
	}
}
