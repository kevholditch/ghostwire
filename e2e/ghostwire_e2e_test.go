//go:build e2e

package e2e

import "testing"

func TestStableNodeIdentityAndWireGuardConnectivity(t *testing.T) {
	given, when, then := NewGhostwireStage(t)

	given.the_linux_binaries_are_built().and().
		the_docker_compose_environment_is_running().and().
		the_agents_have_joined_the_control_plane().and().
		the_operator_cli_can_inspect_the_joined_agents().and().
		the_agents_can_communicate_over_wireguard()
	when.agent_a_is_restarted()
	then.agent_a_keeps_the_same_stable_node_identity().and().
		the_control_plane_still_lists_two_registered_nodes().and().
		agent_a_can_ping_agent_b_over_wireguard().and().
		agent_b_can_ping_agent_a_over_wireguard()
}
