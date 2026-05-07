package controlplane

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
)

type Server struct {
	registry *Registry
	now      func() time.Time
	mux      *http.ServeMux
}

func NewServer(registry *Registry, now func() time.Time) *Server {
	if now == nil {
		now = time.Now
	}
	server := &Server{registry: registry, now: now, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealthz)
	s.mux.HandleFunc("/v1/nodes", s.handleNodes)
	s.mux.HandleFunc("/v1/agents/enroll", s.handleEnroll)
	s.mux.HandleFunc("/v1/agents/heartbeat", s.handleHeartbeat)
	s.mux.HandleFunc("/v1/agents/", s.handleAgent)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req protocol.EnrollRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed json")
		return
	}
	resp, err := s.registry.Enroll(req, s.now())
	if err != nil {
		s.writeRegistryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req protocol.HeartbeatRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed json")
		return
	}
	if err := s.registry.Heartbeat(req, s.now()); err != nil {
		s.writeRegistryError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, s.registry.Nodes(s.now()))
}

func (s *Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/v1/agents/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "peers" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	peers, err := s.registry.Peers(parts[0], s.now())
	if err != nil {
		s.writeRegistryError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, peers)
}

func (s *Server) writeRegistryError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, ErrAgentNotFound):
		writeError(w, http.StatusNotFound, "agent not found")
	case errors.Is(err, ErrInvalidAgent):
		writeError(w, http.StatusBadRequest, "invalid agent")
	case errors.Is(err, ErrCIDRExhausted):
		writeError(w, http.StatusServiceUnavailable, "cidr exhausted")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, protocol.ErrorResponse{Error: msg})
}
