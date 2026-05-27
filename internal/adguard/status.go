package adguard

// GetStatus fetches /control/status from the AdGuard Home instance.
func (c *Client) GetStatus() (*ServerStatus, error) {
	var s ServerStatus
	if err := c.get("/status", &s); err != nil {
		return nil, err
	}
	return &s, nil
}
