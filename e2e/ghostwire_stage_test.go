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
	"github.com/stretchr/testify/require"
)

type ghostwireStageState struct {
	t          *testing.T
	assertions *require.Assertions
	ctx        context.Context
	cancel     context.CancelFunc

	repo        string
	composeArgs []string

	agentABefore protocol.Node
	agentAAfter  protocol.Node
	agentB       protocol.Node

	agentBPrivateIP string
	agentAPrivateIP string
	nodesAfter      protocol.NodesResponse
}

type ghostwireGiven struct {
	state *ghostwireStageState
}

type ghostwireWhen struct {
	state *ghostwireStageState
}

type ghostwireThen struct {
	state *ghostwireStageState
}

func NewGhostwireStage(t *testing.T) (*ghostwireGiven, *ghostwireWhen, *ghostwireThen) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	repo := repoRoot(t)
	compose := composeCommand(t)
	project := fmt.Sprintf("ghostwire-e2e-%d", os.Getpid())
	composeArgs := append(compose, "-f", filepath.Join(repo, "e2e", "docker-compose.yml"), "-p", project)

	state := &ghostwireStageState{
		t:           t,
		assertions:  require.New(t),
		ctx:         ctx,
		cancel:      cancel,
		repo:        repo,
		composeArgs: composeArgs,
	}
	t.Cleanup(func() {
		runBestEffort(t, ctx, append(composeArgs, "down", "-v", "--remove-orphans")...)
		cancel()
	})
	return &ghostwireGiven{state: state}, &ghostwireWhen{state: state}, &ghostwireThen{state: state}
}

func (g *ghostwireGiven) the_linux_binaries_are_built() *ghostwireGiven {
	g.state.t.Helper()
	g.state.t.Log("building linux binaries")
	binDir := filepath.Join(g.state.repo, "e2e", "bin")
	g.state.assertions.NoError(os.MkdirAll(binDir, 0o755))
	g.build("ghostwire-control", "./cmd/ghostwire-control")
	g.build("ghostwire-agent", "./cmd/ghostwire-agent")
	return g
}

func (g *ghostwireGiven) the_docker_compose_environment_is_running() *ghostwireGiven {
	g.state.t.Helper()
	g.state.t.Log("starting docker compose environment")
	run(g.state.t, g.state.ctx, g.state.repo, append(g.state.composeArgs, "up", "--build", "-d")...)
	g.state.t.Log("waiting for control plane health")
	waitForHealth(g.state.t, g.state.ctx)
	return g
}

func (g *ghostwireGiven) the_agents_have_joined_the_control_plane() *ghostwireGiven {
	g.state.t.Helper()
	g.state.t.Log("waiting for both agents to register")
	nodes := waitForNodesByHostname(g.state.t, g.state.ctx, "agent-a", "agent-b")
	g.state.agentABefore = nodes["agent-a"]
	g.state.agentB = nodes["agent-b"]

	g.state.t.Log("waiting for agent-a to see agent-b")
	g.state.agentBPrivateIP = waitForPeerPrivateIP(g.state.t, g.state.ctx, g.state.agentABefore.NodeID, g.state.agentB.NodeID)
	g.state.t.Log("waiting for agent-b to see agent-a")
	g.state.agentAPrivateIP = waitForPeerPrivateIP(g.state.t, g.state.ctx, g.state.agentB.NodeID, g.state.agentABefore.NodeID)
	return g
}

func (g *ghostwireGiven) the_agents_can_communicate_over_wireguard() *ghostwireGiven {
	g.state.t.Helper()
	g.state.assertions.NotEmpty(g.state.agentBPrivateIP)
	g.state.t.Logf("pinging agent-b private IP %s from agent-a", g.state.agentBPrivateIP)
	run(g.state.t, g.state.ctx, g.state.repo, append(g.state.composeArgs, "exec", "-T", "agent-a", "ping", "-c", "3", "-W", "2", g.state.agentBPrivateIP)...)

	g.state.assertions.NotEmpty(g.state.agentAPrivateIP)
	g.state.t.Logf("pinging agent-a private IP %s from agent-b", g.state.agentAPrivateIP)
	run(g.state.t, g.state.ctx, g.state.repo, append(g.state.composeArgs, "exec", "-T", "agent-b", "ping", "-c", "3", "-W", "2", g.state.agentAPrivateIP)...)
	return g
}

