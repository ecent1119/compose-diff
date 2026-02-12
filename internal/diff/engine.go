package diff

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/stackgen-cli/compose-diff/internal/models"
)

// Compare compares two ComposeIR and produces a DiffReport
func Compare(old, new *models.ComposeIR) *models.DiffReport {
	report := models.NewDiffReport()

	// Compare services
	compareServices(old.Services, new.Services, report)

	// Compare volumes
	compareVolumes(old.Volumes, new.Volumes, report)

	// Compare networks
	compareNetworks(old.Networks, new.Networks, report)

	return report
}

// compareServices compares service maps
func compareServices(old, new map[string]models.ServiceIR, report *models.DiffReport) {
	// Find added and removed services
	oldNames := mapKeys(old)
	newNames := mapKeys(new)

	added, removed, common := diffSets(oldNames, newNames)

	// Track counts
	report.Summary.ServicesAdded = len(added)
	report.Summary.ServicesRemoved = len(removed)

	// Report added services
	for _, name := range added {
		report.AddChange(models.Change{
			Kind:     models.ChangeAdded,
			Scope:    models.ScopeService,
			Name:     name,
			Path:     fmt.Sprintf("services.%s", name),
			Before:   nil,
			After:    new[name],
			Severity: models.SeverityInfo,
		})
	}

	// Report removed services (breaking!)
	for _, name := range removed {
		report.AddChange(models.Change{
			Kind:     models.ChangeRemoved,
			Scope:    models.ScopeService,
			Name:     name,
			Path:     fmt.Sprintf("services.%s", name),
			Before:   old[name],
			After:    nil,
			Severity: models.SeverityBreaking,
		})
	}

	// Compare common services
	changedServices := 0
	for _, name := range common {
		oldSvc := old[name]
		newSvc := new[name]
		changes := compareService(name, &oldSvc, &newSvc)
		if len(changes) > 0 {
			changedServices++
			for _, c := range changes {
				report.AddChange(c)
			}
		}
	}
	report.Summary.ServicesChanged = changedServices
}

// compareService compares two services and returns changes
func compareService(name string, old, new *models.ServiceIR) []models.Change {
	var changes []models.Change

	basePath := fmt.Sprintf("services.%s", name)

	// Image
	if !ptrEqual(old.Image, new.Image) {
		changes = append(changes, models.Change{
			Kind:     models.ChangeModified,
			Scope:    models.ScopeService,
			Name:     name,
			Path:     basePath + ".image",
			Before:   ptrValue(old.Image),
			After:    ptrValue(new.Image),
			Severity: imageSeverity(old.Image, new.Image),
		})
	}

	// Environment variables
	envChanges := compareEnv(name, basePath, old.Env, new.Env)
	changes = append(changes, envChanges...)

	// Ports
	portChanges := comparePorts(name, basePath, old.Ports, new.Ports)
	changes = append(changes, portChanges...)

	// Volumes
	volChanges := compareServiceVolumes(name, basePath, old.Volumes, new.Volumes)
	changes = append(changes, volChanges...)

	// Networks
	netChanges := compareStringSlice(name, basePath+".networks", old.Networks, new.Networks, models.SeverityInfo)
	changes = append(changes, netChanges...)

	// DependsOn
	depChanges := compareStringSlice(name, basePath+".depends_on", old.DependsOn, new.DependsOn, models.SeverityWarning)
	changes = append(changes, depChanges...)

	// Healthcheck
	if !healthcheckEqual(old.Healthcheck, new.Healthcheck) {
		sev := models.SeverityInfo
		if old.Healthcheck != nil && new.Healthcheck == nil {
			sev = models.SeverityBreaking // Healthcheck removed
		}
		changes = append(changes, models.Change{
			Kind:     changeKindForPtrs(old.Healthcheck, new.Healthcheck),
			Scope:    models.ScopeService,
			Name:     name,
			Path:     basePath + ".healthcheck",
			Before:   old.Healthcheck,
			After:    new.Healthcheck,
			Severity: sev,
		})
	}

	// Command
	if !sliceEqual(old.Command, new.Command) {
		changes = append(changes, models.Change{
			Kind:     models.ChangeModified,
			Scope:    models.ScopeService,
			Name:     name,
			Path:     basePath + ".command",
			Before:   old.Command,
			After:    new.Command,
			Severity: models.SeverityInfo,
		})
	}

	// Entrypoint
	if !sliceEqual(old.Entrypoint, new.Entrypoint) {
		changes = append(changes, models.Change{
			Kind:     models.ChangeModified,
			Scope:    models.ScopeService,
			Name:     name,
			Path:     basePath + ".entrypoint",
			Before:   old.Entrypoint,
			After:    new.Entrypoint,
			Severity: models.SeverityWarning,
		})
	}

	// Restart
	if !ptrEqual(old.Restart, new.Restart) {
		changes = append(changes, models.Change{
			Kind:     models.ChangeModified,
			Scope:    models.ScopeService,
			Name:     name,
			Path:     basePath + ".restart",
			Before:   ptrValue(old.Restart),
			After:    ptrValue(new.Restart),
			Severity: models.SeverityInfo,
		})
	}

	return changes
}

