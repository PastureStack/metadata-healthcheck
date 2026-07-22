package main

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/PastureStack/metadata-healthcheck/internal/platformapi"
	"github.com/PastureStack/metadata-healthcheck/metadata"
	"github.com/PastureStack/metadata-healthcheck/pkg/haproxy"
)

const (
	serverPrefix  = "endpoint-"
	serverNameKey = "svname"
	statusKey     = "status"
)

func Poll(metadataURL string) error {
	reporter, err := platformapi.NewFromEnvironment()
	if err != nil {
		return err
	}

	metadataPoller := &metadata.Poller{}
	if err := metadataPoller.Init(metadataURL); err != nil {
		return err
	}

	configUpdater := &haproxy.Provider{
		Poller: metadataPoller,
	}
	if err = configUpdater.Start(); err != nil {
		return err
	}

	go configUpdater.Run()

	m := &Monitor{
		reporter:       reporter,
		reportedStatus: newStatusCache(time.Hour),
	}

	for stat := range m.getStats() {
		m.processStat(stat)
	}

	return nil
}

type Monitor struct {
	reporter       statusReporter
	reportedStatus *statusCache
}

type statusReporter interface {
	ReportStatus(context.Context, string, string, int64) error
}

type cachedStatus struct {
	value     string
	expiresAt time.Time
}

type statusCache struct {
	sync.Mutex
	ttl    time.Duration
	now    func() time.Time
	values map[string]cachedStatus
}

func newStatusCache(ttl time.Duration) *statusCache {
	return &statusCache{ttl: ttl, now: time.Now, values: make(map[string]cachedStatus)}
}

func (c *statusCache) get(key string) (string, bool) {
	c.Lock()
	defer c.Unlock()
	entry, ok := c.values[key]
	if !ok || !c.now().Before(entry.expiresAt) {
		delete(c.values, key)
		return "", false
	}
	return entry.value, true
}

func (c *statusCache) set(key, value string) {
	c.Lock()
	defer c.Unlock()
	c.values[key] = cachedStatus{value: value, expiresAt: c.now().Add(c.ttl)}
}

func (m *Monitor) getStats() <-chan haproxy.Stat {
	c := make(chan haproxy.Stat)
	go m.readStats(c)
	return c
}

func (m *Monitor) readStats(c chan<- haproxy.Stat) {
	defer close(c)

	count := 0

	h := &haproxy.Monitor{
		SocketPath: haproxy.HaproxySock,
	}

	for {
		// Sleep before reading so a restart cannot immediately flood the control API.
		time.Sleep(2 * time.Second)

		stats, err := h.Stats()
		currentCount := 0
		if err != nil {
			log.Printf("failed to read HAProxy stats: %v", err)
			continue
		}

		for _, stat := range stats {
			if strings.HasPrefix(stat[serverNameKey], serverPrefix) {
				currentCount++
				c <- stat
			}
		}

		if currentCount != count {
			count = currentCount
			log.Printf("monitoring %d backends", count)
		}

	}
}

func (m *Monitor) processStat(stat haproxy.Stat) {
	serverName := strings.TrimPrefix(stat[serverNameKey], serverPrefix)
	currentStatus := stat[statusKey]
	if serverName == "" || currentStatus == "" {
		return
	}

	previousStatus, _ := m.reportedStatus.get(serverName)
	if strings.HasPrefix(currentStatus, "UP ") {
		// do nothing on partial UP
		return
	}

	if currentStatus == "UP" && previousStatus != "UP" && previousStatus != "INIT" {
		currentStatus = "INIT"
	}

	update := true
	if previousStatus != currentStatus {
		err := m.reportStatus(serverName, currentStatus)
		if err != nil {
			log.Printf("failed to report status %s=%s: %v", serverName, currentStatus, err)
			update = false
		}
	}

	if update {
		m.reportedStatus.set(serverName, currentStatus)
	}
}

func (m *Monitor) reportStatus(serverName, currentStatus string) error {
	if err := m.reporter.ReportStatus(context.Background(), serverName, currentStatus, time.Now().Unix()); err != nil {
		return err
	}
	log.Printf("%s=%s", serverName, currentStatus)
	return nil
}
