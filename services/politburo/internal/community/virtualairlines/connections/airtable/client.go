package airtable

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"infinite-experiment/politburo/internal/metrics"
)

const origin = "https://api.airtable.com"

// Client calls the Airtable REST API.
type Client struct {
	http    *http.Client
	baseURL string
	metrics *metrics.Registry
}

func NewClient(reg *metrics.Registry, timeout time.Duration) *Client {
	return &Client{
		http: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL: origin,
		metrics: reg,
	}
}

// WithBaseURL overrides the API origin (tests).
func (c *Client) WithBaseURL(base string) *Client {
	c.baseURL = strings.TrimRight(base, "/")
	return c
}

type Table struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Fields []Field `json:"fields"`
}

type Field struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type SchemaResponse struct {
	Tables []Table `json:"tables"`
}

type recordsResponse struct {
	Records []RawRecord `json:"records"`
	Offset  string      `json:"offset"`
}

// RawRecord is one Airtable row.
type RawRecord struct {
	ID     string         `json:"id"`
	Fields map[string]any `json:"fields"`
}

// FetchSchema returns tables and fields for a base.
func (c *Client) FetchSchema(ctx context.Context, token, baseID string) ([]Table, error) {
	path := fmt.Sprintf("/v0/meta/bases/%s/tables", url.PathEscape(baseID))
	var resp SchemaResponse
	if err := c.getJSON(ctx, token, "schema", path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Tables, nil
}

// ProbeTable performs a minimal list call to verify read access.
func (c *Client) ProbeTable(ctx context.Context, token, baseID, tableID string) error {
	q := url.Values{}
	q.Set("pageSize", "1")
	path := fmt.Sprintf("/v0/%s/%s", url.PathEscape(baseID), url.PathEscape(tableID))
	return c.getJSON(ctx, token, "records", path, q, &recordsResponse{})
}

// ListRecords loads one upstream page.
func (c *Client) ListRecords(ctx context.Context, token, baseID, tableID string, fieldNames []string, formula string, pageSize int, offset string) ([]RawRecord, string, error) {
	q := url.Values{}
	q.Set("pageSize", strconv.Itoa(pageSize))
	if offset != "" {
		q.Set("offset", offset)
	}
	if formula != "" {
		q.Set("filterByFormula", formula)
	}
	for _, name := range fieldNames {
		q.Add("fields[]", name)
	}
	path := fmt.Sprintf("/v0/%s/%s", url.PathEscape(baseID), url.PathEscape(tableID))
	var resp recordsResponse
	if err := c.getJSON(ctx, token, "records", path, q, &resp); err != nil {
		return nil, "", err
	}
	return resp.Records, resp.Offset, nil
}

// GetRecord loads one record by id.
func (c *Client) GetRecord(ctx context.Context, token, baseID, tableID, recordID string, fieldNames []string) (RawRecord, error) {
	q := url.Values{}
	for _, name := range fieldNames {
		q.Add("fields[]", name)
	}
	path := fmt.Sprintf("/v0/%s/%s/%s", url.PathEscape(baseID), url.PathEscape(tableID), url.PathEscape(recordID))
	var out RawRecord
	if err := c.getJSON(ctx, token, "records", path, q, &out); err != nil {
		return RawRecord{}, err
	}
	if out.ID == "" {
		out.ID = recordID
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, token, operation, path string, query url.Values, dest any) error {
	started := time.Now()
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		mapped := mapRequestErr(err)
		c.observe(operation, requestOutcome(err), time.Since(started))
		return mapped
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))

	if resp.StatusCode == http.StatusTooManyRequests {
		c.observe(operation, "throttled", time.Since(started))
		retry := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &ResponseError{Status: resp.StatusCode, RetryAfter: retry, Err: ErrThrottle}
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		c.observe(operation, "denied", time.Since(started))
		return &ResponseError{Status: resp.StatusCode, Err: ErrDenied}
	}
	if resp.StatusCode == http.StatusNotFound {
		c.observe(operation, "client_error", time.Since(started))
		return &ResponseError{Status: resp.StatusCode, Err: ErrNotFound}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		outcome := "server_error"
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			outcome = "client_error"
		}
		c.observe(operation, outcome, time.Since(started))
		return &ResponseError{Status: resp.StatusCode, Err: ErrUpstream}
	}
	if err := json.Unmarshal(body, dest); err != nil {
		c.observe(operation, "server_error", time.Since(started))
		return fmt.Errorf("%w: decode response", ErrUpstream)
	}
	c.observe(operation, "success", time.Since(started))
	return nil
}

func (c *Client) observe(operation, outcome string, d time.Duration) {
	if c.metrics == nil {
		return
	}
	c.metrics.ObserveDataConnectionsProvider("airtable", operation, outcome, d.Seconds())
}

func requestOutcome(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "canceled"
	}
	if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
		return "timeout"
	}
	return "server_error"
}

func mapRequestErr(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if ne, ok := err.(interface{ Timeout() bool }); ok && ne.Timeout() {
		return ErrTimeout
	}
	return ErrUpstream
}

func parseRetryAfter(raw string) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if secs, err := strconv.Atoi(raw); err == nil {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(raw); err == nil {
		return time.Until(t)
	}
	return 0
}

// EscapeFormulaLiteral escapes a string literal for filterByFormula.
func EscapeFormulaLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}

// FieldRef returns an Airtable formula field reference.
func FieldRef(fieldName string) string {
	return "{" + strings.ReplaceAll(fieldName, "}", "\\}") + "}"
}