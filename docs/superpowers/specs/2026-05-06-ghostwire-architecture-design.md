# Ghostwire Architecture Design

## Goal

Ghostwire is a Go application that provides private network connectivity between enrolled devices using WireGuard. Version 1 is single-tenant: every agent enrolls with the same shared enrollment token, and every enrolled agent may connect to every other enrolled agent.

The system has two binaries:

- `ghostwire-control`: a control-plane service running at a well-known address.
- `ghostwire-agent`: a device-local daemon that joins the private network and configures WireGuard.

The control plane coordinates membership only. Inter-device traffic must flow peer to peer over WireGuard and must not proxy through the control plane.

## Non-Goals For Version 1

- Multi-tenant organizations.
- User accounts, SSO, OAuth, or per-user authentication.
- ACLs or policy-driven reachability.
- Durable control-plane storage.
- Relay servers or DERP-style fallback paths.
- Kernel-independent data-plane implementation.

These are excluded to keep the first version focused on coordination, local WireGuard configuration, and an end-to-end proof that two agents can communicate over private WireGuard IPs.

## Repository Structure

```text
cmd/
  ghostwire-control/
    main.go
  ghostwire-agent/
    main.go

internal/
  controlplane/
    api.go
    config.go
    ipam.go
    registry.go
    server.go
  agent/
    client.go
    config.go
    daemon.go
    identity.go
    reconcile.go
  endpoint/
    detect.go
  log/
    log.go
  wireguard/
    device.go
    fake_device.go
    keys.go
    linux_device.go
    model.go

pkg/
  protocol/
    types.go

e2e/
  Dockerfile
  docker-compose.yml
  ghostwire_e2e_test.go

testdata/
  e2e/
    agent-a.yaml
    agent-b.yaml
    control.yaml
```

`cmd` packages stay thin and only handle process wiring. Long-lived logic belongs in `internal`. Shared API contracts live in `pkg/protocol` so both binaries compile against one protocol definition without sharing internal implementation details.

## Control Plane

The control plane owns membership state and private IP assignment. It does not handle private network traffic.

Responsibilities:

- Load configuration: listen address, API token, private CIDR, lease TTL, and heartbeat TTL.
- Validate the shared API bearer token on all `/v1/*` requests.
- Allocate stable private WireGuard IPs from the configured CIDR.
- Store each live agent's stable ID, hostname, WireGuard public key, private IP, observed endpoint, and heartbeat timestamp.
- Return peer snapshots to agents, excluding the requesting agent.
- Expire stale agents after they miss the configured TTL.
- Expose a health endpoint for tests and container orchestration.

Version 1 storage is in-memory behind a registry abstraction. This keeps the control plane simple while preserving a future path to durable storage.

### HTTP API

The v1 API uses HTTP JSON. It is easier to inspect, test, and debug than a streaming protocol, and it is sufficient for polling-based membership updates.

Endpoints:

```text
POST /v1/agents/enroll
POST /v1/agents/heartbeat
GET  /v1/agents/{agent_id}/peers
GET  /healthz
```

Enrollment request fields:

- `agent_id`
- `hostname`
- `wireguard_public_key`
- `endpoint`

Enrollment response fields:

- `agent_id`
- `private_ip`
- `network_cidr`
- `heartbeat_interval`
- `poll_interval`

Heartbeat request fields:

- `agent_id`
- `hostname`
- `wireguard_public_key`
- `endpoint`

Peer snapshot response fields:

- `peers`: list of peer records containing `agent_id`, `hostname`, `wireguard_public_key`, `private_ip`, `endpoint`, and `last_seen`.

The API token is intentionally simple for v1. It is not a substitute for a full identity system, but it is enough to keep random unauthenticated clients from joining or inspecting a single private network during the first implementation.

## Agent

The agent is a long-running daemon that turns control-plane membership into local WireGuard configuration.

Responsibilities:

- Load configuration: control-plane URL, API token, WireGuard interface name, state directory, endpoint, and timing values.
- Load or create a stable local agent identity.
- Generate and persist a WireGuard private key locally.
- Derive the WireGuard public key and send only the public key to the control plane.
- Enroll with the control plane.
- Heartbeat periodically so the control plane can keep endpoint and liveness state fresh.
- Poll the peer snapshot endpoint.
- Reconcile the local WireGuard interface and peer set to match the latest snapshot.

The agent should tolerate temporary control-plane failures by keeping the last successfully applied WireGuard configuration in place and retrying enrollment, heartbeat, and peer polling with bounded backoff.

## WireGuard Boundary

WireGuard operations must be isolated behind a small interface. Agent orchestration should not shell out directly or depend on netlink details.

