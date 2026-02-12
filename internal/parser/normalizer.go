package parser

import (
	"sort"

	"github.com/stackgen-cli/compose-diff/internal/models"
)

// Normalize normalizes a ComposeIR for consistent comparison
func Normalize(ir *models.ComposeIR) *models.ComposeIR {
	result := models.NewComposeIR()

	// Normalize services
	for name, svc := range ir.Services {
		result.Services[name] = normalizeService(svc)
	}

	// Copy volumes and networks as-is (already normalized during parsing)
	for name, vol := range ir.Volumes {
		result.Volumes[name] = vol
	}
	for name, net := range ir.Networks {
		result.Networks[name] = net
	}

	return result
}

// normalizeService normalizes a service for comparison
func normalizeService(svc models.ServiceIR) models.ServiceIR {
	result := models.ServiceIR{
		Image:       svc.Image,
		Build:       svc.Build,
		Env:         svc.Env,
		EnvFiles:    sortedStrings(svc.EnvFiles),
		Ports:       normalizePorts(svc.Ports),
		Volumes:     normalizeVolumes(svc.Volumes),
		Networks:    sortedStrings(svc.Networks),
		DependsOn:   sortedStrings(svc.DependsOn),
		Healthcheck: svc.Healthcheck,
		Command:     svc.Command,
		Entrypoint:  svc.Entrypoint,
		Profiles:    sortedStrings(svc.Profiles),
		Restart:     svc.Restart,
		Labels:      svc.Labels,
	}

	return result
}

// normalizePorts sorts ports for consistent ordering
func normalizePorts(ports []models.PortIR) []models.PortIR {
	if len(ports) == 0 {
		return ports
	}

	result := make([]models.PortIR, len(ports))
	copy(result, ports)

	sort.Slice(result, func(i, j int) bool {
		// Sort by container port, then host port
		if result[i].ContainerPort != result[j].ContainerPort {
			return result[i].ContainerPort < result[j].ContainerPort
		}
		return result[i].HostPort < result[j].HostPort
	})

	return result
}

// normalizeVolumes sorts volumes for consistent ordering
func normalizeVolumes(volumes []models.MountIR) []models.MountIR {
	if len(volumes) == 0 {
		return volumes
	}

	result := make([]models.MountIR, len(volumes))
	copy(result, volumes)

	sort.Slice(result, func(i, j int) bool {
		return result[i].Target < result[j].Target
	})

	return result
}

// sortedStrings returns a sorted copy of the string slice
func sortedStrings(s []string) []string {
	if len(s) == 0 {
		return s
	}

	result := make([]string, len(s))
	copy(result, s)
	sort.Strings(result)
	return result
}
