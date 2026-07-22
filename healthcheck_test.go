package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/PastureStack/metadata-healthcheck/pkg/haproxy"
)

type reportedEvent struct {
	id     string
	status string
}

type fakeReporter struct {
	events []reportedEvent
	err    error
}

func (f *fakeReporter) ReportStatus(_ context.Context, id, status string, _ int64) error {
	f.events = append(f.events, reportedEvent{id: id, status: status})
	return f.err
}

func TestProcessStatReportsTransitionsAndSuppressesDuplicates(t *testing.T) {
	reporter := &fakeReporter{}
	cache := newStatusCache(time.Hour)
	now := time.Unix(1000, 0)
	cache.now = func() time.Time { return now }
	monitor := &Monitor{reporter: reporter, reportedStatus: cache}

	monitor.processStat(haproxy.Stat{serverNameKey: "endpoint-host_container_1", statusKey: "UP 1/2"})
	monitor.processStat(haproxy.Stat{serverNameKey: "endpoint-host_container_1", statusKey: "DOWN"})
	monitor.processStat(haproxy.Stat{serverNameKey: "endpoint-host_container_1", statusKey: "DOWN"})
	monitor.processStat(haproxy.Stat{serverNameKey: "endpoint-host_container_1", statusKey: "UP"})

	want := []reportedEvent{{id: "host_container_1", status: "DOWN"}, {id: "host_container_1", status: "INIT"}}
	if len(reporter.events) != len(want) {
		t.Fatalf("events = %#v, want %#v", reporter.events, want)
	}
	for i := range want {
		if reporter.events[i] != want[i] {
			t.Fatalf("event %d = %#v, want %#v", i, reporter.events[i], want[i])
		}
	}
}

func TestProcessStatDoesNotCacheFailedReport(t *testing.T) {
	reporter := &fakeReporter{err: errors.New("offline")}
	monitor := &Monitor{reporter: reporter, reportedStatus: newStatusCache(time.Hour)}
	stat := haproxy.Stat{serverNameKey: "endpoint-id", statusKey: "DOWN"}

	monitor.processStat(stat)
	monitor.processStat(stat)
	if len(reporter.events) != 2 {
		t.Fatalf("attempts = %d, want 2", len(reporter.events))
	}
}

func TestStatusCacheExpires(t *testing.T) {
	cache := newStatusCache(time.Hour)
	now := time.Unix(1000, 0)
	cache.now = func() time.Time { return now }
	cache.set("id", "UP")
	if value, ok := cache.get("id"); !ok || value != "UP" {
		t.Fatalf("initial cache = %q, %t", value, ok)
	}
	now = now.Add(time.Hour)
	if _, ok := cache.get("id"); ok {
		t.Fatal("expected expired entry")
	}
}