```go
type Device interface {
    EnsureInterface(ctx context.Context, cfg InterfaceConfig) error
    SyncPeers(ctx context.Context, peers []PeerConfig) error
    Close(ctx context.Context) error
}
```

Production implementation:

- Creates or updates the configured WireGuard interface.
- Assigns the agent's private WireGuard IP to the interface.
- Applies the local private key and listen port.
- Replaces the configured peer set with the current desired full-mesh peer list.

Test implementation:

- Records the requested interface and peer state.
- Allows agent reconciliation logic to be tested without `NET_ADMIN`, `/dev/net/tun`, or a WireGuard kernel module.

This boundary is the main protection against implementation-coupled tests.

## Data Flow

Startup flow:

1. Agent loads or creates `agent_id`.
2. Agent loads or creates WireGuard private key.
3. Agent derives public key.
4. Agent enrolls with the control plane using the shared enrollment token.
5. Control plane validates the token, assigns or returns the agent's private IP, and stores the agent record.
6. Agent ensures its local WireGuard interface exists with the assigned private IP.
7. Agent polls peers and applies them to WireGuard.

Ongoing flow:

1. Agent heartbeats with current endpoint and public key.
2. Control plane updates `last_seen` and endpoint state.
3. Agent polls peer snapshots.
4. Agent reconciles local WireGuard peers.
5. Private traffic flows directly between agent WireGuard endpoints.

## Endpoint Handling

Version 1 accepts an explicit endpoint in agent configuration. If the endpoint is not configured, the agent may use local best-effort detection, but the first reliable e2e path should configure endpoints through Docker networking.

This avoids coupling v1 to NAT traversal complexity. Persistent keepalive should be configured for peers so the basic shape remains compatible with NATed environments later.

## Error Handling

Control plane:

- Invalid or missing enrollment token returns `401 Unauthorized`.
- Malformed JSON returns `400 Bad Request`.
- Unknown agent on heartbeat or peer lookup returns `404 Not Found`.
- CIDR exhaustion returns `503 Service Unavailable`.
- Health endpoint returns `200 OK` after configuration and registry initialization succeed.

Agent:

- Enrollment failure logs the cause and retries with bounded backoff.
- Heartbeat and peer polling failures log and retry without tearing down existing WireGuard configuration.
- WireGuard reconciliation failures log and retry on the next loop.
- Invalid local identity or key files fail fast with a clear error because silently replacing identity would create duplicate devices.

## Testing Strategy

Tests should cover stable behavior boundaries, not incidental implementation details.

Unit tests:

- `internal/controlplane/ipam`: deterministic IP allocation from a CIDR, stable lease return for an existing agent, and no duplicate leases.
- `internal/controlplane/registry`: successful enrollment, heartbeat refresh, stale-agent expiry, peer snapshots excluding the requesting agent.
- `internal/agent/reconcile`: given a private IP and peer snapshot, the agent asks the WireGuard device to converge to the expected interface and peer state.
- `internal/wireguard/keys`: key generation and public-key derivation if this logic is owned by Ghostwire rather than delegated entirely to a library.

End-to-end tests:

- Gated behind the `e2e` build tag.
- Build Linux binaries for `ghostwire-control` and `ghostwire-agent`.
- Start three Docker containers: one control plane and two agents.
- Grant agent containers `NET_ADMIN` and `/dev/net/tun`.
- Wait for both agents to enroll.
- Assert each agent receives the other in its peer snapshot.
- Ping agent B's private WireGuard IP from agent A.
- Ping agent A's private WireGuard IP from agent B.

The e2e test command is:

```bash
go test -tags=e2e ./e2e
```

Normal test runs should not require privileged Docker containers:

```bash
go test ./...
```

## Architectural Decisions

- Use HTTP JSON for v1 control-plane communication.
- Use an in-memory registry for v1 control-plane state.
- Generate WireGuard keys on agents, never on the control plane.
- Store only WireGuard public keys in the control plane.
- Configure a full mesh where every live enrolled agent can reach every other live enrolled agent.
- Poll peer snapshots rather than keeping a streaming control-plane connection.
- Keep WireGuard system integration behind `internal/wireguard.Device`.
- Use Docker e2e tests as the proof that peer-to-peer private connectivity works.

## Future Extension Points

The design leaves room for these future changes without reshaping the first implementation:

- Persistent registry storage.
- Rotating enrollment tokens.
- User authentication and device ownership.
- ACLs and tags.
- gRPC or server-sent event peer updates.
- NAT traversal and relay fallback.
- Control-plane clustering.
