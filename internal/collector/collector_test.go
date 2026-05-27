package collector_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/t0mer/AGHexporter/internal/adguard"
	"github.com/t0mer/AGHexporter/internal/collector"
	"github.com/t0mer/AGHexporter/internal/instances"
)

// stubADH creates an httptest server that serves canned status+stats responses.
func stubADH(t *testing.T, status adguard.ServerStatus, stats adguard.Stats) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/control/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	})
	mux.HandleFunc("/control/stats", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats)
	})
	return httptest.NewServer(mux)
}

func TestCollect_AllScalarMetricsPresent(t *testing.T) {
	srv := stubADH(t,
		adguard.ServerStatus{ProtectionEnabled: true, Running: true},
		adguard.Stats{
			NumDNSQueries:       500,
			NumBlockedFiltering: 50,
			AvgProcessingTime:   0.001,
			TopClients:          []map[string]float64{{"192.168.0.1": 100}},
			TopQueriedDomains:   []map[string]float64{{"example.com": 200}},
			TopBlockedDomains:   []map[string]float64{{"ads.com": 30}},
		},
	)
	defer srv.Close()

	inst := instances.Instance{Name: "test-instance", URL: srv.URL, Username: "u", Password: "p"}
	coll := collector.New([]instances.Instance{inst})

	reg := prometheus.NewRegistry()
	reg.MustRegister(coll)

	for _, name := range []string{
		"adguard_up",
		"adguard_protection_enabled",
		"adguard_dns_queries_total",
		"adguard_blocked_filtering_total",
		"adguard_blocked_safebrowsing_total",
		"adguard_blocked_parental_total",
		"adguard_enforced_safesearch_total",
		"adguard_avg_processing_time_seconds",
		"adguard_scrape_duration_seconds",
		"adguard_scrape_errors_total",
		"adguard_top_clients",
		"adguard_top_queried_domains",
		"adguard_top_blocked_domains",
	} {
		count := testutil.CollectAndCount(coll, name)
		if count == 0 {
			t.Errorf("metric %q not collected", name)
		}
	}
}

func TestCollect_UpIsOne_WhenReachable(t *testing.T) {
	srv := stubADH(t, adguard.ServerStatus{ProtectionEnabled: true}, adguard.Stats{})
	defer srv.Close()

	inst := instances.Instance{Name: "online", URL: srv.URL, Username: "u", Password: "p"}
	coll := collector.New([]instances.Instance{inst})

	want := `
# HELP adguard_up 1 if the AdGuard Home instance is reachable, 0 otherwise.
# TYPE adguard_up gauge
adguard_up{instance="online"} 1
`
	if err := testutil.CollectAndCompare(coll, strings.NewReader(want), "adguard_up"); err != nil {
		t.Error(err)
	}
}

func TestCollect_UpIsZero_WhenUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	inst := instances.Instance{Name: "offline", URL: srv.URL, Username: "u", Password: "p"}
	coll := collector.New([]instances.Instance{inst})

	want := `
# HELP adguard_up 1 if the AdGuard Home instance is reachable, 0 otherwise.
# TYPE adguard_up gauge
adguard_up{instance="offline"} 0
`
	if err := testutil.CollectAndCompare(coll, strings.NewReader(want), "adguard_up"); err != nil {
		t.Error(err)
	}
}

func TestCollect_TwoInstances_OneDown(t *testing.T) {
	good := stubADH(t, adguard.ServerStatus{ProtectionEnabled: true}, adguard.Stats{})
	defer good.Close()

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	bad.Close()

	insts := []instances.Instance{
		{Name: "good", URL: good.URL, Username: "u", Password: "p"},
		{Name: "bad", URL: bad.URL, Username: "u", Password: "p"},
	}
	coll := collector.New(insts)

	reg := prometheus.NewRegistry()
	reg.MustRegister(coll)

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}

	upByInstance := make(map[string]float64)
	for _, mf := range mfs {
		if mf.GetName() != "adguard_up" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "instance" {
					upByInstance[lp.GetValue()] = m.GetGauge().GetValue()
				}
			}
		}
	}

	if upByInstance["good"] != 1.0 {
		t.Errorf("good instance: want adguard_up=1, got %v", upByInstance["good"])
	}
	if upByInstance["bad"] != 0.0 {
		t.Errorf("bad instance: want adguard_up=0, got %v", upByInstance["bad"])
	}
}

func TestCollect_TopClientsHaveInstanceLabel(t *testing.T) {
	srv := stubADH(t,
		adguard.ServerStatus{},
		adguard.Stats{TopClients: []map[string]float64{{"10.0.0.1": 42}}},
	)
	defer srv.Close()

	inst := instances.Instance{Name: "labeled", URL: srv.URL, Username: "u", Password: "p"}
	coll := collector.New([]instances.Instance{inst})

	want := `
# HELP adguard_top_clients Number of DNS queries from top clients.
# TYPE adguard_top_clients gauge
adguard_top_clients{client="10.0.0.1",instance="labeled"} 42
`
	if err := testutil.CollectAndCompare(coll, strings.NewReader(want), "adguard_top_clients"); err != nil {
		t.Error(err)
	}
}
