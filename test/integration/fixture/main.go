package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type serviceEvent struct {
	HealthcheckUUID   string `json:"healthcheckUuid"`
	ReportedHealth    string `json:"reportedHealth"`
	ExternalTimestamp int64  `json:"externalTimestamp"`
}

type fixture struct {
	targetAddress string
	mu            sync.Mutex
	events        []serviceEvent
}

func main() {
	targetAddress, err := firstNonLoopbackIPv4()
	if err != nil {
		log.Fatal(err)
	}

	f := &fixture{targetAddress: targetAddress}
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		server := &http.Server{
			Addr:              ":18080",
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
		}
		log.Fatal(server.ListenAndServe())
	}()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           f.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("fixture target=%s", targetAddress)
	log.Fatal(server.ListenAndServe())
}

func firstNonLoopbackIPv4() (string, error) {
	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addresses {
		ip, _, err := net.ParseCIDR(address.String())
		if err == nil && ip.To4() != nil && !ip.IsLoopback() {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no non-loopback IPv4 address")
}

func (f *fixture) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/2015-12-19/version", func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("wait") == "true" && request.URL.Query().Get("value") == "fixture-version-1" {
			time.Sleep(500 * time.Millisecond)
		}
		writeJSON(w, "fixture-version-1")
	})
	mux.HandleFunc("/2015-12-19/services", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []any{map[string]any{
			"uuid": "service-fixture",
			"health_check": map[string]any{
				"port": 18080, "interval": 250, "healthy_threshold": 1,
				"unhealthy_threshold": 1, "request_line": "GET /ready HTTP/1.0",
				"response_timeout": 1000,
			},
			"containers": []any{f.container()},
		}})
	})
	mux.HandleFunc("/2015-12-19/containers", func(w http.ResponseWriter, _ *http.Request) {
		container := f.container()
		container["service_name"] = "fixture"
		writeJSON(w, []any{container})
	})
	mux.HandleFunc("/2015-12-19/self/host", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"uuid": "host-fixture"})
	})
	mux.HandleFunc("/v2-beta/serviceevents", f.recordEvent)
	mux.HandleFunc("/events", func(w http.ResponseWriter, _ *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		writeJSON(w, f.events)
	})
	return mux
}

func (f *fixture) container() map[string]any {
	return map[string]any{
		"uuid": "container-fixture", "name": "fixture", "primary_ip": f.targetAddress,
		"state": "running", "start_count": 1, "health_check_hosts": []string{"host-fixture"},
	}
}

func (f *fixture) recordEvent(w http.ResponseWriter, request *http.Request) {
	username, password, ok := request.BasicAuth()
	if request.Method != http.MethodPost || !ok || username != "fixture-access" || password != "fixture-secret" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	defer request.Body.Close()
	var event serviceEvent
	decoder := json.NewDecoder(http.MaxBytesReader(w, request.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil || event.HealthcheckUUID != "host-fixture_container-fixture_1" ||
		event.ReportedHealth == "" || event.ExternalTimestamp <= 0 {
		http.Error(w, "invalid event", http.StatusBadRequest)
		return
	}
	f.mu.Lock()
	f.events = append(f.events, event)
	f.mu.Unlock()
	w.WriteHeader(http.StatusCreated)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
