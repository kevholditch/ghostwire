# Ghostwire V1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first runnable Ghostwire control plane and agent, including behavior-focused unit tests and a Docker-backed e2e test that proves two agents can communicate over WireGuard private IPs.

**Architecture:** The control plane exposes a small HTTP JSON API backed by an in-memory registry and CIDR allocator. Agents create persistent identity and WireGuard key state, enroll with the control plane, poll for peers, and reconcile local WireGuard state through a narrow device interface. Production WireGuard integration uses Linux `ip` and `wg` commands isolated in `internal/wireguard`; tests use fakes.

**Tech Stack:** Go standard library, Linux `ip` command, `wireguard-tools`, Docker Compose for privileged e2e tests.

---

## File Structure

- Create `go.mod`: module declaration for `github.com/kevholditch/ghostwire`.
- Create `pkg/protocol/types.go`: stable JSON API request/response types shared by both binaries.
- Create `internal/controlplane/config.go`: control-plane configuration with environment parsing.
- Create `internal/controlplane/ipam.go` and `internal/controlplane/ipam_test.go`: CIDR allocation behavior.
- Create `internal/controlplane/registry.go` and `internal/controlplane/registry_test.go`: enrollment, heartbeat, stale expiry, peer snapshot behavior.
- Create `internal/controlplane/server.go` and `internal/controlplane/server_test.go`: HTTP API handlers and auth/error behavior.
- Create `cmd/ghostwire-control/main.go`: process entrypoint.
- Create `internal/wireguard/model.go`: interface and config types.
- Create `internal/wireguard/fake_device.go`: fake implementation for tests.
- Create `internal/wireguard/keys.go`: local key generation/public derivation via `wg`.
- Create `internal/wireguard/linux_device.go`: production interface and peer reconciliation via `ip`/`wg`.
- Create `internal/agent/config.go`: agent configuration with environment parsing.
- Create `internal/agent/identity.go`: stable identity and key file management.
- Create `internal/agent/client.go`: HTTP client for control-plane API.
- Create `internal/agent/reconcile.go` and `internal/agent/reconcile_test.go`: translate peer snapshots into WireGuard desired state.
- Create `internal/agent/daemon.go`: enrollment, heartbeat, poll, and reconciliation loop.
- Create `cmd/ghostwire-agent/main.go`: process entrypoint.
- Create `e2e/Dockerfile`, `e2e/docker-compose.yml`, and `e2e/ghostwire_e2e_test.go`: privileged Docker e2e test pack.
- Create `README.md`: concise build/test/run instructions.

## Tasks

### Task 1: Module and protocol contract

- [ ] Write `go.mod` with module path `github.com/kevholditch/ghostwire` and Go 1.22.
- [ ] Write `pkg/protocol/types.go` with enroll, heartbeat, peer, and error response types using JSON field names from the approved design.
- [ ] Run `go test ./...`; expect package discovery to compile once code exists, or report no packages before later tasks.

### Task 2: Control-plane IPAM

- [ ] Write failing tests in `internal/controlplane/ipam_test.go` for deterministic allocation, stable lease reuse for the same agent, duplicate prevention, and CIDR exhaustion.
- [ ] Run `go test ./internal/controlplane -run TestIPAM`; expected failure is missing IPAM implementation.
- [ ] Implement `internal/controlplane/ipam.go` with a mutex-protected allocator that skips network and broadcast addresses for IPv4 CIDRs.
- [ ] Run `go test ./internal/controlplane -run TestIPAM`; expected pass.

### Task 3: Control-plane registry

- [ ] Write failing tests in `internal/controlplane/registry_test.go` for enrollment, heartbeat refresh, peer listing excluding self, unauthorized token rejection, and stale expiry.
- [ ] Run `go test ./internal/controlplane -run 'TestRegistry'`; expected failure is missing registry implementation.
- [ ] Implement `internal/controlplane/registry.go` using the IPAM allocator and in-memory agent records.
- [ ] Run `go test ./internal/controlplane -run 'TestRegistry|TestIPAM'`; expected pass.

