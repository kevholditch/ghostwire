package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Enroll(ctx context.Context, req protocol.EnrollRequest) (protocol.EnrollResponse, error) {
	var resp protocol.EnrollResponse
	if err := c.post(ctx, "/v1/agents/enroll", req, http.StatusOK, &resp); err != nil {
		return protocol.EnrollResponse{}, err
	}
	return resp, nil
}

func (c *Client) Heartbeat(ctx context.Context, req protocol.HeartbeatRequest) error {
	return c.post(ctx, "/v1/agents/heartbeat", req, http.StatusNoContent, nil)
}

func (c *Client) Peers(ctx context.Context, agentID string) (protocol.PeersResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/agents/"+agentID+"/peers", nil)
	if err != nil {
		return protocol.PeersResponse{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return protocol.PeersResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return protocol.PeersResponse{}, fmt.Errorf("peers status %d", resp.StatusCode)
	}
	var peers protocol.PeersResponse
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return protocol.PeersResponse{}, err
	}
	return peers, nil
}

func (c *Client) post(ctx context.Context, path string, body any, wantStatus int, out any) error {
	buf := bytes.Buffer{}
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		return fmt.Errorf("%s status %d", path, resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}
