package collector

import "github.com/prometheus/client_golang/prometheus"

var (
	descUp = prometheus.NewDesc(
		"adguard_up",
		"1 if the AdGuard Home instance is reachable, 0 otherwise.",
		[]string{"instance"}, nil,
	)
	descProtectionEnabled = prometheus.NewDesc(
		"adguard_protection_enabled",
		"1 if DNS protection is enabled, 0 otherwise.",
		[]string{"instance"}, nil,
	)
	descDNSQueries = prometheus.NewDesc(
		"adguard_dns_queries_total",
		"Total DNS queries in the current stats window (rolling, may decrease).",
		[]string{"instance"}, nil,
	)
	descBlockedFiltering = prometheus.NewDesc(
		"adguard_blocked_filtering_total",
		"Requests blocked by filter lists in the current stats window.",
		[]string{"instance"}, nil,
	)
	descBlockedSafebrowsing = prometheus.NewDesc(
		"adguard_blocked_safebrowsing_total",
		"Requests blocked by safebrowsing in the current stats window.",
		[]string{"instance"}, nil,
	)
	descBlockedParental = prometheus.NewDesc(
		"adguard_blocked_parental_total",
		"Requests blocked by parental control in the current stats window.",
		[]string{"instance"}, nil,
	)
	descEnforcedSafesearch = prometheus.NewDesc(
		"adguard_enforced_safesearch_total",
		"Safe-search enforcements in the current stats window.",
		[]string{"instance"}, nil,
	)
	descAvgProcessingTime = prometheus.NewDesc(
		"adguard_avg_processing_time_seconds",
		"Average DNS processing time in seconds.",
		[]string{"instance"}, nil,
	)
	descScrapeDuration = prometheus.NewDesc(
		"adguard_scrape_duration_seconds",
		"Duration of the last scrape for this instance in seconds.",
		[]string{"instance"}, nil,
	)
	descScrapeErrors = prometheus.NewDesc(
		"adguard_scrape_errors_total",
		"Total number of failed scrapes for this instance since process start.",
		[]string{"instance"}, nil,
	)
	descTopClients = prometheus.NewDesc(
		"adguard_top_clients",
		"Number of DNS queries from top clients.",
		[]string{"instance", "client"}, nil,
	)
	descTopQueriedDomains = prometheus.NewDesc(
		"adguard_top_queried_domains",
		"Number of queries for top queried domains.",
		[]string{"instance", "domain"}, nil,
	)
	descTopBlockedDomains = prometheus.NewDesc(
		"adguard_top_blocked_domains",
		"Number of blocked queries for top blocked domains.",
		[]string{"instance", "domain"}, nil,
	)
	descTopUpstreams = prometheus.NewDesc(
		"adguard_top_upstreams",
		"Number of responses from top upstreams.",
		[]string{"instance", "upstream"}, nil,
	)
	descTopUpstreamsAvgTime = prometheus.NewDesc(
		"adguard_top_upstreams_avg_time_seconds",
		"Average processing time in seconds for top upstreams.",
		[]string{"instance", "upstream"}, nil,
	)
)
