package wireguard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"sync"
)

type LinuxDevice struct {
	mu   sync.Mutex
	name string
	cfg  InterfaceConfig
}

func NewLinuxDevice(name string) *LinuxDevice {
	return &LinuxDevice{name: name}
}

func (d *LinuxDevice) EnsureInterface(ctx context.Context, cfg InterfaceConfig) error {
	if cfg.Name == "" {
		cfg.Name = d.name
	}
	if err := run(ctx, "ip", "link", "show", cfg.Name); err != nil {
		if err := run(ctx, "ip", "link", "add", "dev", cfg.Name, "type", "wireguard"); err != nil {
			return fmt.Errorf("create wireguard interface %s: %w", cfg.Name, err)
		}
	}
	if err := run(ctx, "ip", "addr", "replace", cfg.PrivateIP+cidrSuffix(cfg.NetworkCIDR), "dev", cfg.Name); err != nil {
		return fmt.Errorf("assign private address: %w", err)
	}
	if err := run(ctx, "ip", "link", "set", "up", "dev", cfg.Name); err != nil {
		return fmt.Errorf("bring interface up: %w", err)
	}
	listenPort := fmt.Sprintf("%d", cfg.ListenPort)
	if err := runWithInput(ctx, cfg.PrivateKey+"\n", "wg", "set", cfg.Name, "private-key", "/dev/stdin", "listen-port", listenPort); err != nil {
		return fmt.Errorf("configure private key: %w", err)
	}
	d.mu.Lock()
	d.name = cfg.Name
	d.cfg = cfg
	d.mu.Unlock()
	return nil
}

func (d *LinuxDevice) SyncPeers(ctx context.Context, peers []PeerConfig) error {
	d.mu.Lock()
	cfg := d.cfg
	d.mu.Unlock()
	if cfg.Name == "" || cfg.PrivateKey == "" {
		return fmt.Errorf("wireguard interface must be ensured before syncing peers")
	}
	content := buildWGConfig(cfg, peers)
	file, err := os.CreateTemp("", "ghostwire-wg-*.conf")
	if err != nil {
		return fmt.Errorf("create wg config: %w", err)
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		return fmt.Errorf("write wg config: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close wg config: %w", err)
	}
	if err := run(ctx, "wg", "syncconf", d.name, path); err != nil {
		return fmt.Errorf("sync wireguard peers: %w", err)
	}
	return nil
}

func (d *LinuxDevice) Close(ctx context.Context) error {
	if err := run(ctx, "ip", "link", "delete", "dev", d.name); err != nil {
		return fmt.Errorf("delete interface %s: %w", d.name, err)
	}
	return nil
}

func buildWGConfig(cfg InterfaceConfig, peers []PeerConfig) string {
	ordered := append([]PeerConfig(nil), peers...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].AgentID < ordered[j].AgentID })
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + cfg.PrivateKey + "\n")
	if cfg.ListenPort > 0 {
		b.WriteString(fmt.Sprintf("ListenPort = %d\n", cfg.ListenPort))
	}
	b.WriteString("\n")
	for _, peer := range ordered {
		b.WriteString("[Peer]\n")
		b.WriteString("PublicKey = " + peer.PublicKey + "\n")
		b.WriteString("AllowedIPs = " + peer.AllowedIP + "/32\n")
		if peer.Endpoint != "" {
			b.WriteString("Endpoint = " + peer.Endpoint + "\n")
		}
		if peer.PersistentKeepaliveSeconds > 0 {
			b.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", peer.PersistentKeepaliveSeconds))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func cidrSuffix(cidr string) string {
	idx := strings.LastIndex(cidr, "/")
	if idx == -1 {
		return "/32"
	}
	return cidr[idx:]
}

func run(ctx context.Context, name string, args ...string) error {
	return runWithInput(ctx, "", name, args...)
}

func runWithInput(ctx context.Context, input, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return nil
}
