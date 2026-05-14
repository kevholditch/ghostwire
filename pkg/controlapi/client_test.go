package controlapi

import "testing"

func TestClientListNodesSendsBearerTokenAndDecodesResponse(t *testing.T) {
	given, when, then := NewClientStage(t)

	given.there_is_a_control_plane_with_operator_nodes()
	when.nodes_are_listed()
	then.the_request_succeeds().and().
		the_bearer_token_was_sent().and().
		agent_a_is_returned()
}

func TestClientGetNodeReturnsStructuredAPIError(t *testing.T) {
	given, when, then := NewClientStage(t)

	given.there_is_a_control_plane_that_rejects_requests_as_unauthorized()
	when.node_agent_a_is_requested()
	then.the_request_fails_with_unauthorized_api_error()
}

func TestClientListNodePeersEscapesNodeIDPathSegment(t *testing.T) {
	given, when, then := NewClientStage(t)

	given.there_is_a_control_plane_that_records_peer_requests()
	when.peers_are_listed_for_node_id_containing_a_slash()
	then.the_request_succeeds().and().
		the_node_id_path_segment_was_escaped()
}
