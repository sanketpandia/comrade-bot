package infiniteflight

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LiveOrganization is a row from GET /live/organizations.
type LiveOrganization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role int    `json:"role"`
}

// LiveClient calls Infinite Flight Public API v3 with a user OAuth access token.
type LiveClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewLiveClient builds a client for the Live API base URL (typically /public/v3).
func NewLiveClient(baseURL string, timeout time.Duration) (*LiveClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("Live API base URL is empty")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &LiveClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// ListOrganizations returns organizations for the bearer access token.
func (c *LiveClient) ListOrganizations(ctx context.Context, accessToken string) ([]LiveOrganization, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("access token is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/live/organizations", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list Infinite Flight organizations: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read Infinite Flight organizations response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list Infinite Flight organizations: HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		ErrorCode int                `json:"errorCode"`
		Result    []LiveOrganization `json:"result"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, fmt.Errorf("decode Infinite Flight organizations response: %w", err)
	}
	if parsed.ErrorCode != upstreamErrorCodeOK {
		return nil, fmt.Errorf("list Infinite Flight organizations: errorCode %d", parsed.ErrorCode)
	}
	return parsed.Result, nil
}

// GetOrganization returns one organization for the bearer access token.
func (c *LiveClient) GetOrganization(ctx context.Context, accessToken, organizationID string) (LiveOrganization, error) {
	accessToken = strings.TrimSpace(accessToken)
	organizationID = strings.TrimSpace(organizationID)
	if accessToken == "" {
		return LiveOrganization{}, fmt.Errorf("access token is required")
	}
	if organizationID == "" {
		return LiveOrganization{}, fmt.Errorf("organization id is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/live/organizations/"+organizationID, nil)
	if err != nil {
		return LiveOrganization{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return LiveOrganization{}, fmt.Errorf("get Infinite Flight organization: %w", err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return LiveOrganization{}, fmt.Errorf("read Infinite Flight organization response: %w", err)
	}
	if resp.StatusCode == http.StatusNotFound {
		return LiveOrganization{}, ErrLiveOrganizationNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return LiveOrganization{}, fmt.Errorf("get Infinite Flight organization: HTTP %d", resp.StatusCode)
	}
	var parsed struct {
		ErrorCode int `json:"errorCode"`
		Result    struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Role int    `json:"role"`
		} `json:"result"`
	}
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return LiveOrganization{}, fmt.Errorf("decode Infinite Flight organization response: %w", err)
	}
	if parsed.ErrorCode != upstreamErrorCodeOK {
		return LiveOrganization{}, fmt.Errorf("get Infinite Flight organization: errorCode %d", parsed.ErrorCode)
	}
	return LiveOrganization{
		ID:   parsed.Result.ID,
		Name: parsed.Result.Name,
		Role: parsed.Result.Role,
	}, nil
}
