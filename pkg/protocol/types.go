package protocol

import "time"

type EnrollRequest struct {
	AgentID            string `json:"agent_id"`
	Hostname           string `json:"hostname"`
	WireGuardPublicKey string `json:"wireguard_public_key"`
	Endpoint           string `json:"endpoint"`
}

type EnrollResponse struct {
	AgentID           string        `json:"agent_id"`
	PrivateIP         string        `json:"private_ip"`
	NetworkCIDR       string        `json:"network_cidr"`
	HeartbeatInterval time.Duration `json:"heartbeat_interval"`
	PollInterval      time.Duration `json:"poll_interval"`
}

type HeartbeatRequest struct {
	AgentID            string `json:"agent_id"`
	Hostname           string `json:"hostname"`
	WireGuardPublicKey string `json:"wireguard_public_key"`
	Endpoint           string `json:"endpoint"`
}

type Peer struct {
	AgentID            string    `json:"agent_id"`
	Hostname           string    `json:"hostname"`
	WireGuardPublicKey string    `json:"wireguard_public_key"`
	PrivateIP          string    `json:"private_ip"`
	Endpoint           string    `json:"endpoint"`
	LastSeen           time.Time `json:"last_seen"`
}

type PeersResponse struct {
	Peers []Peer `json:"peers"`
}

type Node struct {
	NodeID             string    `json:"node_id"`
	Hostname           string    `json:"hostname"`
	WireGuardPublicKey string    `json:"wireguard_public_key"`
	GhostwireIP        string    `json:"ghostwire_ip"`
	Endpoint           string    `json:"endpoint"`
	LastSeen           time.Time `json:"last_seen"`
	Status             string    `json:"status"`
}

type NodesResponse struct {
	Nodes []Node `json:"nodes"`
}

const (
	NodeStatusOnline = "online"
	NodeStatusStale  = "stale"
)

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
