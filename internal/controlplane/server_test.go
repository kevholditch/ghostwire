package controlplane

import "testing"

func TestServerHealthz(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.the_health_endpoint_is_requested()
	then.the_request_succeeds()
}

func TestServerEnrollHeartbeatAndPeers(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.agent_a_enrolls()
	then.agent_a_is_given_the_first_ghostwire_ip()
}

func TestServerAcceptsAgentHeartbeat(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.agent_a_sends_a_heartbeat_with_updated_metadata()
	then.the_heartbeat_is_accepted()
}

func TestServerReturnsPeersForAgent(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a().and().
		there_is_an_agent_that_exists_with_hostname_bravo()
	when.agent_a_requests_its_peers()
	then.only_agent_b_is_returned_as_a_peer()
}

func TestServerRejectsMissingAPIToken(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.agent_a_enrolls_without_an_api_token()
	then.the_request_is_rejected_as_unauthorized().and().
		the_error_code_is("unauthorized")
}

func TestServerRejectsMalformedJSON(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.malformed_json_is_posted_to_the_enroll_endpoint()
	then.the_request_is_rejected_as_bad_request()
}

func TestServerUnknownAgent(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.peers_are_requested_for_a_missing_agent()
	then.the_request_is_rejected_as_not_found().and().
		the_error_code_is("not_found")
}

func TestServerListsRegisteredNodes(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.the_nodes_endpoint_is_requested()
	then.the_request_succeeds().and().
		agent_a_is_listed_with_its_node_metadata()
}

func TestServerRejectsUnauthenticatedNodesList(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.the_nodes_endpoint_is_requested_without_an_api_token()
	then.the_request_is_rejected_as_unauthorized().and().
		the_error_code_is("unauthorized")
}

func TestServerReturnsRegisteredNode(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.node_agent_a_is_requested()
	then.the_request_succeeds().and().
		agent_a_is_returned_with_its_node_metadata()
}

func TestServerRejectsUnauthenticatedRegisteredNode(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.node_agent_a_is_requested_without_an_api_token()
	then.the_request_is_rejected_as_unauthorized().and().
		the_error_code_is("unauthorized")
}

func TestServerReturnsNotFoundForMissingRegisteredNode(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.node_missing_is_requested()
	then.the_request_is_rejected_as_not_found().and().
		the_error_code_is("not_found")
}

func TestServerRejectsUnsupportedRegisteredNodeMethod(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.node_agent_a_is_deleted()
	then.the_request_is_rejected_because_the_method_is_not_allowed().and().
		the_error_code_is("method_not_allowed")
}

func TestServerReturnsOperatorPeersForRegisteredNode(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a().and().
		there_is_an_agent_that_exists_with_hostname_bravo()
	when.node_agent_a_peers_are_requested()
	then.the_request_succeeds().and().
		only_agent_b_is_returned_as_an_operator_peer()
}

func TestServerRejectsUnauthenticatedOperatorPeers(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.node_agent_a_peers_are_requested_without_an_api_token()
	then.the_request_is_rejected_as_unauthorized().and().
		the_error_code_is("unauthorized")
}

func TestServerReturnsNotFoundForMissingOperatorPeersNode(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.node_missing_peers_are_requested()
	then.the_request_is_rejected_as_not_found().and().
		the_error_code_is("not_found")
}

func TestServerRejectsUnknownNodeSubpath(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.node_agent_a_unknown_subpath_is_requested()
	then.the_request_is_rejected_as_not_found().and().
		the_error_code_is("not_found")
}

func TestServerRejectsUnsupportedOperatorPeersMethod(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_an_agent_that_exists_with_hostname_agent_a()
	when.node_agent_a_peers_are_deleted()
	then.the_request_is_rejected_because_the_method_is_not_allowed().and().
		the_error_code_is("method_not_allowed")
}

func TestServerRejectsUnsupportedNodesMethod(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.the_nodes_endpoint_is_posted_to()
	then.the_request_is_rejected_because_the_method_is_not_allowed()
}

func TestServerRejectsUnknownV1RouteWithStructuredNotFound(t *testing.T) {
	given, when, then := NewServerStage(t)

	given.there_is_a_control_plane()
	when.an_unknown_v1_route_is_requested()
	then.the_request_is_rejected_as_not_found().and().
		the_error_code_is("not_found")
}
