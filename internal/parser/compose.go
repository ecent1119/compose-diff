package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackgen-cli/compose-diff/internal/models"
	"gopkg.in/yaml.v3"
)

// RawComposeFile represents the raw YAML structure
type RawComposeFile struct {
	Version  string                        `yaml:"version,omitempty"`
	Services map[string]RawService         `yaml:"services"`
	Volumes  map[string]yaml.Node          `yaml:"volumes,omitempty"`
	Networks map[string]yaml.Node          `yaml:"networks,omitempty"`
}

// RawService represents a raw service from YAML
type RawService struct {
	Image       string                 `yaml:"image,omitempty"`
	Build       yaml.Node              `yaml:"build,omitempty"`
	Environment yaml.Node              `yaml:"environment,omitempty"`
	EnvFile     yaml.Node              `yaml:"env_file,omitempty"`
	Ports       yaml.Node              `yaml:"ports,omitempty"`
	Volumes     yaml.Node              `yaml:"volumes,omitempty"`
	Networks    yaml.Node              `yaml:"networks,omitempty"`
	DependsOn   yaml.Node              `yaml:"depends_on,omitempty"`
	Healthcheck yaml.Node              `yaml:"healthcheck,omitempty"`
	Command     yaml.Node              `yaml:"command,omitempty"`
	Entrypoint  yaml.Node              `yaml:"entrypoint,omitempty"`
	Profiles    []string               `yaml:"profiles,omitempty"`
	Restart     string                 `yaml:"restart,omitempty"`
	Labels      yaml.Node              `yaml:"labels,omitempty"`
}

// ParseComposeFile parses a Docker Compose file into the intermediate representation
func ParseComposeFile(filePath string) (*models.ComposeIR, error) {
	// Handle auto-detection of compose file
	actualPath, err := resolveComposePath(filePath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(actualPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var raw RawComposeFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return convertToIR(&raw)
}

// resolveComposePath finds the actual compose file, supporting auto-detection
func resolveComposePath(path string) (string, error) {
	// If it's a directory, look for compose files
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		candidates := []string{
			"compose.yaml",
			"compose.yml",
			"docker-compose.yaml",
			"docker-compose.yml",
		}
		for _, candidate := range candidates {
			fullPath := filepath.Join(path, candidate)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath, nil
			}
		}
		return "", fmt.Errorf("no compose file found in directory: %s", path)
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("file not found: %s", path)
	}

	return path, nil
}

// convertToIR converts a raw compose file to the intermediate representation
func convertToIR(raw *RawComposeFile) (*models.ComposeIR, error) {
	ir := models.NewComposeIR()

	// Convert services
	for name, rawSvc := range raw.Services {
		rawSvcCopy := rawSvc // Create local copy to avoid loop variable capture
		svc, err := convertService(&rawSvcCopy)
		if err != nil {
			return nil, fmt.Errorf("error converting service %s: %w", name, err)
		}
		ir.Services[name] = *svc
	}

	// Convert volumes
	for name, node := range raw.Volumes {
		nodeCopy := node // Create local copy
		vol, err := convertVolume(&nodeCopy)
		if err != nil {
			return nil, fmt.Errorf("error converting volume %s: %w", name, err)
		}
		ir.Volumes[name] = *vol
	}

	// Convert networks
	for name, node := range raw.Networks {
		nodeCopy := node // Create local copy
		net, err := convertNetwork(&nodeCopy)
		if err != nil {
			return nil, fmt.Errorf("error converting network %s: %w", name, err)
		}
		ir.Networks[name] = *net
	}

	return ir, nil
}

