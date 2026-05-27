package adguard

// GetStats fetches /control/stats from the AdGuard Home instance.
func (c *Client) GetStats() (*Stats, error) {
	var s Stats
	if err := c.get("/stats", &s); err != nil {
		return nil, err
	}
	return &s, nil
}
