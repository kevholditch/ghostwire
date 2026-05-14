# Ghostwire

Ghostwire provides private peer-to-peer network connectivity between enrolled devices using WireGuard.

Version 1 is single-tenant:

- One control plane runs at a well-known address.
- Agents enroll with a shared enrollment token.
- Every enrolled agent can reach every other enrolled agent.
- The control plane coordinates membership only; private traffic flows directly between agents over WireGuard.

## Binaries

```bash
go build ./cmd/ghostwire-control
go build ./cmd/ghostwire-agent
go build ./cmd/ghostwire
```

## Control Plane

Required environment:

- `GHOSTWIRE_API_TOKEN`: bearer token required for all `/v1/*` endpoints.

Optional environment:

- `GHOSTWIRE_CONTROL_LISTEN`: listen address, default `:8080`.
- `GHOSTWIRE_NETWORK_CIDR`: private WireGuard CIDR, default `10.44.0.0/24`.
- `GHOSTWIRE_HEARTBEAT_INTERVAL`: interval returned to agents, default `5s`.
- `GHOSTWIRE_POLL_INTERVAL`: peer polling interval returned to agents, default `5s`.
- `GHOSTWIRE_AGENT_TTL`: stale-agent timeout, default `30s`.

Run:

```bash
GHOSTWIRE_API_TOKEN=secret ./ghostwire-control
```

## Agent

Required environment:

- `GHOSTWIRE_CONTROL_URL`: control-plane base URL.
- `GHOSTWIRE_API_TOKEN`: shared API bearer token.

Optional environment:

- `GHOSTWIRE_AGENT_ID`: stable agent ID override; if omitted, one is generated and stored.
- `GHOSTWIRE_HOSTNAME`: advertised hostname; if omitted, OS hostname is used.
- `GHOSTWIRE_STATE_DIR`: identity/key state directory, default `/var/lib/ghostwire`.
- `GHOSTWIRE_ENDPOINT`: advertised WireGuard endpoint, e.g. `203.0.113.10:51820`.
- `GHOSTWIRE_INTERFACE`: WireGuard interface name, default `gw0`.
- `GHOSTWIRE_LISTEN_PORT`: WireGuard listen port, default `51820`.
- `GHOSTWIRE_PERSISTENT_KEEPALIVE`: peer keepalive seconds, default `25`.
- `GHOSTWIRE_HEARTBEAT_INTERVAL`: fallback heartbeat interval, default `5s`.
- `GHOSTWIRE_POLL_INTERVAL`: fallback poll interval, default `5s`.

The agent requires Linux networking privileges and `wireguard-tools` because it configures WireGuard through `ip` and `wg`.

Run:

```bash
sudo GHOSTWIRE_CONTROL_URL=http://control.example:8080 \
  GHOSTWIRE_API_TOKEN=secret \
  GHOSTWIRE_ENDPOINT=203.0.113.10:51820 \
  ./ghostwire-agent
```

## Operator CLI

The `ghostwire` CLI inspects the control plane over the v1 API:

```bash
GHOSTWIRE_CONTROL_URL=http://control.example:8080 \
  GHOSTWIRE_API_TOKEN=secret \
  ./ghostwire nodes list

./ghostwire --control-url http://control.example:8080 \
  --api-token secret \
  nodes get agent-a

./ghostwire --output json nodes peers agent-a
```

The operator API contract is documented in `docs/openapi/v1.json`. `/healthz` is unauthenticated; all `/v1/*` routes require `Authorization: Bearer <token>`.

## Tests

Normal tests do not require Docker or privileged networking:

```bash
go test ./...
```

The end-to-end test builds Linux binaries, starts a control-plane container and two privileged agent containers, waits for peer discovery, then pings both private WireGuard IPs:

```bash
go test -tags=e2e ./e2e
```

E2E requirements:

- Docker with Compose v2 or `docker-compose`.
- `/dev/net/tun` available to containers.
- Containers allowed `NET_ADMIN` and privileged networking operations.
- Host kernel or Docker VM supports WireGuard interfaces.
