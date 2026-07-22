package types

type HealthCheck struct {
	HealthyThreshold   int    `json:"healthy_threshold"`
	Interval           int    `json:"interval"`
	Port               int    `json:"port"`
	RequestLine        string `json:"request_line"`
	ResponseTimeout    int    `json:"response_timeout"`
	UnhealthyThreshold int    `json:"unhealthy_threshold"`
}

type Service struct {
	UUID        string
	HealthCheck HealthCheck
	Servers     []Server
}

type Server struct {
	UUID    string
	Address string
}

type MetadataPoller interface {
	GetHealthCheckServices() (services []Service, err error)
	OnChange(do func(string))
	Init(string) error
}

type ConfigUpdater interface {
	ApplyConfig(string)
	Start() error
	Run()
}
