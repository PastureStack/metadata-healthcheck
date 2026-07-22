package haproxy

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/PastureStack/metadata-healthcheck/internal/logging"
	"github.com/PastureStack/metadata-healthcheck/types"
)

var validProxyName = regexp.MustCompile(`^[A-Za-z0-9_.:-]{1,240}$`)

type Provider struct {
	builder *haproxyConfigBuilder
	Poller  types.MetadataPoller
}

type Backend struct {
	UUID            string
	Protocol        string
	ResponseTimeout int
	RequestLine     string
	Server          *Server
}

type Server struct {
	UUID    string
	Address string
	Port    int
	Config  string
}

type haproxyConfigBuilder struct {
	Name         string
	ReloadScript string
	Config       string
	Template     string
}

type Backends []*Backend

func (p *Provider) Start() error {
	logging.Info.Print("starting HAProxy listener")

	p.builder = &haproxyConfigBuilder{
		ReloadScript: "haproxy_reload.sh",
		Config:       "/etc/haproxy/haproxy_new.cfg",
		Template:     "/etc/haproxy/haproxy_template.cfg",
		Name:         "healthCheck",
	}
	if err := p.builder.write(nil); err != nil {
		return err
	}
	if err := p.applyHaproxyConfig(false); err != nil {
		return err
	}
	return nil
}

func (p *Provider) ApplyConfig(string) {
	logging.Info.Print("scheduling HAProxy configuration update")
	if err := p.applyConfig(); err != nil {
		logging.Error.Printf("failed to apply HAProxy configuration: %v", err)
	}
}

func (p *Provider) applyConfig() error {
	bes, err := p.buildConfig()
	if err != nil {
		return fmt.Errorf("build configuration: %w", err)
	}
	if err := p.builder.write(bes); err != nil {
		return fmt.Errorf("write configuration: %w", err)
	}
	return p.applyHaproxyConfig(true)
}

func (p *Provider) Run() {
	p.Poller.OnChange(p.ApplyConfig)
}

func (p *Provider) buildConfig() (backends Backends, err error) {
	svcs, err := p.Poller.GetHealthCheckServices()
	if err != nil {
		return nil, err
	}

	var bes Backends
	for _, svc := range svcs {
		hc := svc.HealthCheck
		if !validProxyName.MatchString(svc.UUID) {
			return nil, fmt.Errorf("invalid service identifier")
		}
		if hc.Port < 1 || hc.Port > 65535 {
			return nil, fmt.Errorf("service %q has invalid healthcheck port", svc.UUID)
		}
		if hc.Interval < 0 || hc.HealthyThreshold < 0 || hc.UnhealthyThreshold < 0 || hc.ResponseTimeout < 0 {
			return nil, fmt.Errorf("service %q has a negative healthcheck setting", svc.UUID)
		}
		if strings.ContainsAny(hc.RequestLine, "\r\n\x00") || len(hc.RequestLine) > 4096 {
			return nil, fmt.Errorf("service %q has an invalid HTTP request line", svc.UUID)
		}
		port := hc.Port
		config := ""
		if hc.Interval != 0 {
			config = fmt.Sprintf("inter %v", hc.Interval)
		}
		if hc.HealthyThreshold != 0 {
			config = fmt.Sprintf("%s rise %v", config, hc.HealthyThreshold)
		}
		if hc.UnhealthyThreshold != 0 {
			config = fmt.Sprintf("%s fall %v", config, hc.UnhealthyThreshold)
		}
		proto := "tcp"
		if hc.RequestLine != "" {
			proto = "http"
		}
		for _, c := range svc.Servers {
			if !validProxyName.MatchString(c.UUID) {
				return nil, fmt.Errorf("service %q has an invalid endpoint identifier", svc.UUID)
			}
			address := net.ParseIP(c.Address)
			if address == nil {
				return nil, fmt.Errorf("service %q has an invalid endpoint address", svc.UUID)
			}
			srvr := &Server{
				UUID:    c.UUID,
				Address: net.JoinHostPort(address.String(), strconv.Itoa(port)),
				Port:    port,
				Config:  config,
			}
			be := &Backend{
				UUID:            c.UUID,
				Protocol:        proto,
				ResponseTimeout: hc.ResponseTimeout,
				RequestLine:     hc.RequestLine,
				Server:          srvr,
			}
			bes = append(bes, be)
		}

	}
	sort.Sort(bes)

	return bes, nil
}

func (b *haproxyConfigBuilder) write(backends Backends) (err error) {
	t, err := template.ParseFiles(b.Template)
	if err != nil {
		return err
	}
	w, err := os.CreateTemp(filepath.Dir(b.Config), ".metadata-healthcheck-*.cfg")
	if err != nil {
		return err
	}
	temporaryPath := w.Name()
	defer os.Remove(temporaryPath)
	if err := w.Chmod(0o644); err != nil {
		_ = w.Close()
		return err
	}
	conf := make(map[string]interface{})
	conf["backends"] = backends
	if err = t.Execute(w, conf); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, b.Config)
}

func (p *Provider) applyHaproxyConfig(reload bool) error {
	if reload {
		return p.builder.exec("reload")
	}
	return p.builder.exec("start")
}

func (b *haproxyConfigBuilder) exec(action string) error {
	output, err := exec.Command(b.ReloadScript, "/etc/haproxy/haproxy.cfg", action).CombinedOutput()
	msg := fmt.Sprintf("%v -- %v", b.Name, string(output))
	if err != nil {
		logging.Error.Print(msg)
		return fmt.Errorf("error reloading %v: %v", msg, err)
	}
	logging.Info.Print(msg)
	return nil
}

func (s Backends) Len() int {
	return len(s)
}
func (s Backends) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s Backends) Less(i, j int) bool {
	return strings.Compare(s[i].UUID, s[j].UUID) > 0
}
