package adguard_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/t0mer/AGHexporter/internal/adguard"
	"github.com/t0mer/AGHexporter/internal/instances"
)

// stubADH creates an httptest server serving /control/status and /control/stats.
// It enforces Basic Auth with username "admin" / password "secret".
func stubADH(t *testing.T, status adguard.ServerStatus, stats adguard.Stats) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/control/status", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "admin" || p != "secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	})
	mux.HandleFunc("/control/stats", func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "admin" || p != "secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(stats)
	})
	return httptest.NewServer(mux)
}

func newTestClient(t *testing.T, srv *httptest.Server) *adguard.Client {
	t.Helper()
	return adguard.NewClient(instances.Instance{
		URL:      srv.URL,
		Username: "admin",
		Password: "secret",
		SkipTLS:  false,
	})
}

func TestGetStatus_OK(t *testing.T) {
	srv := stubADH(t, adguard.ServerStatus{ProtectionEnabled: true, Running: true}, adguard.Stats{})
	defer srv.Close()

	status, err := newTestClient(t, srv).GetStatus()
	if err != nil {
		t.Fatal(err)
	}
	if !status.ProtectionEnabled {
		t.Error("want ProtectionEnabled=true")
	}
	if !status.Running {
		t.Error("want Running=true")
	}
}

func TestGetStats_OK(t *testing.T) {
	want := adguard.Stats{
		NumDNSQueries:       1000,
		NumBlockedFiltering: 100,
		AvgProcessingTime:   0.001,
		TopClients:          []map[string]float64{{"192.168.1.1": 500}},
	}
	srv := stubADH(t, adguard.ServerStatus{}, want)
	defer srv.Close()

	got, err := newTestClient(t, srv).GetStats()
	if err != nil {
		t.Fatal(err)
	}
	if got.NumDNSQueries != 1000 {
		t.Errorf("NumDNSQueries: want 1000, got %d", got.NumDNSQueries)
	}
	if len(got.TopClients) != 1 {
		t.Fatalf("TopClients: want 1 entry, got %d", len(got.TopClients))
	}
	if got.TopClients[0]["192.168.1.1"] != 500 {
		t.Errorf("TopClients[0][192.168.1.1]: want 500, got %v", got.TopClients[0]["192.168.1.1"])
	}
}

func TestGetStatus_WrongCredentials(t *testing.T) {
	srv := stubADH(t, adguard.ServerStatus{}, adguard.Stats{})
	defer srv.Close()

	client := adguard.NewClient(instances.Instance{
		URL:      srv.URL,
		Username: "wrong",
		Password: "wrong",
	})
	_, err := client.GetStatus()
	if err == nil {
		t.Fatal("want error for wrong credentials, got nil")
	}
}

func TestGetStatus_AuthHeaderPresent(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(adguard.ServerStatus{})
	}))
	defer srv.Close()

	client := adguard.NewClient(instances.Instance{URL: srv.URL, Username: "u", Password: "p"})
	_, _ = client.GetStatus()

	if gotAuth == "" {
		t.Error("want Authorization header on every request, got empty")
	}
}
