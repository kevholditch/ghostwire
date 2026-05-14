package controlplane

import "testing"

func TestRegistryEnrollsAgentAndAssignsStablePrivateIP(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_empty_registry()
	when.agent_a_enrolls()
	then.agent_a_is_given_the_first_ghostwire_ip()
}

func TestRegistryKeepsPrivateIPWhenAgentReenrolls(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_agent_that_exists_with_hostname_alpha()
	when.agent_a_reenrolls()
	then.agent_a_keeps_its_ghostwire_ip()
}

func TestRegistryHeartbeatRefreshesAgent(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_agent_that_exists_with_hostname_alpha()
	when.agent_a_sends_a_heartbeat_with_updated_metadata()
	then.agent_a_metadata_is_refreshed()
}

func TestRegistryHeartbeatUnknownAgent(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_empty_registry()
	when.a_missing_agent_sends_a_heartbeat()
	then.the_agent_is_rejected_as_not_found()
}

func TestRegistryPeersExcludeRequesterAndExpiredAgents(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_active_agent_a().and().
		there_is_an_active_agent_b().and().
		there_is_an_expired_agent_c()
	when.agent_a_requests_its_peers()
	then.only_agent_b_is_returned_as_a_peer()
}

func TestControlPlaneListsRegisteredNodes(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.the_control_plane_lists_registered_nodes()
	then.agent_a_is_listed_with_its_node_metadata()
}

func TestControlPlaneKeepsGhostwireIPWhenNodeReenrolls(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.agent_a_reenrolls_with_new_metadata()
	then.agent_a_keeps_its_ghostwire_ip().and().
		agent_a_metadata_is_updated()
}

func TestControlPlaneListsNodesByNodeID(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_agent_that_exists_with_hostname_zulu().and().
		there_is_an_agent_that_exists_with_hostname_alpha()
	when.the_control_plane_lists_registered_nodes()
	then.agent_a_is_listed_before_agent_z()
}

func TestControlPlaneListsStaleNodesWithStatus(t *testing.T) {
	given, when, then := NewRegistryStage(t)

	given.there_is_an_active_agent_a().and().
		there_is_an_expired_agent_c()
	when.the_control_plane_lists_registered_nodes()
	then.agent_a_is_listed_as_online().and().
		agent_c_is_listed_as_stale()
}
