package haproxy

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PastureStack/metadata-healthcheck/types"
)

type fakePoller struct {
	services []types.Service
	err      error
}

func (f *fakePoller) GetHealthCheckServices() ([]types.Service, error) {
	return f.services, f.err
}

func (f *fakePoller) OnChange(func(string)) {}
func (f *fakePoller) Init(string) error     { return nil }

func TestBuildConfigCreatesTCPAndHTTPBackends(t *testing.T) {
	provider := &Provider{Poller: &fakePoller{services: []types.Service{
		{
			UUID:        "service-1",
			HealthCheck: types.HealthCheck{Port: 8080, Interval: 2000, HealthyThreshold: 2, UnhealthyThreshold: 3},
			Servers:     []types.Server{{UUID: "endpoint-b", Address: "10.0.0.2"}},
		},
		{
			UUID:        "service-2",
			HealthCheck: types.HealthCheck{Port: 9090, RequestLine: "GET /ready HTTP/1.0", ResponseTimeout: 1500},
			Servers:     []types.Server{{UUID: "endpoint-a", Address: "10.0.0.3"}},
		},
	}}}

	backends, err := provider.buildConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(backends) != 2 {
		t.Fatalf("backends = %d", len(backends))
	}
	if backends[0].UUID != "endpoint-b" || backends[0].Protocol != "tcp" || !strings.Contains(backends[0].Server.Config, "rise 2") {
		t.Fatalf("unexpected TCP backend: %#v", backends[0])
	}
	if backends[1].UUID != "endpoint-a" || backends[1].Protocol != "http" || backends[1].RequestLine == "" {
		t.Fatalf("unexpected HTTP backend: %#v", backends[1])
	}
}

func TestConfigBuilderWritesAndClosesOutput(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "template.cfg")
	configPath := filepath.Join(directory, "generated.cfg")
	if err := os.WriteFile(templatePath, []byte(`{{range .backends}}{{.UUID}}={{.Server.Address}}:{{.Server.Port}}{{end}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	builder := &haproxyConfigBuilder{Template: templatePath, Config: configPath}
	if err := builder.write(Backends{&Backend{UUID: "endpoint-a", Server: &Server{Address: "10.0.0.3", Port: 80}}}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(content); got != "endpoint-a=10.0.0.3:80" {
		t.Fatalf("generated config = %q", got)
	}
	if err := os.Rename(configPath, configPath+".moved"); err != nil {
		t.Fatalf("output file appears open: %v", err)
	}
}

func TestParseStats(t *testing.T) {
	stats, err := parseStats([]string{"# pxname,svname,status", "backend,endpoint-a,UP", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) != 1 || stats[0]["svname"] != "endpoint-a" || stats[0]["status"] != "UP" {
		t.Fatalf("unexpected stats: %#v", stats)
	}

	if _, err := parseStats([]string{"# a,b", "only-one"}); err == nil {
		t.Fatal("expected invalid field count error")
	}
}

func TestBuildConfigRejectsConfigurationInjection(t *testing.T) {
	for name, service := range map[string]types.Service{
		"request-line": {
			UUID:        "service-1",
			HealthCheck: types.HealthCheck{Port: 8080, RequestLine: "GET /ready HTTP/1.0\nserver injected 127.0.0.1:1"},
			Servers:     []types.Server{{UUID: "endpoint-1", Address: "192.0.2.10"}},
		},
		"address": {
			UUID:        "service-1",
			HealthCheck: types.HealthCheck{Port: 8080},
			Servers:     []types.Server{{UUID: "endpoint-1", Address: "192.0.2.10 extra"}},
		},
		"identifier": {
			UUID:        "service-1\nbackend injected",
			HealthCheck: types.HealthCheck{Port: 8080},
			Servers:     []types.Server{{UUID: "endpoint-1", Address: "192.0.2.10"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			provider := &Provider{Poller: &fakePoller{services: []types.Service{service}}}
			if _, err := provider.buildConfig(); err == nil {
				t.Fatal("expected invalid configuration to be rejected")
			}
		})
	}
}

func TestApplyConfigStopsWhenMetadataFails(t *testing.T) {
	provider := &Provider{
		Poller: &fakePoller{err: errors.New("metadata unavailable")},
		builder: &haproxyConfigBuilder{
			ReloadScript: filepath.Join(t.TempDir(), "must-not-run"),
			Config:       filepath.Join(t.TempDir(), "must-not-write"),
			Template:     filepath.Join(t.TempDir(), "missing-template"),
		},
	}
	if err := provider.applyConfig(); err == nil || !strings.Contains(err.Error(), "metadata unavailable") {
		t.Fatalf("applyConfig error = %v", err)
	}
}
