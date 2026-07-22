package haproxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"strings"
)

const HaproxySock = "/var/run/haproxy.sock"

type Stat map[string]string

type Monitor struct {
	SocketPath string
}

func (h *Monitor) Stats() ([]Stat, error) {
	lines, err := h.runCommand("show stat")
	if err != nil {
		return nil, err
	}
	return parseStats(lines)
}

func parseStats(lines []string) ([]Stat, error) {
	stats := []Stat{}

	if len(lines) == 0 || !strings.HasPrefix(lines[0], "# ") {
		return nil, fmt.Errorf("Failed to find stats")
	}

	keys := strings.Split(strings.TrimPrefix(lines[0], "# "), ",")

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}

		values := strings.Split(line, ",")
		if len(keys) != len(values) {
			return nil, fmt.Errorf("invalid HAProxy stat field count: got %d, want %d", len(values), len(keys))
		}

		stat := Stat{}

		for i := 0; i < len(values); i++ {
			stat[keys[i]] = values[i]
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

func (h *Monitor) runCommand(cmd string) ([]string, error) {
	conn, err := net.Dial("unix", h.SocketPath)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	_, err = io.WriteString(conn, cmd+"\n")
	if err != nil {
		return nil, err
	}

	result := []string{}
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		result = append(result, scanner.Text())
	}

	return result, scanner.Err()
}
