package adguard

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/t0mer/AGHexporter/internal/instances"
)

const requestTimeout = 10 * time.Second

// Client is an HTTP client for a single AdGuard Home instance.
type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
}

// NewClient constructs a Client for the given instance.
func NewClient(inst instances.Instance) *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: inst.SkipTLS}, //nolint:gosec
	}
	return &Client{
		baseURL:  inst.URL,
		username: inst.Username,
		password: inst.Password,
		http:     &http.Client{Timeout: requestTimeout, Transport: transport},
	}
}

// get performs an authenticated GET to baseURL+"/control"+path and JSON-decodes into target.
func (c *Client) get(path string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/control"+path, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.SetBasicAuth(c.username, c.password)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status %d from %s", resp.StatusCode, req.URL)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
