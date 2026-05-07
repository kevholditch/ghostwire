package agent

import (
	"time"

	"github.com/kevholditch/ghostwire/internal/wireguard"
	"github.com/kevholditch/ghostwire/pkg/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

type reconcilerStageState struct {
	t          *testing.T
	assertions *require.Assertions

	device     *wireguard.FakeDevice
	reconciler *Reconciler
	peers      []protocol.Peer
	applyErr   error

	iface       wireguard.InterfaceConfig
	devicePeers []wireguard.PeerConfig
	closed      bool
}

type reconcilerGiven struct {
	state *reconcilerStageState
}

type reconcilerWhen struct {
	state *reconcilerStageState
}

type reconcilerThen struct {
	state *reconcilerStageState
}

func NewReconcilerStage(t *testing.T) (*reconcilerGiven, *reconcilerWhen, *reconcilerThen) {
	t.Helper()
	state := &reconcilerStageState{
		t:          t,
		assertions: require.New(t),
		device:     &wireguard.FakeDevice{},
	}
	return &reconcilerGiven{state: state}, &reconcilerWhen{state: state}, &reconcilerThen{state: state}
}

func (g *reconcilerGiven) there_is_a_reconciler_for_agent_a() *reconcilerGiven {
	g.state.t.Helper()
	g.state.reconciler = NewReconciler(g.state.device, WireGuardConfig{
		InterfaceName:              "gw0",
		PrivateKey:                 "private-key",
		PrivateIP:                  "10.44.0.1",
		NetworkCIDR:                "10.44.0.0/24",
		ListenPort:                 51820,
		PersistentKeepaliveSeconds: 25,
	})
	return g
}

func (g *reconcilerGiven) there_is_a_peer_snapshot_for_agent_b() *reconcilerGiven {
	g.state.t.Helper()
	g.state.peers = []protocol.Peer{
		{
			AgentID:            "agent-b",
			Hostname:           "bravo",
			WireGuardPublicKey: "pub-b",
			PrivateIP:          "10.44.0.2",
			Endpoint:           "172.28.0.12:51820",
			LastSeen:           time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		},
	}
	return g
}

func (g *reconcilerGiven) and() *reconcilerGiven {
	g.state.t.Helper()
	return g
}

func (w *reconcilerWhen) the_reconciler_applies_the_peer_snapshot() {
	w.state.t.Helper()
	w.state.applyErr = w.state.reconciler.Apply(w.state.t.Context(), w.state.peers)
	w.state.iface, w.state.devicePeers, w.state.closed = w.state.device.Snapshot()
}

func (th *reconcilerThen) the_wireguard_interface_is_configured_for_agent_a() *reconcilerThen {
	th.state.t.Helper()
	th.state.assertions.NoError(th.state.applyErr)
	th.state.assertions.Equal(wireguard.InterfaceConfig{
		Name:        "gw0",
		PrivateKey:  "private-key",
		PrivateIP:   "10.44.0.1",
		NetworkCIDR: "10.44.0.0/24",
		ListenPort:  51820,
	}, th.state.iface)
	return th
}

func (th *reconcilerThen) agent_b_is_configured_as_a_wireguard_peer() *reconcilerThen {
	th.state.t.Helper()
	th.state.assertions.Equal([]wireguard.PeerConfig{
		{
			AgentID:                    "agent-b",
			PublicKey:                  "pub-b",
			AllowedIP:                  "10.44.0.2",
			Endpoint:                   "172.28.0.12:51820",
			PersistentKeepaliveSeconds: 25,
		},
	}, th.state.devicePeers)
	return th
}

func (th *reconcilerThen) the_wireguard_device_is_not_closed() *reconcilerThen {
	th.state.t.Helper()
	th.state.assertions.False(th.state.closed)
	return th
}

func (th *reconcilerThen) and() *reconcilerThen {
	th.state.t.Helper()
	return th
}
