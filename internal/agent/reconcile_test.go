package agent

import "testing"

func TestReconcilerAppliesInterfaceAndPeers(t *testing.T) {
	given, when, then := NewReconcilerStage(t)

	given.there_is_a_reconciler_for_agent_a().and().
		there_is_a_peer_snapshot_for_agent_b()
	when.the_reconciler_applies_the_peer_snapshot()
	then.the_wireguard_interface_is_configured_for_agent_a().and().
		agent_b_is_configured_as_a_wireguard_peer().and().
		the_wireguard_device_is_not_closed()
}
