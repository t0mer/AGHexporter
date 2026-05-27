package instances

import (
	"fmt"
	"net/url"
)

// Instance holds the resolved configuration for a single AdGuard Home endpoint.
type Instance struct {
	Name     string
	URL      string
	Username string
	Password string
	SkipTLS  bool
}

// Validate returns an error if the Instance is not fully and correctly configured.
func (i Instance) Validate() error {
	if i.Name == "" {
		return fmt.Errorf("instance name is empty")
	}
	u, err := url.Parse(i.URL)
	if err != nil {
		return fmt.Errorf("instance %q: invalid URL: %w", i.Name, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("instance %q: URL scheme must be http or https, got %q", i.Name, u.Scheme)
	}
	if i.Username == "" {
		return fmt.Errorf("instance %q: username is required", i.Name)
	}
	if i.Password == "" {
		return fmt.Errorf("instance %q: password is required", i.Name)
	}
	return nil
}
