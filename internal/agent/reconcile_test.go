package agent

import (
	"context"
	"testing"
	"time"

	"github.com/kevholditch/ghostwire/internal/wireguard"
	"github.com/kevholditch/ghostwire/pkg/protocol"
)

func TestReconcilerAppliesInterfaceAndPeers(t *testing.T) {
	device := &wireguard.FakeDevice{}
	reconciler := NewReconciler(device, WireGuardConfig{
		InterfaceName:              "gw0",
		PrivateKey:                 "private-key",
		PrivateIP:                  "10.44.0.1",
		NetworkCIDR:                "10.44.0.0/24",
		ListenPort:                 51820,
		PersistentKeepaliveSeconds: 25,
	})

	err := reconciler.Apply(context.Background(), []protocol.Peer{
		{
			AgentID:            "agent-b",
			Hostname:           "bravo",
			WireGuardPublicKey: "pub-b",
			PrivateIP:          "10.44.0.2",
			Endpoint:           "172.28.0.12:51820",
			LastSeen:           time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}

	iface, peers, closed := device.Snapshot()
	if closed {
		t.Fatal("device should not be closed")
	}
	if iface.Name != "gw0" || iface.PrivateKey != "private-key" || iface.PrivateIP != "10.44.0.1" || iface.NetworkCIDR != "10.44.0.0/24" || iface.ListenPort != 51820 {
		t.Fatalf("interface config = %+v", iface)
	}
	if len(peers) != 1 {
		t.Fatalf("peer len = %d, want 1", len(peers))
	}
	wantPeer := wireguard.PeerConfig{
		AgentID:                    "agent-b",
		PublicKey:                  "pub-b",
		AllowedIP:                  "10.44.0.2",
		Endpoint:                   "172.28.0.12:51820",
		PersistentKeepaliveSeconds: 25,
	}
	if peers[0] != wantPeer {
		t.Fatalf("peer = %+v, want %+v", peers[0], wantPeer)
	}
}
