package platformapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReportStatus(t *testing.T) {
	var received serviceEvent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2-beta/serviceevents" {
			t.Errorf("path = %q", r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "access" || password != "secret" {
			t.Errorf("unexpected Basic Auth: %q %q %t", username, password, ok)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	client, err := New(server.URL+"/v1", "access", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ReportStatus(context.Background(), "host_container_1", "UP", 1234); err != nil {
		t.Fatal(err)
	}

	if received.HealthcheckUUID != "host_container_1" || received.ReportedHealth != "UP" || received.ExternalTimestamp != 1234 {
		t.Fatalf("unexpected event: %#v", received)
	}
}

func TestReportStatusRejectsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "sensitive diagnostic", http.StatusBadGateway)
	}))
	defer server.Close()

	client, err := New(server.URL, "access", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ReportStatus(context.Background(), "id", "DOWN", 1); err == nil {
		t.Fatal("expected an error")
	}
}

func TestNewRequiresAllSettings(t *testing.T) {
	if _, err := New("", "", ""); err == nil {
		t.Fatal("expected missing settings error")
	}
}

func TestNewRejectsUnsafeURLForms(t *testing.T) {
	for _, raw := range []string{
		"ftp://control.example",
		"http://user:password@control.example",
		"http://control.example?token=value",
		"http://control.example/#fragment",
	} {
		if _, err := New(raw, "access", "secret"); err == nil {
			t.Errorf("New(%q) succeeded", raw)
		}
	}
}

func TestReportStatusDoesNotFollowRedirects(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		redirected = true
	}))
	defer target.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()

	client, err := New(server.URL, "access", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.ReportStatus(context.Background(), "id", "UP", 1); err == nil {
		t.Fatal("expected redirect response to fail")
	}
	if redirected {
		t.Fatal("control API credentials were sent through a redirect")
	}
}
