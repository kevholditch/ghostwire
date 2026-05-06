package agent

import (
	"context"

	"github.com/kevholditch/ghostwire/internal/wireguard"
	"github.com/kevholditch/ghostwire/pkg/protocol"
)

type WireGuardConfig struct {
	InterfaceName              string
	PrivateKey                 string
	PrivateIP                  string
	NetworkCIDR                string
	ListenPort                 int
	PersistentKeepaliveSeconds int
}

type Reconciler struct {
	device wireguard.Device
	cfg    WireGuardConfig
}

func NewReconciler(device wireguard.Device, cfg WireGuardConfig) *Reconciler {
	return &Reconciler{device: device, cfg: cfg}
}

func (r *Reconciler) Apply(ctx context.Context, peers []protocol.Peer) error {
	if err := r.device.EnsureInterface(ctx, wireguard.InterfaceConfig{
		Name:        r.cfg.InterfaceName,
		PrivateKey:  r.cfg.PrivateKey,
		PrivateIP:   r.cfg.PrivateIP,
		NetworkCIDR: r.cfg.NetworkCIDR,
		ListenPort:  r.cfg.ListenPort,
	}); err != nil {
		return err
	}
	configs := make([]wireguard.PeerConfig, 0, len(peers))
	for _, peer := range peers {
		configs = append(configs, wireguard.PeerConfig{
			AgentID:                    peer.AgentID,
			PublicKey:                  peer.WireGuardPublicKey,
			AllowedIP:                  peer.PrivateIP,
			Endpoint:                   peer.Endpoint,
			PersistentKeepaliveSeconds: r.cfg.PersistentKeepaliveSeconds,
		})
	}
	return r.device.SyncPeers(ctx, configs)
}
