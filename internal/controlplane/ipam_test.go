package controlplane

import "testing"

func TestIPAMAllocatesDeterministicUsableIPv4Addresses(t *testing.T) {
	given, when, then := NewIPAMStage(t)

	given.there_is_an_ipam_for_a_29_network()
	when.agent_a_and_agent_b_are_allocated_addresses()
	then.agent_a_is_given_the_first_usable_address().and().
		agent_b_is_given_the_second_usable_address().and().
		the_agents_have_unique_addresses()
}

func TestIPAMReturnsStableLeaseForExistingAgent(t *testing.T) {
	given, when, then := NewIPAMStage(t)

	given.there_is_an_ipam_for_a_29_network()
	when.agent_a_is_allocated_an_address_twice()
	then.agent_a_is_given_the_same_address_both_times()
}

func TestIPAMReportsExhaustion(t *testing.T) {
	given, when, then := NewIPAMStage(t)

	given.there_is_an_ipam_for_a_30_network()
	when.three_agents_are_allocated_addresses()
	then.the_third_agent_is_rejected_because_the_cidr_is_exhausted()
}

func TestIPAMRejectsInvalidCIDR(t *testing.T) {
	given, when, then := NewIPAMStage(t)

	given.there_is_no_existing_ipam()
	when.an_ipam_is_created_with_an_invalid_cidr()
	then.the_cidr_is_rejected()
}
