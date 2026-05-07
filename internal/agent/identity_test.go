package agent

import "testing"

func TestGeneratedAgentIDIsReused(t *testing.T) {
	given, when, then := NewIdentityStage(t)

	given.there_is_no_existing_agent_identity()
	when.the_agent_id_is_loaded_twice()
	then.a_generated_agent_id_is_reused()
}

func TestConfiguredAgentIDIsStoredWhenNoIdentityExists(t *testing.T) {
	given, when, then := NewIdentityStage(t)

	given.there_is_no_existing_agent_identity()
	when.the_agent_id_is_loaded_with_configured_agent_a()
	then.the_configured_agent_a_id_is_stored()
}

func TestExistingAgentIDWinsOverConfiguredAgentID(t *testing.T) {
	given, when, then := NewIdentityStage(t)

	given.there_is_an_existing_agent_identity()
	when.the_agent_id_is_loaded_with_configured_agent_b()
	then.the_existing_agent_id_is_used()
}

func TestExistingWireGuardPrivateKeyIsReused(t *testing.T) {
	given, when, then := NewIdentityStage(t)

	given.there_is_an_existing_wireguard_private_key()
	when.the_wireguard_private_key_is_loaded_twice()
	then.the_existing_wireguard_private_key_is_reused()
}
