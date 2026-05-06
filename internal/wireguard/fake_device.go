package wireguard

import (
	"context"
	"sync"
)

type FakeDevice struct {
	mu              sync.Mutex
	InterfaceConfig InterfaceConfig
	Peers           []PeerConfig
	Closed          bool
}

func (f *FakeDevice) EnsureInterface(_ context.Context, cfg InterfaceConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.InterfaceConfig = cfg
	return nil
}

func (f *FakeDevice) SyncPeers(_ context.Context, peers []PeerConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Peers = append([]PeerConfig(nil), peers...)
	return nil
}

func (f *FakeDevice) Close(_ context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Closed = true
	return nil
}

func (f *FakeDevice) Snapshot() (InterfaceConfig, []PeerConfig, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.InterfaceConfig, append([]PeerConfig(nil), f.Peers...), f.Closed
}