// compareEnv compares environment variable maps
func compareEnv(svcName, basePath string, old, new map[string]*string) []models.Change {
	var changes []models.Change

	oldKeys := envKeys(old)
	newKeys := envKeys(new)

	added, removed, common := diffSets(oldKeys, newKeys)

	// Added env vars
	for _, key := range added {
		changes = append(changes, models.Change{
			Kind:     models.ChangeAdded,
			Scope:    models.ScopeService,
			Name:     svcName,
			Path:     fmt.Sprintf("%s.environment.%s", basePath, key),
			Before:   nil,
			After:    ptrValue(new[key]),
			Severity: models.SeverityInfo,
		})
	}

	// Removed env vars (breaking!)
	for _, key := range removed {
		changes = append(changes, models.Change{
			Kind:     models.ChangeRemoved,
			Scope:    models.ScopeService,
			Name:     svcName,
			Path:     fmt.Sprintf("%s.environment.%s", basePath, key),
			Before:   ptrValue(old[key]),
			After:    nil,
			Severity: models.SeverityBreaking,
		})
	}

	// Modified env vars
	for _, key := range common {
		oldVal := ptrValue(old[key])
		newVal := ptrValue(new[key])
		if oldVal != newVal {
			changes = append(changes, models.Change{
				Kind:     models.ChangeModified,
				Scope:    models.ScopeService,
				Name:     svcName,
				Path:     fmt.Sprintf("%s.environment.%s", basePath, key),
				Before:   oldVal,
				After:    newVal,
				Severity: models.SeverityWarning,
			})
		}
	}

	return changes
}

// comparePorts compares port mappings
func comparePorts(svcName, basePath string, old, new []models.PortIR) []models.Change {
	var changes []models.Change

	oldMap := portMap(old)
	newMap := portMap(new)

	oldKeys := mapKeys(oldMap)
	newKeys := mapKeys(newMap)

	added, removed, common := diffSets(oldKeys, newKeys)

	// Added ports
	for _, key := range added {
		changes = append(changes, models.Change{
			Kind:     models.ChangeAdded,
			Scope:    models.ScopeService,
			Name:     svcName,
			Path:     fmt.Sprintf("%s.ports.%s", basePath, key),
			Before:   nil,
			After:    newMap[key],
			Severity: models.SeverityInfo,
		})
	}

	// Removed ports (breaking!)
	for _, key := range removed {
		changes = append(changes, models.Change{
			Kind:     models.ChangeRemoved,
			Scope:    models.ScopeService,
			Name:     svcName,
			Path:     fmt.Sprintf("%s.ports.%s", basePath, key),
			Before:   oldMap[key],
			After:    nil,
			Severity: models.SeverityBreaking,
		})
	}

	// Modified ports
	for _, key := range common {
		oldPort := oldMap[key]
		newPort := newMap[key]
		if !reflect.DeepEqual(oldPort, newPort) {
			changes = append(changes, models.Change{
				Kind:     models.ChangeModified,
				Scope:    models.ScopeService,
				Name:     svcName,
				Path:     fmt.Sprintf("%s.ports.%s", basePath, key),
				Before:   oldPort,
				After:    newPort,
				Severity: models.SeverityWarning,
			})
		}
	}

	return changes
}

// compareServiceVolumes compares volume mounts
func compareServiceVolumes(svcName, basePath string, old, new []models.MountIR) []models.Change {
	var changes []models.Change

	oldMap := mountMap(old)
	newMap := mountMap(new)

	oldKeys := mapKeys(oldMap)
	newKeys := mapKeys(newMap)

	added, removed, common := diffSets(oldKeys, newKeys)

	// Added volumes
	for _, key := range added {
		changes = append(changes, models.Change{
			Kind:     models.ChangeAdded,
			Scope:    models.ScopeService,
			Name:     svcName,
			Path:     fmt.Sprintf("%s.volumes.%s", basePath, key),
			Before:   nil,
			After:    newMap[key],
			Severity: models.SeverityInfo,
		})
	}

	// Removed volumes (breaking!)
	for _, key := range removed {
		changes = append(changes, models.Change{
			Kind:     models.ChangeRemoved,
			Scope:    models.ScopeService,
			Name:     svcName,
			Path:     fmt.Sprintf("%s.volumes.%s", basePath, key),
			Before:   oldMap[key],
			After:    nil,
			Severity: models.SeverityBreaking,
		})
	}

	// Modified volumes
	for _, key := range common {
		oldMount := oldMap[key]
		newMount := newMap[key]
		if !reflect.DeepEqual(oldMount, newMount) {
			changes = append(changes, models.Change{
				Kind:     models.ChangeModified,
				Scope:    models.ScopeService,
				Name:     svcName,
				Path:     fmt.Sprintf("%s.volumes.%s", basePath, key),
				Before:   oldMount,
				After:    newMount,
				Severity: models.SeverityWarning,
			})
		}
	}

	return changes
}

