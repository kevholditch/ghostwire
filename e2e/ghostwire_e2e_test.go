//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
)

func TestAgentsCommunicateOverWireGuardPrivateIPs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	repo := repoRoot(t)
	t.Log("building linux binaries")
	buildLinuxBinaries(t, ctx, repo)

	compose := composeCommand(t)
	project := fmt.Sprintf("ghostwire-e2e-%d", os.Getpid())
	args := append(compose, "-f", filepath.Join(repo, "e2e", "docker-compose.yml"), "-p", project)
	defer runBestEffort(t, ctx, append(args, "down", "-v", "--remove-orphans")...)

	t.Log("starting docker compose environment")
	run(t, ctx, repo, append(args, "up", "--build", "-d")...)
	t.Log("waiting for control plane health")
	waitForHealth(t, ctx)
	t.Log("waiting for agent-a to see agent-b")
	agentBPrivateIP := waitForPeerPrivateIP(t, ctx, "agent-a", "agent-b")
	t.Logf("agent-a sees agent-b at %s", agentBPrivateIP)
	t.Log("waiting for agent-b to see agent-a")
	agentAPrivateIP := waitForPeerPrivateIP(t, ctx, "agent-b", "agent-a")
	t.Logf("agent-b sees agent-a at %s", agentAPrivateIP)

	t.Logf("pinging agent-b private IP %s from agent-a", agentBPrivateIP)
	run(t, ctx, repo, append(args, "exec", "-T", "agent-a", "ping", "-c", "3", "-W", "2", agentBPrivateIP)...)
	t.Logf("pinging agent-a private IP %s from agent-b", agentAPrivateIP)
	run(t, ctx, repo, append(args, "exec", "-T", "agent-b", "ping", "-c", "3", "-W", "2", agentAPrivateIP)...)
}

func buildLinuxBinaries(t *testing.T, ctx context.Context, repo string) {
	t.Helper()
	binDir := filepath.Join(repo, "e2e", "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("create bin dir: %v", err)
	}
	build := func(output, pkg string) {
		t.Helper()
		cmd := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(binDir, output), pkg)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("build %s: %v\n%s", pkg, err, out)
		}
	}
	build("ghostwire-control", "./cmd/ghostwire-control")
	build("ghostwire-agent", "./cmd/ghostwire-agent")
}

func waitForHealth(t *testing.T, ctx context.Context) {
	t.Helper()
	waitFor(t, ctx, "control health", func() bool {
		resp, err := http.Get("http://127.0.0.1:18080/healthz")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	})
}

func waitForPeerPrivateIP(t *testing.T, ctx context.Context, agentID, wantPeer string) string {
	t.Helper()
	var privateIP string
	waitFor(t, ctx, agentID+" sees "+wantPeer, func() bool {
		resp, err := http.Get("http://127.0.0.1:18080/v1/agents/" + agentID + "/peers")
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return false
		}
		var peers protocol.PeersResponse
		if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
			return false
		}
		for _, peer := range peers.Peers {
			if peer.AgentID == wantPeer {
				privateIP = peer.PrivateIP
				return privateIP != ""
			}
		}
		return false
	})
	return privateIP
}

func waitFor(t *testing.T, ctx context.Context, name string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s: %v", name, ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}
	t.Fatalf("timed out waiting for %s", name)
}

func composeCommand(t *testing.T) []string {
	t.Helper()
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.Command("docker", "compose", "version")
		if err := cmd.Run(); err == nil {
			return []string{"docker", "compose"}
		}
	}
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}
	}
	t.Fatal("docker compose is required for e2e tests")
	return nil
}

func repoRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	return strings.TrimSpace(string(out))
}

func run(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run %s: %v", strings.Join(args, " "), err)
	}
}

func runBestEffort(t *testing.T, ctx context.Context, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	_ = cmd.Run()
}
