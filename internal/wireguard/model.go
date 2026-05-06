package wireguard

import "context"

type Device interface {
	EnsureInterface(ctx context.Context, cfg InterfaceConfig) error
	SyncPeers(ctx context.Context, peers []PeerConfig) error
	Close(ctx context.Context) error
}

type InterfaceConfig struct {
	Name        string
	PrivateKey  string
	PrivateIP   string
	NetworkCIDR string
	ListenPort  int
}

type PeerConfig struct {
	AgentID                    string
	PublicKey                  string
	AllowedIP                  string
	Endpoint                   string
	PersistentKeepaliveSeconds int
}
