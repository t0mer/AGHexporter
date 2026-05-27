package adguard

// ServerStatus maps /control/status response (swagger: ServerStatus).
type ServerStatus struct {
	ProtectionEnabled bool   `json:"protection_enabled"`
	Running           bool   `json:"running"`
	Version           string `json:"version"`
}

// Stats maps /control/stats response (swagger: Stats).
// All time fields are already in seconds per swagger.yml.
type Stats struct {
	NumDNSQueries           int                  `json:"num_dns_queries"`
	NumBlockedFiltering     int                  `json:"num_blocked_filtering"`
	NumReplacedSafebrowsing int                  `json:"num_replaced_safebrowsing"`
	NumReplacedSafesearch   int                  `json:"num_replaced_safesearch"`
	NumReplacedParental     int                  `json:"num_replaced_parental"`
	AvgProcessingTime       float64              `json:"avg_processing_time"`
	TopQueriedDomains       []map[string]float64 `json:"top_queried_domains"`
	TopClients              []map[string]float64 `json:"top_clients"`
	TopBlockedDomains       []map[string]float64 `json:"top_blocked_domains"`
	TopUpstreamsResponses   []map[string]float64 `json:"top_upstreams_responses"`
	TopUpstreamsAvgTime     []map[string]float64 `json:"top_upstreams_avg_time"`
}
