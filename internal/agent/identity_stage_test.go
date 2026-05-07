package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type identityStageState struct {
	t          *testing.T
	assertions *require.Assertions
	stateDir   string

	firstAgentID  string
	secondAgentID string
	firstKey      string
	secondKey     string
}

type identityGiven struct {
	state *identityStageState
}

type identityWhen struct {
	state *identityStageState
}

type identityThen struct {
	state *identityStageState
}

func NewIdentityStage(t *testing.T) (*identityGiven, *identityWhen, *identityThen) {
	t.Helper()
	state := &identityStageState{
		t:          t,
		assertions: require.New(t),
		stateDir:   t.TempDir(),
	}
	return &identityGiven{state: state}, &identityWhen{state: state}, &identityThen{state: state}
}

func (g *identityGiven) there_is_no_existing_agent_identity() {
	g.state.t.Helper()
}

func (g *identityGiven) there_is_an_existing_agent_identity() {
	g.state.t.Helper()
	g.writeAgentID("agent-existing")
}

func (g *identityGiven) there_is_an_existing_wireguard_private_key() {
	g.state.t.Helper()
	g.state.assertions.NoError(os.MkdirAll(g.state.stateDir, 0o700))
	g.state.assertions.NoError(os.WriteFile(filepath.Join(g.state.stateDir, "wireguard.key"), []byte("private-key-existing\n"), 0o600))
}

func (w *identityWhen) the_agent_id_is_loaded_twice() {
	w.state.t.Helper()
	first, err := loadOrCreateAgentID(filepath.Join(w.state.stateDir, "agent-id"), "")
	w.state.assertions.NoError(err)
	second, err := loadOrCreateAgentID(filepath.Join(w.state.stateDir, "agent-id"), "")
	w.state.assertions.NoError(err)
	w.state.firstAgentID = first
	w.state.secondAgentID = second
}

func (w *identityWhen) the_agent_id_is_loaded_with_configured_agent_a() {
	w.state.t.Helper()
	id, err := loadOrCreateAgentID(filepath.Join(w.state.stateDir, "agent-id"), "agent-a")
	w.state.assertions.NoError(err)
	w.state.firstAgentID = id
}

func (w *identityWhen) the_agent_id_is_loaded_with_configured_agent_b() {
	w.state.t.Helper()
	id, err := loadOrCreateAgentID(filepath.Join(w.state.stateDir, "agent-id"), "agent-b")
	w.state.assertions.NoError(err)
	w.state.firstAgentID = id
}

func (w *identityWhen) the_wireguard_private_key_is_loaded_twice() {
	w.state.t.Helper()
	first, err := loadOrCreatePrivateKey(w.state.t.Context(), filepath.Join(w.state.stateDir, "wireguard.key"))
	w.state.assertions.NoError(err)
	second, err := loadOrCreatePrivateKey(w.state.t.Context(), filepath.Join(w.state.stateDir, "wireguard.key"))
	w.state.assertions.NoError(err)
	w.state.firstKey = first
	w.state.secondKey = second
}

func (th *identityThen) a_generated_agent_id_is_reused() *identityThen {
	th.state.t.Helper()
	th.state.assertions.NotEmpty(th.state.firstAgentID)
	th.state.assertions.True(strings.HasPrefix(th.state.firstAgentID, "agent-"))
	th.state.assertions.Equal(th.state.firstAgentID, th.state.secondAgentID)
	return th
}

func (th *identityThen) the_configured_agent_a_id_is_stored() *identityThen {
	th.state.t.Helper()
	th.state.assertions.Equal("agent-a", th.state.firstAgentID)
	data, err := os.ReadFile(filepath.Join(th.state.stateDir, "agent-id"))
	th.state.assertions.NoError(err)
	th.state.assertions.Equal("agent-a", strings.TrimSpace(string(data)))
	return th
}

func (th *identityThen) the_existing_agent_id_is_used() *identityThen {
	th.state.t.Helper()
	th.state.assertions.Equal("agent-existing", th.state.firstAgentID)
	return th
}

func (th *identityThen) the_existing_wireguard_private_key_is_reused() *identityThen {
	th.state.t.Helper()
	th.state.assertions.Equal("private-key-existing", th.state.firstKey)
	th.state.assertions.Equal("private-key-existing", th.state.secondKey)
	return th
}

func (th *identityThen) and() *identityThen {
	th.state.t.Helper()
	return th
}

func (g *identityGiven) writeAgentID(id string) {
	g.state.t.Helper()
	g.state.assertions.NoError(os.MkdirAll(g.state.stateDir, 0o700))
	g.state.assertions.NoError(os.WriteFile(filepath.Join(g.state.stateDir, "agent-id"), []byte(id+"\n"), 0o600))
}