func (g *ghostwireGiven) and() *ghostwireGiven {
	g.state.t.Helper()
	return g
}

func (w *ghostwireWhen) the_agents_join_the_control_plane() {
	w.state.t.Helper()
	w.state.t.Log("waiting for both agents to register")
	nodes := waitForNodesByHostname(w.state.t, w.state.ctx, "agent-a", "agent-b")
	w.state.agentABefore = nodes["agent-a"]
	w.state.agentB = nodes["agent-b"]

	w.state.t.Log("waiting for agent-a to see agent-b")
	w.state.agentBPrivateIP = waitForPeerPrivateIP(w.state.t, w.state.ctx, w.state.agentABefore.NodeID, w.state.agentB.NodeID)
	w.state.t.Log("waiting for agent-b to see agent-a")
	w.state.agentAPrivateIP = waitForPeerPrivateIP(w.state.t, w.state.ctx, w.state.agentB.NodeID, w.state.agentABefore.NodeID)
}

func (w *ghostwireWhen) agent_a_is_restarted() {
	w.state.t.Helper()
	w.state.t.Log("restarting agent-a")
	run(w.state.t, w.state.ctx, w.state.repo, append(w.state.composeArgs, "restart", "agent-a")...)
	w.state.t.Log("waiting for agent-a to return with node metadata")
	w.state.agentAAfter = waitForNodeAfterRestart(w.state.t, w.state.ctx, "agent-a", w.state.agentABefore.LastSeen)
	w.state.nodesAfter = waitForNodes(w.state.t, w.state.ctx, 2)

	w.state.t.Log("waiting for peers after agent-a restart")
	w.state.agentBPrivateIP = waitForPeerPrivateIP(w.state.t, w.state.ctx, w.state.agentAAfter.NodeID, w.state.agentB.NodeID)
	w.state.agentAPrivateIP = waitForPeerPrivateIP(w.state.t, w.state.ctx, w.state.agentB.NodeID, w.state.agentAAfter.NodeID)
}

func (th *ghostwireThen) agent_a_is_registered_with_node_metadata() *ghostwireThen {
	th.state.t.Helper()
	assertNodeMetadata(th.state.t, th.state.agentABefore, "agent-a")
	return th
}

func (th *ghostwireThen) agent_b_is_registered_with_node_metadata() *ghostwireThen {
	th.state.t.Helper()
	assertNodeMetadata(th.state.t, th.state.agentB, "agent-b")
	return th
}

func (th *ghostwireThen) agent_a_can_ping_agent_b_over_wireguard() *ghostwireThen {
	th.state.t.Helper()
	th.state.assertions.NotEmpty(th.state.agentBPrivateIP)
	th.state.t.Logf("pinging agent-b private IP %s from agent-a", th.state.agentBPrivateIP)
	run(th.state.t, th.state.ctx, th.state.repo, append(th.state.composeArgs, "exec", "-T", "agent-a", "ping", "-c", "3", "-W", "2", th.state.agentBPrivateIP)...)
	return th
}

func (th *ghostwireThen) agent_b_can_ping_agent_a_over_wireguard() *ghostwireThen {
	th.state.t.Helper()
	th.state.assertions.NotEmpty(th.state.agentAPrivateIP)
	th.state.t.Logf("pinging agent-a private IP %s from agent-b", th.state.agentAPrivateIP)
	run(th.state.t, th.state.ctx, th.state.repo, append(th.state.composeArgs, "exec", "-T", "agent-b", "ping", "-c", "3", "-W", "2", th.state.agentAPrivateIP)...)
	return th
}

func (th *ghostwireThen) agent_a_keeps_the_same_stable_node_identity() *ghostwireThen {
	th.state.t.Helper()
	before := th.state.agentABefore
	after := th.state.agentAAfter
	th.state.assertions.Equal(before.NodeID, after.NodeID)
	th.state.assertions.Equal(before.WireGuardPublicKey, after.WireGuardPublicKey)
	th.state.assertions.Equal(before.GhostwireIP, after.GhostwireIP)
	th.state.assertions.True(after.LastSeen.After(before.LastSeen))
	return th
}