// convertService converts a raw service to ServiceIR
func convertService(raw *RawService) (*models.ServiceIR, error) {
	svc := &models.ServiceIR{}

	// Image
	if raw.Image != "" {
		img := raw.Image // Create local copy
		svc.Image = &img
	}

	// Build
	if raw.Build.Kind != 0 {
		build, err := parseBuild(&raw.Build)
		if err != nil {
			return nil, err
		}
		svc.Build = build
	}

	// Environment
	if raw.Environment.Kind != 0 {
		env, err := parseEnvironment(&raw.Environment)
		if err != nil {
			return nil, err
		}
		svc.Env = env
	}

	// Env files
	if raw.EnvFile.Kind != 0 {
		files, err := parseStringOrList(&raw.EnvFile)
		if err != nil {
			return nil, err
		}
		svc.EnvFiles = files
	}

	// Ports
	if raw.Ports.Kind != 0 {
		ports, err := parsePorts(&raw.Ports)
		if err != nil {
			return nil, err
		}
		svc.Ports = ports
	}

	// Volumes
	if raw.Volumes.Kind != 0 {
		volumes, err := parseVolumes(&raw.Volumes)
		if err != nil {
			return nil, err
		}
		svc.Volumes = volumes
	}

	// Networks
	if raw.Networks.Kind != 0 {
		networks, err := parseNetworksRef(&raw.Networks)
		if err != nil {
			return nil, err
		}
		svc.Networks = networks
	}

	// DependsOn
	if raw.DependsOn.Kind != 0 {
		deps, err := parseDependsOn(&raw.DependsOn)
		if err != nil {
			return nil, err
		}
		svc.DependsOn = deps
	}

	// Healthcheck
	if raw.Healthcheck.Kind != 0 {
		hc, err := parseHealthcheck(&raw.Healthcheck)
		if err != nil {
			return nil, err
		}
		svc.Healthcheck = hc
	}

	// Command
	if raw.Command.Kind != 0 {
		cmd, err := parseStringOrList(&raw.Command)
		if err != nil {
			return nil, err
		}
		svc.Command = cmd
	}

	// Entrypoint
	if raw.Entrypoint.Kind != 0 {
		ep, err := parseStringOrList(&raw.Entrypoint)
		if err != nil {
			return nil, err
		}
		svc.Entrypoint = ep
	}

	// Profiles
	svc.Profiles = raw.Profiles

	// Restart
	if raw.Restart != "" {
		restart := raw.Restart // Create local copy
		svc.Restart = &restart
	}

	// Labels
	if raw.Labels.Kind != 0 {
		labels, err := parseLabels(&raw.Labels)
		if err != nil {
			return nil, err
		}
		svc.Labels = labels
	}

	return svc, nil
}

// parseBuild parses the build configuration
func parseBuild(node *yaml.Node) (*models.BuildIR, error) {
	build := &models.BuildIR{}

	// Simple string form: build: ./dir
	if node.Kind == yaml.ScalarNode {
		build.Context = node.Value
		return build, nil
	}

	// Map form
	if node.Kind == yaml.MappingNode {
		var raw struct {
			Context    string            `yaml:"context"`
			Dockerfile string            `yaml:"dockerfile"`
			Args       map[string]string `yaml:"args"`
			Target     string            `yaml:"target"`
		}
		if err := node.Decode(&raw); err != nil {
			return nil, err
		}
		build.Context = raw.Context
		build.Dockerfile = raw.Dockerfile
		build.Args = raw.Args
		build.Target = raw.Target
	}

	return build, nil
}

// parseEnvironment parses environment variables (list or map form)
func parseEnvironment(node *yaml.Node) (map[string]*string, error) {
	env := make(map[string]*string)

	// List form: - KEY=VALUE or - KEY
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			if item.Kind != yaml.ScalarNode {
				continue
			}
			key, value := parseEnvVar(item.Value)
			if value != "" {
				env[key] = &value
			} else {
				env[key] = nil
			}
		}
		return env, nil
	}

	// Map form: KEY: VALUE
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i].Value
			valueNode := node.Content[i+1]
			if valueNode.Kind == yaml.ScalarNode {
				if valueNode.Value == "" {
					env[key] = nil
				} else {
					val := valueNode.Value
					env[key] = &val
				}
			}
		}
		return env, nil
	}

	return env, nil
}

// parseEnvVar splits KEY=VALUE or returns KEY with empty value
func parseEnvVar(s string) (string, string) {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return parts[0], ""
}

// parsePorts parses port mappings
func parsePorts(node *yaml.Node) ([]models.PortIR, error) {
	var ports []models.PortIR

	if node.Kind != yaml.SequenceNode {
		return ports, nil
	}

	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode {
			// String form: "8080:80" or "8080:80/udp"
			port, err := parsePortString(item.Value)
			if err != nil {
				return nil, err
			}
			ports = append(ports, *port)
		} else if item.Kind == yaml.MappingNode {
			// Long form
			port, err := parsePortMapping(item)
			if err != nil {
				return nil, err
			}
			ports = append(ports, *port)
		}
	}

	return ports, nil
}

