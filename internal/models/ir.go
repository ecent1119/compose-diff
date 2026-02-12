package models

// ComposeIR is the canonical intermediate representation of a Docker Compose file
type ComposeIR struct {
	Services map[string]ServiceIR `json:"services"`
	Volumes  map[string]VolumeIR  `json:"volumes"`
	Networks map[string]NetworkIR `json:"networks"`
}

// ServiceIR represents a normalized service configuration
type ServiceIR struct {
	Image       *string        `json:"image,omitempty"`
	Build       *BuildIR       `json:"build,omitempty"`
	Env         map[string]*string `json:"environment,omitempty"` // nil value means present but empty
	EnvFiles    []string       `json:"env_file,omitempty"`
	Ports       []PortIR       `json:"ports,omitempty"`
	Volumes     []MountIR      `json:"volumes,omitempty"`
	Networks    []string       `json:"networks,omitempty"`
	DependsOn   []string       `json:"depends_on,omitempty"`
	Healthcheck *HealthcheckIR `json:"healthcheck,omitempty"`
	Command     []string       `json:"command,omitempty"`
	Entrypoint  []string       `json:"entrypoint,omitempty"`
	Profiles    []string       `json:"profiles,omitempty"`
	Restart     *string        `json:"restart,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// BuildIR represents a build configuration
type BuildIR struct {
	Context    string            `json:"context,omitempty"`
	Dockerfile string            `json:"dockerfile,omitempty"`
	Args       map[string]string `json:"args,omitempty"`
	Target     string            `json:"target,omitempty"`
}

// PortIR represents a normalized port mapping
type PortIR struct {
	HostIP        string `json:"host_ip,omitempty"`
	HostPort      string `json:"host_port,omitempty"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol,omitempty"` // tcp or udp
}

// MountIR represents a normalized volume mount
type MountIR struct {
	Type     string `json:"type"` // volume, bind, tmpfs
	Source   string `json:"source,omitempty"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

// HealthcheckIR represents a healthcheck configuration
type HealthcheckIR struct {
	Test        []string `json:"test,omitempty"`
	Interval    string   `json:"interval,omitempty"`
	Timeout     string   `json:"timeout,omitempty"`
	Retries     int      `json:"retries,omitempty"`
	StartPeriod string   `json:"start_period,omitempty"`
	Disable     bool     `json:"disable,omitempty"`
}

// VolumeIR represents a top-level volume definition
type VolumeIR struct {
	Driver     string            `json:"driver,omitempty"`
	DriverOpts map[string]string `json:"driver_opts,omitempty"`
	External   bool              `json:"external,omitempty"`
	Name       string            `json:"name,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// NetworkIR represents a top-level network definition
type NetworkIR struct {
	Driver     string            `json:"driver,omitempty"`
	DriverOpts map[string]string `json:"driver_opts,omitempty"`
	External   bool              `json:"external,omitempty"`
	Name       string            `json:"name,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// NewComposeIR creates an empty ComposeIR with initialized maps
func NewComposeIR() *ComposeIR {
	return &ComposeIR{
		Services: make(map[string]ServiceIR),
		Volumes:  make(map[string]VolumeIR),
		Networks: make(map[string]NetworkIR),
	}
}
