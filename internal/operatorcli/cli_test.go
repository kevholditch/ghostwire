package operatorcli

import "testing"

func TestNodesListRendersTableAndUsesEnvConfig(t *testing.T) {
	given, when, then := NewCLIStage(t)

	given.there_is_a_control_plane_with_operator_nodes()
	when.nodes_are_listed_using_environment_config()
	then.the_command_succeeds().and().
		the_nodes_table_is_rendered()
}

func TestNodesGetRendersJSON(t *testing.T) {
	given, when, then := NewCLIStage(t)

	given.there_is_a_control_plane_with_operator_nodes()
	when.node_agent_a_is_requested_as_json_using_flags()
	then.the_command_succeeds().and().
		agent_a_json_is_rendered()
}

func TestNodesPeersRendersPeerTable(t *testing.T) {
	given, when, then := NewCLIStage(t)

	given.there_is_a_control_plane_with_operator_nodes()
	when.agent_a_peers_are_listed_using_environment_config()
	then.the_command_succeeds().and().
		only_agent_b_is_rendered_as_a_peer()
}

func TestUnauthorizedResponseReturnsCleanError(t *testing.T) {
	given, when, then := NewCLIStage(t)

	given.there_is_a_control_plane_with_operator_nodes()
	when.nodes_are_listed_with_an_invalid_api_token()
	then.the_command_fails().and().
		the_unauthorized_error_is_rendered()
}