// parsePortString parses a port string like "8080:80" or "127.0.0.1:8080:80/udp"
func parsePortString(s string) (*models.PortIR, error) {
	port := &models.PortIR{Protocol: "tcp"}

	// Check for protocol suffix
	if strings.HasSuffix(s, "/udp") {
		port.Protocol = "udp"
		s = strings.TrimSuffix(s, "/udp")
	} else if strings.HasSuffix(s, "/tcp") {
		s = strings.TrimSuffix(s, "/tcp")
	}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		// Just container port: "80"
		port.ContainerPort = parts[0]
	case 2:
		// host:container or container:container
		port.HostPort = parts[0]
		port.ContainerPort = parts[1]
	case 3:
		// ip:host:container
		port.HostIP = parts[0]
		port.HostPort = parts[1]
		port.ContainerPort = parts[2]
	}

	return port, nil
}

// parsePortMapping parses the long-form port mapping
func parsePortMapping(node *yaml.Node) (*models.PortIR, error) {
	var raw struct {
		Target    string `yaml:"target"`
		Published string `yaml:"published"`
		HostIP    string `yaml:"host_ip"`
		Protocol  string `yaml:"protocol"`
	}
	if err := node.Decode(&raw); err != nil {
		return nil, err
	}

	protocol := raw.Protocol
	if protocol == "" {
		protocol = "tcp"
	}

	return &models.PortIR{
		HostIP:        raw.HostIP,
		HostPort:      raw.Published,
		ContainerPort: raw.Target,
		Protocol:      protocol,
	}, nil
}

// parseVolumes parses volume mounts
func parseVolumes(node *yaml.Node) ([]models.MountIR, error) {
	var mounts []models.MountIR

	if node.Kind != yaml.SequenceNode {
		return mounts, nil
	}

	for _, item := range node.Content {
		if item.Kind == yaml.ScalarNode {
			// String form: "./data:/app/data:ro"
			mount, err := parseVolumeString(item.Value)
			if err != nil {
				return nil, err
			}
			mounts = append(mounts, *mount)
		} else if item.Kind == yaml.MappingNode {
			// Long form
			mount, err := parseVolumeMapping(item)
			if err != nil {
				return nil, err
			}
			mounts = append(mounts, *mount)
		}
	}

	return mounts, nil
}

// parseVolumeString parses a volume string like "./data:/app/data:ro"
func parseVolumeString(s string) (*models.MountIR, error) {
	mount := &models.MountIR{Type: "bind"}

	parts := strings.Split(s, ":")
	switch len(parts) {
	case 1:
		// Just target (anonymous volume)
		mount.Target = parts[0]
		mount.Type = "volume"
	case 2:
		// source:target
		mount.Source = parts[0]
		mount.Target = parts[1]
		mount.Type = inferMountType(parts[0])
	case 3:
		// source:target:mode
		mount.Source = parts[0]
		mount.Target = parts[1]
		mount.Type = inferMountType(parts[0])
		if parts[2] == "ro" {
			mount.ReadOnly = true
		}
	}

	return mount, nil
}

// inferMountType determines if a source is a bind mount or named volume
func inferMountType(source string) string {
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "/") || strings.HasPrefix(source, "~") {
		return "bind"
	}
	return "volume"
}

// parseVolumeMapping parses long-form volume config
func parseVolumeMapping(node *yaml.Node) (*models.MountIR, error) {
	var raw struct {
		Type     string `yaml:"type"`
		Source   string `yaml:"source"`
		Target   string `yaml:"target"`
		ReadOnly bool   `yaml:"read_only"`
	}
	if err := node.Decode(&raw); err != nil {
		return nil, err
	}

	mountType := raw.Type
	if mountType == "" {
		mountType = inferMountType(raw.Source)
	}

	return &models.MountIR{
		Type:     mountType,
		Source:   raw.Source,
		Target:   raw.Target,
		ReadOnly: raw.ReadOnly,
	}, nil
}

// parseNetworksRef parses network references in a service
func parseNetworksRef(node *yaml.Node) ([]string, error) {
	var networks []string

	// List form: - network1
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			if item.Kind == yaml.ScalarNode {
				networks = append(networks, item.Value)
			}
		}
		return networks, nil
	}

	// Map form: network1: { aliases: [...] }
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			networks = append(networks, node.Content[i].Value)
		}
		return networks, nil
	}

	return networks, nil
}

// parseDependsOn parses depends_on (list or map form)
func parseDependsOn(node *yaml.Node) ([]string, error) {
	var deps []string

	// List form
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			if item.Kind == yaml.ScalarNode {
				deps = append(deps, item.Value)
			}
		}
		return deps, nil
	}

	// Map form (compose v2): db: { condition: service_healthy }
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			deps = append(deps, node.Content[i].Value)
		}
		return deps, nil
	}

	return deps, nil
}