### Task 4: Control-plane HTTP server and binary

- [ ] Write failing handler tests in `internal/controlplane/server_test.go` for health, enroll success, heartbeat, peers, unauthorized token, malformed JSON, and unknown agent.
- [ ] Run `go test ./internal/controlplane`; expected failure is missing HTTP server implementation.
- [ ] Implement `internal/controlplane/config.go` and `internal/controlplane/server.go`.
- [ ] Implement `cmd/ghostwire-control/main.go` to load env config and serve HTTP.
- [ ] Run `go test ./internal/controlplane` and `go test ./...`; expected pass.

### Task 5: WireGuard abstraction

- [ ] Write `internal/wireguard/model.go` with `Device`, `InterfaceConfig`, and `PeerConfig`.
- [ ] Write `internal/wireguard/fake_device.go` for tests.
- [ ] Write `internal/wireguard/keys.go` using `wg genkey` and `wg pubkey` with explicit command errors.
- [ ] Write `internal/wireguard/linux_device.go` using `ip link`, `ip addr`, and `wg setconf` with a generated config file.
- [ ] Run `gofmt` and `go test ./internal/wireguard`; expected pass.

### Task 6: Agent identity, client, and reconciliation

- [ ] Write failing tests in `internal/agent/reconcile_test.go` that verify interface config and peer config derived from a peer snapshot.
- [ ] Run `go test ./internal/agent -run TestReconcile`; expected failure is missing agent implementation.
- [ ] Implement `internal/agent/config.go`, `internal/agent/identity.go`, `internal/agent/client.go`, and `internal/agent/reconcile.go`.
- [ ] Run `go test ./internal/agent -run TestReconcile`; expected pass.

### Task 7: Agent daemon and binary

- [ ] Implement `internal/agent/daemon.go` with enroll retry, heartbeat loop, peer poll loop, and reconciliation loop.
- [ ] Implement `cmd/ghostwire-agent/main.go` to load env config, create key state, create the Linux WireGuard device, and run the daemon until interrupted.
- [ ] Run `go test ./...`; expected pass.

### Task 8: E2E Docker pack

- [ ] Write `e2e/Dockerfile` installing `wireguard-tools`, `iproute2`, and `iputils-ping`, and copying built binaries.
- [ ] Write `e2e/docker-compose.yml` with one control container and two privileged agent containers using `/dev/net/tun`, static Docker network IPs, explicit WireGuard endpoints, and private IPs allocated by the control plane.
- [ ] Write `e2e/ghostwire_e2e_test.go` with build tag `e2e`; it builds Linux binaries, starts Docker Compose, waits for enrollment, and pings private WireGuard IPs both directions.
- [ ] Run `go test ./...`; expected pass without running e2e.
- [ ] Run `go test -tags=e2e ./e2e`; expected pass if Docker supports TUN and WireGuard in the local environment, otherwise report the exact environmental blocker.

### Task 9: Documentation and final verification

- [ ] Write `README.md` with architecture summary, normal test command, e2e test command, required Docker privileges, and environment variables.
- [ ] Run `gofmt` across all Go files.
- [ ] Run `go test ./...`.
- [ ] Run `go test -tags=e2e ./e2e` or report the environment-specific blocker.
- [ ] Commit implementation changes with a focused commit message.

## Self-Review

- Spec coverage: the plan covers the two binaries, HTTP control-plane API, in-memory registry, agent identity and reconciliation loop, WireGuard boundary, behavior-focused unit tests, and Docker e2e tests.
- Placeholder scan: no task relies on placeholder behavior; each task names concrete files and expected verification commands.
- Type consistency: shared API types live in `pkg/protocol`; control-plane and agent tasks use the same request/response names.
