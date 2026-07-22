package metadata

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PastureStack/metadata-healthcheck/types"
)

func TestPollerBuildsHealthcheckServices(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/2015-12-19/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`"7"`))
	})
	mux.HandleFunc("/2015-12-19/services", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{
			"uuid":"service-1",
			"health_check":{"port":8080,"interval":2000,"healthy_threshold":2,"unhealthy_threshold":3},
			"containers":[{"uuid":"container-1","name":"api","primary_ip":"10.0.0.8","state":"running","start_count":4,"health_check_hosts":["host-1"]}]
		}]`))
	})
	mux.HandleFunc("/2015-12-19/containers", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"uuid":"container-1","service_name":"api","primary_ip":"10.0.0.8","state":"running"},
			{"uuid":"standalone-1","primary_ip":"10.0.0.9","state":"starting","start_count":1,"health_check_hosts":["host-1"],"health_check":{"port":9090}}
		]`))
	})
	mux.HandleFunc("/2015-12-19/self/host", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"uuid":"host-1"}`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	poller := &Poller{}
	if err := poller.Init(server.URL + "/2015-12-19"); err != nil {
		t.Fatal(err)
	}
	services, err := poller.GetHealthCheckServices()
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 {
		t.Fatalf("services = %d, want 2", len(services))
	}
	if got := services[0].Servers[0]; got.UUID != "endpoint-host-1_container-1_4" || got.Address != "10.0.0.8" {
		t.Fatalf("unexpected service server: %#v", got)
	}
	if got := services[1]; got.UUID != "standalone-1" || got.HealthCheck.Port != 9090 {
		t.Fatalf("unexpected standalone service: %#v", got)
	}
}

func TestAddServerSkipsUnassignedOrInvalidContainers(t *testing.T) {
	host := Host{UUID: "host-1"}
	servers := []types.Server{}
	for _, container := range []Container{
		{UUID: "no-ip", State: "running", HealthCheckHosts: []string{"host-1"}},
		{UUID: "stopped", PrimaryIP: "10.0.0.2", State: "stopped", HealthCheckHosts: []string{"host-1"}},
		{UUID: "other-host", PrimaryIP: "10.0.0.3", State: "running", HealthCheckHosts: []string{"host-2"}},
	} {
		addServer(&container, &servers, &host)
	}
	if len(servers) != 0 {
		t.Fatalf("servers = %#v, want none", servers)
	}
}

func TestFetcherRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]` + strings.Repeat(" ", maxResponseBytes)))
	}))
	defer server.Close()

	fetcher, err := NewHTTPFetcher(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fetcher.GetServices(); err == nil {
		t.Fatal("expected oversized response error")
	}
}

func TestNewHTTPFetcherRejectsRelativeURL(t *testing.T) {
	if _, err := NewHTTPFetcher("metadata/2015-12-19"); err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func TestNewHTTPFetcherRejectsUnsafeURLForms(t *testing.T) {
	for _, raw := range []string{
		"ftp://metadata.example/2015-12-19",
		"http://user:password@metadata.example/2015-12-19",
		"http://metadata.example/2015-12-19?token=value",
		"http://metadata.example/2015-12-19#fragment",
	} {
		if _, err := NewHTTPFetcher(raw); err == nil {
			t.Errorf("NewHTTPFetcher(%q) succeeded", raw)
		}
	}
}
