package metadata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PastureStack/metadata-healthcheck/internal/logging"
	"github.com/PastureStack/metadata-healthcheck/types"
)

const maxResponseBytes = 8 << 20

type Service struct {
	UUID        string            `json:"uuid"`
	Containers  []Container       `json:"containers"`
	HealthCheck types.HealthCheck `json:"health_check"`
}

type Container struct {
	Name             string            `json:"name"`
	PrimaryIP        string            `json:"primary_ip"`
	ServiceName      string            `json:"service_name"`
	UUID             string            `json:"uuid"`
	State            string            `json:"state"`
	StartCount       int               `json:"start_count"`
	HealthCheckHosts []string          `json:"health_check_hosts"`
	HealthCheck      types.HealthCheck `json:"health_check"`
}

type Host struct {
	UUID string `json:"uuid"`
}

type HTTPFetcher struct {
	baseURL string
	client  *http.Client
}

type Poller struct {
	MetaFetcher Fetcher
}

type Fetcher interface {
	OnChange(intervalSeconds int, do func(string))
	GetServices() ([]Service, error)
	GetSelfHost() (Host, error)
	GetContainers() ([]Container, error)
}

func NewHTTPFetcher(baseURL string) (*HTTPFetcher, error) {
	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid metadata URL")
	}

	return &HTTPFetcher{
		baseURL: parsed.String(),
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (f *HTTPFetcher) get(path string, target any) error {
	request, err := http.NewRequest(http.MethodGet, f.baseURL+path, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")

	response, err := f.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		return fmt.Errorf("metadata request %s returned %s", path, response.Status)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read metadata response %s: %w", path, err)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("metadata response %s exceeds %d bytes", path, maxResponseBytes)
	}
	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("decode metadata response %s: %w", path, err)
	}
	return nil
}

func (f *HTTPFetcher) OnChange(intervalSeconds int, do func(string)) {
	version := "init"
	for {
		var next string
		path := fmt.Sprintf("/version?wait=true&value=%s&maxWait=%d", url.QueryEscape(version), intervalSeconds)
		if err := f.get(path, &next); err != nil {
			logging.Error.Printf("metadata version check failed: %v", err)
			time.Sleep(time.Duration(intervalSeconds) * time.Second)
			continue
		}
		if next != version {
			version = next
			do(version)
		}
	}
}

func (f *HTTPFetcher) GetServices() (services []Service, err error) {
	err = f.get("/services", &services)
	return
}

func (f *HTTPFetcher) GetContainers() (containers []Container, err error) {
	err = f.get("/containers", &containers)
	return
}

func (f *HTTPFetcher) GetSelfHost() (host Host, err error) {
	err = f.get("/self/host", &host)
	return
}

func (p *Poller) OnChange(do func(string)) {
	p.MetaFetcher.OnChange(5, do)
}

func (p *Poller) Init(metadataURL string) error {
	fetcher, err := NewHTTPFetcher(metadataURL)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(20 * time.Second)
	delay := time.Second
	for {
		var version string
		err = fetcher.get("/version", &version)
		if err == nil {
			p.MetaFetcher = fetcher
			return nil
		}
		if time.Now().Add(delay).After(deadline) {
			return fmt.Errorf("metadata service is unavailable: %w", err)
		}
		time.Sleep(delay)
		if delay < 4*time.Second {
			delay *= 2
		}
	}
}

func (p *Poller) GetHealthCheckServices() (services []types.Service, err error) {
	svcs, err := p.MetaFetcher.GetServices()
	if err != nil {
		return nil, err
	}
	cs, err := p.MetaFetcher.GetContainers()
	if err != nil {
		return nil, err
	}
	selfHost, err := p.MetaFetcher.GetSelfHost()
	if err != nil {
		return nil, err
	}
	var result []types.Service
	// process services
	for _, svc := range svcs {
		if svc.HealthCheck.Port == 0 {
			continue
		}
		var servers []types.Server
		for _, c := range svc.Containers {
			addServer(&c, &servers, &selfHost)
		}
		if len(servers) == 0 {
			continue
		}
		s := types.Service{
			UUID:        svc.UUID,
			HealthCheck: svc.HealthCheck,
			Servers:     servers,
		}
		result = append(result, s)
	}
	// process standalone containers
	for _, c := range cs {
		if c.ServiceName != "" || c.HealthCheck.Port == 0 {
			continue
		}
		var servers []types.Server
		addServer(&c, &servers, &selfHost)
		if len(servers) == 0 {
			continue
		}
		s := types.Service{
			UUID:        c.UUID,
			HealthCheck: c.HealthCheck,
			Servers:     servers,
		}
		result = append(result, s)
	}
	return result, nil
}

func addServer(c *Container, servers *[]types.Server, selfHost *Host) {
	if c.PrimaryIP == "" {
		return
	}
	if !(strings.EqualFold(c.State, "running") || strings.EqualFold(c.State, "starting")) {
		return
	}
	//only configure on the host
	// when container health checkers contain host id
	skipCheck := true
	for _, hostUUID := range c.HealthCheckHosts {
		if strings.EqualFold(selfHost.UUID, hostUUID) {
			skipCheck = false
			break
		}
	}
	if skipCheck {
		logging.Info.Printf("health check for container %q is not assigned to this host; skipping", c.Name)
		return
	}

	srvr := types.Server{
		UUID:    fmt.Sprintf("endpoint-%s_%s_%v", selfHost.UUID, c.UUID, c.StartCount),
		Address: c.PrimaryIP,
	}
	*servers = append(*servers, srvr)
}