func (th *ghostwireThen) the_control_plane_still_lists_two_registered_nodes() *ghostwireThen {
	th.state.t.Helper()
	th.state.assertions.Len(th.state.nodesAfter.Nodes, 2)
	return th
}

func (th *ghostwireThen) and() *ghostwireThen {
	th.state.t.Helper()
	return th
}

func (g *ghostwireGiven) build(output, pkg string) {
	g.state.t.Helper()
	cmd := exec.CommandContext(g.state.ctx, "go", "build", "-o", filepath.Join(g.state.repo, "e2e", "bin", output), pkg)
	cmd.Dir = g.state.repo
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	g.state.assertions.NoError(err, "build %s\n%s", pkg, out)
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

func waitForNodesByHostname(t *testing.T, ctx context.Context, hostnames ...string) map[string]protocol.Node {
	t.Helper()
	var matched map[string]protocol.Node
	waitFor(t, ctx, "nodes by hostname", func() bool {
		nodes, ok := getNodes(t, ctx)
		if !ok {
			return false
		}
		matched = map[string]protocol.Node{}
		for _, node := range nodes.Nodes {
			for _, hostname := range hostnames {
				if node.Hostname == hostname {
					matched[hostname] = node
				}
			}
		}
		for _, hostname := range hostnames {
			if matched[hostname].NodeID == "" {
				return false
			}
		}
		return true
	})
	return matched
}

func waitForNodeAfterRestart(t *testing.T, ctx context.Context, hostname string, previousLastSeen time.Time) protocol.Node {
	t.Helper()
	var restarted protocol.Node
	waitFor(t, ctx, hostname+" after restart", func() bool {
		nodes := waitForNodesByHostname(t, ctx, hostname)
		restarted = nodes[hostname]
		return restarted.LastSeen.After(previousLastSeen)
	})
	return restarted
}

func waitForNodes(t *testing.T, ctx context.Context, count int) protocol.NodesResponse {
	t.Helper()
	var nodes protocol.NodesResponse
	waitFor(t, ctx, "node count", func() bool {
		var ok bool
		nodes, ok = getNodes(t, ctx)
		return ok && len(nodes.Nodes) == count
	})
	return nodes
}

func getNodes(t *testing.T, ctx context.Context) (protocol.NodesResponse, bool) {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:18080/v1/nodes", nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return protocol.NodesResponse{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return protocol.NodesResponse{}, false
	}
	var nodes protocol.NodesResponse
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		return protocol.NodesResponse{}, false
	}
	return nodes, true
}

func waitForPeerPrivateIP(t *testing.T, ctx context.Context, agentID, wantPeer string) string {
	t.Helper()
	var privateIP string
	waitFor(t, ctx, agentID+" sees "+wantPeer, func() bool {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:18080/v1/agents/"+agentID+"/peers", nil)
		require.NoError(t, err)
		resp, err := http.DefaultClient.Do(req)
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
			require.FailNowf(t, "timed out", "timed out waiting for %s: %v", name, ctx.Err())
		case <-time.After(500 * time.Millisecond):
		}
	}
	require.FailNowf(t, "timed out", "timed out waiting for %s", name)
}

func assertNodeMetadata(t *testing.T, node protocol.Node, hostname string) {
	t.Helper()
	require.NotEmpty(t, node.NodeID)
	require.Equal(t, hostname, node.Hostname)
	require.NotEmpty(t, node.WireGuardPublicKey)
	require.NotEmpty(t, node.GhostwireIP)
	require.False(t, node.LastSeen.IsZero())
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
	require.FailNow(t, "docker compose is required for e2e tests")
	return nil
}

func repoRoot(t *testing.T) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(out))
}

func run(t *testing.T, ctx context.Context, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	require.NoError(t, cmd.Run(), "run %s", strings.Join(args, " "))
}

func runBestEffort(t *testing.T, ctx context.Context, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	_ = cmd.Run()
}