// parseHealthcheck parses healthcheck configuration
func parseHealthcheck(node *yaml.Node) (*models.HealthcheckIR, error) {
	var raw struct {
		Test        yaml.Node `yaml:"test"`
		Interval    string    `yaml:"interval"`
		Timeout     string    `yaml:"timeout"`
		Retries     int       `yaml:"retries"`
		StartPeriod string    `yaml:"start_period"`
		Disable     bool      `yaml:"disable"`
	}
	if err := node.Decode(&raw); err != nil {
		return nil, err
	}

	hc := &models.HealthcheckIR{
		Interval:    raw.Interval,
		Timeout:     raw.Timeout,
		Retries:     raw.Retries,
		StartPeriod: raw.StartPeriod,
		Disable:     raw.Disable,
	}

	// Parse test command
	if raw.Test.Kind == yaml.ScalarNode {
		hc.Test = []string{raw.Test.Value}
	} else if raw.Test.Kind == yaml.SequenceNode {
		for _, item := range raw.Test.Content {
			if item.Kind == yaml.ScalarNode {
				hc.Test = append(hc.Test, item.Value)
			}
		}
	}

	return hc, nil
}

// parseStringOrList parses a value that can be string or list
func parseStringOrList(node *yaml.Node) ([]string, error) {
	var result []string

	if node.Kind == yaml.ScalarNode {
		// Single string - split by spaces for commands
		result = append(result, node.Value)
		return result, nil
	}

	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			if item.Kind == yaml.ScalarNode {
				result = append(result, item.Value)
			}
		}
	}

	return result, nil
}

// parseLabels parses labels (list or map form)
func parseLabels(node *yaml.Node) (map[string]string, error) {
	labels := make(map[string]string)

	// List form: - key=value
	if node.Kind == yaml.SequenceNode {
		for _, item := range node.Content {
			if item.Kind == yaml.ScalarNode {
				key, value := parseEnvVar(item.Value)
				labels[key] = value
			}
		}
		return labels, nil
	}

	// Map form
	if node.Kind == yaml.MappingNode {
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i].Value
			value := node.Content[i+1].Value
			labels[key] = value
		}
		return labels, nil
	}

	return labels, nil
}

// convertVolume converts raw volume definition to VolumeIR
func convertVolume(node *yaml.Node) (*models.VolumeIR, error) {
	vol := &models.VolumeIR{}

	// Empty volume (just declared)
	if node.Kind == 0 || (node.Kind == yaml.ScalarNode && node.Value == "") {
		return vol, nil
	}

	// Full definition
	if node.Kind == yaml.MappingNode {
		var raw struct {
			Driver     string            `yaml:"driver"`
			DriverOpts map[string]string `yaml:"driver_opts"`
			External   bool              `yaml:"external"`
			Name       string            `yaml:"name"`
			Labels     map[string]string `yaml:"labels"`
		}
		if err := node.Decode(&raw); err != nil {
			return nil, err
		}
		vol.Driver = raw.Driver
		vol.DriverOpts = raw.DriverOpts
		vol.External = raw.External
		vol.Name = raw.Name
		vol.Labels = raw.Labels
	}

	return vol, nil
}

// convertNetwork converts raw network definition to NetworkIR
func convertNetwork(node *yaml.Node) (*models.NetworkIR, error) {
	net := &models.NetworkIR{}

	// Empty network (just declared)
	if node.Kind == 0 || (node.Kind == yaml.ScalarNode && node.Value == "") {
		return net, nil
	}

	// Full definition
	if node.Kind == yaml.MappingNode {
		var raw struct {
			Driver     string            `yaml:"driver"`
			DriverOpts map[string]string `yaml:"driver_opts"`
			External   bool              `yaml:"external"`
			Name       string            `yaml:"name"`
			Labels     map[string]string `yaml:"labels"`
		}
		if err := node.Decode(&raw); err != nil {
			return nil, err
		}
		net.Driver = raw.Driver
		net.DriverOpts = raw.DriverOpts
		net.External = raw.External
		net.Name = raw.Name
		net.Labels = raw.Labels
	}

	return net, nil
}

// ParseFromMap converts a raw map (from baseline or resolved config) to ComposeIR
func ParseFromMap(data map[string]any) (*models.ComposeIR, error) {
	// Marshal back to YAML and re-parse through normal path
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var raw RawComposeFile
	if err := yaml.Unmarshal(yamlBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return convertToIR(&raw)
}