// compareStringSlice compares string slices and reports changes
func compareStringSlice(svcName, path string, old, new []string, severity models.Severity) []models.Change {
	var changes []models.Change

	added, removed, _ := diffSets(old, new)

	for _, item := range added {
		changes = append(changes, models.Change{
			Kind:     models.ChangeAdded,
			Scope:    models.ScopeService,
			Name:     svcName,
			Path:     fmt.Sprintf("%s.%s", path, item),
			Before:   nil,
			After:    item,
			Severity: models.SeverityInfo,
		})
	}

	for _, item := range removed {
		changes = append(changes, models.Change{
			Kind:     models.ChangeRemoved,
			Scope:    models.ScopeService,
			Name:     svcName,
			Path:     fmt.Sprintf("%s.%s", path, item),
			Before:   item,
			After:    nil,
			Severity: severity,
		})
	}

	return changes
}

// compareVolumes compares top-level volume definitions
func compareVolumes(old, new map[string]models.VolumeIR, report *models.DiffReport) {
	oldNames := mapKeys(old)
	newNames := mapKeys(new)

	added, removed, _ := diffSets(oldNames, newNames)

	report.Summary.VolumesAdded = len(added)
	report.Summary.VolumesRemoved = len(removed)

	for _, name := range added {
		report.AddChange(models.Change{
			Kind:     models.ChangeAdded,
			Scope:    models.ScopeVolume,
			Name:     name,
			Path:     fmt.Sprintf("volumes.%s", name),
			Before:   nil,
			After:    new[name],
			Severity: models.SeverityInfo,
		})
	}

	for _, name := range removed {
		report.AddChange(models.Change{
			Kind:     models.ChangeRemoved,
			Scope:    models.ScopeVolume,
			Name:     name,
			Path:     fmt.Sprintf("volumes.%s", name),
			Before:   old[name],
			After:    nil,
			Severity: models.SeverityBreaking,
		})
	}
}

// compareNetworks compares top-level network definitions
func compareNetworks(old, new map[string]models.NetworkIR, report *models.DiffReport) {
	oldNames := mapKeys(old)
	newNames := mapKeys(new)

	added, removed, _ := diffSets(oldNames, newNames)

	report.Summary.NetworksAdded = len(added)
	report.Summary.NetworksRemoved = len(removed)

	for _, name := range added {
		report.AddChange(models.Change{
			Kind:     models.ChangeAdded,
			Scope:    models.ScopeNetwork,
			Name:     name,
			Path:     fmt.Sprintf("networks.%s", name),
			Before:   nil,
			After:    new[name],
			Severity: models.SeverityInfo,
		})
	}

	for _, name := range removed {
		report.AddChange(models.Change{
			Kind:     models.ChangeRemoved,
			Scope:    models.ScopeNetwork,
			Name:     name,
			Path:     fmt.Sprintf("networks.%s", name),
			Before:   old[name],
			After:    nil,
			Severity: models.SeverityWarning,
		})
	}
}

// FilterByService filters a report to only include changes for a specific service
func FilterByService(report *models.DiffReport, service string) *models.DiffReport {
	filtered := models.NewDiffReport()
	filtered.Summary = report.Summary

	for _, c := range report.Changes {
		if c.Scope == models.ScopeService && c.Name == service {
			filtered.Changes = append(filtered.Changes, c)
		}
	}

	return filtered
}

// FilterBySeverity filters a report to only include changes at or above a severity level
func FilterBySeverity(report *models.DiffReport, minSeverity string) *models.DiffReport {
	minLevel := models.SeverityLevel(models.ParseSeverity(minSeverity))

	filtered := models.NewDiffReport()
	filtered.Summary = report.Summary

	for _, c := range report.Changes {
		if models.SeverityLevel(c.Severity) >= minLevel {
			filtered.Changes = append(filtered.Changes, c)
		}
	}

	return filtered
}

// Helper functions

func mapKeys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func envKeys(m map[string]*string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func diffSets(old, new []string) (added, removed, common []string) {
	oldSet := make(map[string]bool)
	newSet := make(map[string]bool)

	for _, s := range old {
		oldSet[s] = true
	}
	for _, s := range new {
		newSet[s] = true
	}

	for s := range newSet {
		if !oldSet[s] {
			added = append(added, s)
		}
	}
	for s := range oldSet {
		if !newSet[s] {
			removed = append(removed, s)
		} else {
			common = append(common, s)
		}
	}

	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(common)
	return
}

func ptrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func ptrValue(p *string) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func healthcheckEqual(a, b *models.HealthcheckIR) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func changeKindForPtrs(old, new interface{}) models.ChangeKind {
	if old == nil {
		return models.ChangeAdded
	}
	if new == nil {
		return models.ChangeRemoved
	}
	return models.ChangeModified
}

func portMap(ports []models.PortIR) map[string]models.PortIR {
	m := make(map[string]models.PortIR)
	for _, p := range ports {
		key := fmt.Sprintf("%s:%s/%s", p.HostPort, p.ContainerPort, p.Protocol)
		m[key] = p
	}
	return m
}

func mountMap(mounts []models.MountIR) map[string]models.MountIR {
	m := make(map[string]models.MountIR)
	for _, mount := range mounts {
		m[mount.Target] = mount
	}
	return m
}
