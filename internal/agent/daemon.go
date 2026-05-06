package agent

import (
	"context"
	"log"
	"time"

	"github.com/kevholditch/ghostwire/internal/wireguard"
	"github.com/kevholditch/ghostwire/pkg/protocol"
)

type Daemon struct {
	cfg      Config
	identity Identity
	client   *Client
	device   wireguard.Device
}

func NewDaemon(cfg Config, identity Identity, client *Client, device wireguard.Device) *Daemon {
	return &Daemon{cfg: cfg, identity: identity, client: client, device: device}
}

func (d *Daemon) Run(ctx context.Context) error {
	enroll, err := d.enrollUntilSuccess(ctx)
	if err != nil {
		return err
	}
	reconciler := NewReconciler(d.device, WireGuardConfig{
		InterfaceName:              d.cfg.InterfaceName,
		PrivateKey:                 d.identity.PrivateKey,
		PrivateIP:                  enroll.PrivateIP,
		NetworkCIDR:                enroll.NetworkCIDR,
		ListenPort:                 d.cfg.ListenPort,
		PersistentKeepaliveSeconds: d.cfg.PersistentKeepaliveSeconds,
	})

	heartbeatInterval := firstPositive(enroll.HeartbeatInterval, d.cfg.HeartbeatInterval, 5*time.Second)
	pollInterval := firstPositive(enroll.PollInterval, d.cfg.PollInterval, 5*time.Second)
	heartbeatTicker := time.NewTicker(heartbeatInterval)
	pollTicker := time.NewTicker(pollInterval)
	defer heartbeatTicker.Stop()
	defer pollTicker.Stop()

	if err := d.heartbeat(ctx); err != nil {
		log.Printf("heartbeat failed: %v", err)
	}
	d.pollAndApply(ctx, reconciler)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-heartbeatTicker.C:
			if err := d.heartbeat(ctx); err != nil {
				log.Printf("heartbeat failed: %v", err)
			}
		case <-pollTicker.C:
			d.pollAndApply(ctx, reconciler)
		}
	}
}

func (d *Daemon) enrollUntilSuccess(ctx context.Context) (protocol.EnrollResponse, error) {
	backoff := time.Second
	for {
		resp, err := d.client.Enroll(ctx, protocol.EnrollRequest{
			AgentID:            d.identity.AgentID,
			Hostname:           d.cfg.Hostname,
			WireGuardPublicKey: d.identity.PublicKey,
			Endpoint:           d.cfg.Endpoint,
			EnrollmentToken:    d.cfg.EnrollmentToken,
		})
		if err == nil {
			log.Printf("enrolled agent_id=%s private_ip=%s", resp.AgentID, resp.PrivateIP)
			return resp, nil
		}
		log.Printf("enroll failed: %v", err)
		select {
		case <-ctx.Done():
			return protocol.EnrollResponse{}, ctx.Err()
		case <-time.After(backoff):
			if backoff < 10*time.Second {
				backoff *= 2
			}
		}
	}
}

func (d *Daemon) heartbeat(ctx context.Context) error {
	return d.client.Heartbeat(ctx, protocol.HeartbeatRequest{
		AgentID:            d.identity.AgentID,
		Hostname:           d.cfg.Hostname,
		WireGuardPublicKey: d.identity.PublicKey,
		Endpoint:           d.cfg.Endpoint,
		EnrollmentToken:    d.cfg.EnrollmentToken,
	})
}

func (d *Daemon) pollAndApply(ctx context.Context, reconciler *Reconciler) {
	peers, err := d.client.Peers(ctx, d.identity.AgentID)
	if err != nil {
		log.Printf("peer poll failed: %v", err)
		return
	}
	if err := reconciler.Apply(ctx, peers.Peers); err != nil {
		log.Printf("wireguard reconcile failed: %v", err)
		return
	}
	log.Printf("reconciled %d peers", len(peers.Peers))
}

func firstPositive(values ...time.Duration) time.Duration {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
