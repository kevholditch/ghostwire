package controlapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kevholditch/ghostwire/pkg/protocol"
)

type Client struct {
	baseURL  string
	apiToken string
	http     *http.Client
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" && e.Message != "" {
		return e.Code + ": " + e.Message
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("request failed with status %d", e.StatusCode)
}

func NewClient(baseURL, apiToken string) *Client {
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiToken: apiToken,
		http:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) ListNodes(ctx context.Context) (protocol.NodesResponse, error) {
	var nodes protocol.NodesResponse
	if err := c.get(ctx, []string{"v1", "nodes"}, &nodes); err != nil {
		return protocol.NodesResponse{}, err
	}
	return nodes, nil
}

func (c *Client) GetNode(ctx context.Context, nodeID string) (protocol.Node, error) {
	var node protocol.Node
	if err := c.get(ctx, []string{"v1", "nodes", nodeID}, &node); err != nil {
		return protocol.Node{}, err
	}
	return node, nil
}

func (c *Client) ListNodePeers(ctx context.Context, nodeID string) (protocol.NodesResponse, error) {
	var nodes protocol.NodesResponse
	if err := c.get(ctx, []string{"v1", "nodes", nodeID, "peers"}, &nodes); err != nil {
		return protocol.NodesResponse{}, err
	}
	return nodes, nil
}

func (c *Client) get(ctx context.Context, path []string, out any) error {
	endpoint := c.url(path...)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("authorization", "Bearer "+c.apiToken)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return decodeAPIError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		closeErr := resp.Body.Close()
		if closeErr != nil {
			return fmt.Errorf("decode response: %w; close response body: %v", err, closeErr)
		}
		return fmt.Errorf("decode response: %w", err)
	}
	if err := resp.Body.Close(); err != nil {
		return fmt.Errorf("close response body: %w", err)
	}
	return nil
}

func (c *Client) url(path ...string) string {
	escaped := make([]string, 0, len(path))
	for _, part := range path {
		escaped = append(escaped, url.PathEscape(part))
	}
	return c.baseURL + "/" + strings.Join(escaped, "/")
}

func decodeAPIError(resp *http.Response) error {
	var apiErr protocol.ErrorResponse
	decodeErr := json.NewDecoder(resp.Body).Decode(&apiErr)
	closeErr := resp.Body.Close()
	if decodeErr != nil {
		if closeErr != nil {
			return fmt.Errorf("request failed with status %d; decode error: %w; close response body: %v", resp.StatusCode, decodeErr, closeErr)
		}
		return fmt.Errorf("request failed with status %d; decode error: %w", resp.StatusCode, decodeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("request failed with status %d; close response body: %w", resp.StatusCode, closeErr)
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Code:       apiErr.Code,
		Message:    apiErr.Message,
	}
}
