package controlplane

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type ipamStageState struct {
	t          *testing.T
	assertions *require.Assertions

	ipam *IPAM

	agentAFirst  string
	agentASecond string
	agentB       string
	thirdErr     error
	createErr    error
}

type ipamGiven struct {
	state *ipamStageState
}

type ipamWhen struct {
	state *ipamStageState
}

type ipamThen struct {
	state *ipamStageState
}

func NewIPAMStage(t *testing.T) (*ipamGiven, *ipamWhen, *ipamThen) {
	t.Helper()
	state := &ipamStageState{
		t:          t,
		assertions: require.New(t),
	}
	return &ipamGiven{state: state}, &ipamWhen{state: state}, &ipamThen{state: state}
}

func (g *ipamGiven) there_is_an_ipam_for_a_29_network() {
	g.state.t.Helper()
	ipam, err := NewIPAM("10.44.0.0/29")
	g.state.assertions.NoError(err)
	g.state.ipam = ipam
}

func (g *ipamGiven) there_is_an_ipam_for_a_30_network() {
	g.state.t.Helper()
	ipam, err := NewIPAM("10.44.0.0/30")
	g.state.assertions.NoError(err)
	g.state.ipam = ipam
}

func (g *ipamGiven) there_is_no_existing_ipam() {
	g.state.t.Helper()
}

func (w *ipamWhen) agent_a_and_agent_b_are_allocated_addresses() {
	w.state.t.Helper()
	first, err := w.state.ipam.Allocate("agent-a")
	w.state.assertions.NoError(err)
	second, err := w.state.ipam.Allocate("agent-b")
	w.state.assertions.NoError(err)
	w.state.agentAFirst = first
	w.state.agentB = second
}

func (w *ipamWhen) agent_a_is_allocated_an_address_twice() {
	w.state.t.Helper()
	first, err := w.state.ipam.Allocate("agent-a")
	w.state.assertions.NoError(err)
	second, err := w.state.ipam.Allocate("agent-a")
	w.state.assertions.NoError(err)
	w.state.agentAFirst = first
	w.state.agentASecond = second
}

func (w *ipamWhen) three_agents_are_allocated_addresses() {
	w.state.t.Helper()
	_, err := w.state.ipam.Allocate("agent-a")
	w.state.assertions.NoError(err)
	_, err = w.state.ipam.Allocate("agent-b")
	w.state.assertions.NoError(err)
	_, w.state.thirdErr = w.state.ipam.Allocate("agent-c")
}

func (w *ipamWhen) an_ipam_is_created_with_an_invalid_cidr() {
	w.state.t.Helper()
	_, w.state.createErr = NewIPAM("not-a-cidr")
}

func (th *ipamThen) agent_a_is_given_the_first_usable_address() *ipamThen {
	th.state.t.Helper()
	th.state.assertions.Equal("10.44.0.1", th.state.agentAFirst)
	return th
}

func (th *ipamThen) agent_b_is_given_the_second_usable_address() *ipamThen {
	th.state.t.Helper()
	th.state.assertions.Equal("10.44.0.2", th.state.agentB)
	return th
}

func (th *ipamThen) the_agents_have_unique_addresses() *ipamThen {
	th.state.t.Helper()
	th.state.assertions.NotEqual(th.state.agentAFirst, th.state.agentB)
	return th
}

func (th *ipamThen) agent_a_is_given_the_same_address_both_times() *ipamThen {
	th.state.t.Helper()
	th.state.assertions.Equal(th.state.agentAFirst, th.state.agentASecond)
	return th
}

func (th *ipamThen) the_third_agent_is_rejected_because_the_cidr_is_exhausted() *ipamThen {
	th.state.t.Helper()
	th.state.assertions.ErrorIs(th.state.thirdErr, ErrCIDRExhausted)
	return th
}

func (th *ipamThen) the_cidr_is_rejected() *ipamThen {
	th.state.t.Helper()
	th.state.assertions.Error(th.state.createErr)
	return th
}

func (th *ipamThen) and() *ipamThen {
	th.state.t.Helper()
	return th
}
