package diff

import (
	"testing"

	"github.com/stackgen-cli/compose-diff/internal/models"
)

func TestCompareServices(t *testing.T) {
	img1 := "node:18"
	img2 := "node:20"
	env1 := "development"
	env2 := "production"

	old := &models.ComposeIR{
		Services: map[string]models.ServiceIR{
			"api": {
				Image: &img1,
				Env: map[string]*string{
					"NODE_ENV":     &env1,
					"DATABASE_URL": ptrStr("postgres://old"),
				},
			},
			"worker": {
				Image: &img1,
			},
		},
		Volumes:  make(map[string]models.VolumeIR),
		Networks: make(map[string]models.NetworkIR),
	}

	new := &models.ComposeIR{
		Services: map[string]models.ServiceIR{
			"api": {
				Image: &img2,
				Env: map[string]*string{
					"NODE_ENV": &env2,
					"API_KEY":  ptrStr("secret"),
				},
			},
			"cache": {
				Image: ptrStr("redis:7"),
			},
		},
		Volumes:  make(map[string]models.VolumeIR),
		Networks: make(map[string]models.NetworkIR),
	}

	report := Compare(old, new)

	// Check summary
	if report.Summary.ServicesAdded != 1 {
		t.Errorf("Expected 1 service added, got %d", report.Summary.ServicesAdded)
	}
	if report.Summary.ServicesRemoved != 1 {
		t.Errorf("Expected 1 service removed, got %d", report.Summary.ServicesRemoved)
	}
	if report.Summary.ServicesChanged != 1 {
		t.Errorf("Expected 1 service changed, got %d", report.Summary.ServicesChanged)
	}

	// Check for specific changes
	hasImageChange := false
	hasEnvRemoved := false
	hasEnvAdded := false
	hasServiceRemoved := false

	for _, c := range report.Changes {
		if c.Path == "services.api.image" {
			hasImageChange = true
		}
		if c.Path == "services.api.environment.DATABASE_URL" && c.Kind == models.ChangeRemoved {
			hasEnvRemoved = true
			if c.Severity != models.SeverityBreaking {
				t.Error("Removed env var should be breaking")
			}
		}
		if c.Path == "services.api.environment.API_KEY" && c.Kind == models.ChangeAdded {
			hasEnvAdded = true
		}
		if c.Name == "worker" && c.Kind == models.ChangeRemoved {
			hasServiceRemoved = true
			if c.Severity != models.SeverityBreaking {
				t.Error("Removed service should be breaking")
			}
		}
	}

	if !hasImageChange {
		t.Error("Expected image change not found")
	}
	if !hasEnvRemoved {
		t.Error("Expected DATABASE_URL removal not found")
	}
	if !hasEnvAdded {
		t.Error("Expected API_KEY addition not found")
	}
	if !hasServiceRemoved {
		t.Error("Expected worker removal not found")
	}
}

func TestCompareVolumes(t *testing.T) {
	old := &models.ComposeIR{
		Services: make(map[string]models.ServiceIR),
		Volumes: map[string]models.VolumeIR{
			"pgdata":    {},
			"redisdata": {},
		},
		Networks: make(map[string]models.NetworkIR),
	}

	new := &models.ComposeIR{
		Services: make(map[string]models.ServiceIR),
		Volumes: map[string]models.VolumeIR{
			"pgdata":  {},
			"esdata":  {},
		},
		Networks: make(map[string]models.NetworkIR),
	}

	report := Compare(old, new)

	if report.Summary.VolumesAdded != 1 {
		t.Errorf("Expected 1 volume added, got %d", report.Summary.VolumesAdded)
	}
	if report.Summary.VolumesRemoved != 1 {
		t.Errorf("Expected 1 volume removed, got %d", report.Summary.VolumesRemoved)
	}

	// Volume removal should be breaking
	for _, c := range report.Changes {
		if c.Scope == models.ScopeVolume && c.Kind == models.ChangeRemoved {
			if c.Severity != models.SeverityBreaking {
				t.Error("Removed volume should be breaking")
			}
		}
	}
}

func TestComparePorts(t *testing.T) {
	img := "nginx:latest"

	old := &models.ComposeIR{
		Services: map[string]models.ServiceIR{
			"web": {
				Image: &img,
				Ports: []models.PortIR{
					{HostPort: "80", ContainerPort: "80", Protocol: "tcp"},
					{HostPort: "443", ContainerPort: "443", Protocol: "tcp"},
				},
			},
		},
		Volumes:  make(map[string]models.VolumeIR),
		Networks: make(map[string]models.NetworkIR),
	}

	new := &models.ComposeIR{
		Services: map[string]models.ServiceIR{
			"web": {
				Image: &img,
				Ports: []models.PortIR{
					{HostPort: "8080", ContainerPort: "80", Protocol: "tcp"},
				},
			},
		},
		Volumes:  make(map[string]models.VolumeIR),
		Networks: make(map[string]models.NetworkIR),
	}

	report := Compare(old, new)

	hasPortRemoved := false
	hasPortAdded := false

	for _, c := range report.Changes {
		if c.Kind == models.ChangeRemoved && c.Name == "web" {
			hasPortRemoved = true
		}
		if c.Kind == models.ChangeAdded && c.Name == "web" {
			hasPortAdded = true
		}
	}

	if !hasPortRemoved {
		t.Error("Expected port removal (old 80:80 and 443:443)")
	}
	if !hasPortAdded {
		t.Error("Expected port addition (new 8080:80)")
	}
}

func TestFilterByService(t *testing.T) {
	report := &models.DiffReport{
		Changes: []models.Change{
			{Scope: models.ScopeService, Name: "api", Path: "services.api.image"},
			{Scope: models.ScopeService, Name: "db", Path: "services.db.image"},
			{Scope: models.ScopeVolume, Name: "data", Path: "volumes.data"},
		},
	}

	filtered := FilterByService(report, "api")

	if len(filtered.Changes) != 1 {
		t.Errorf("Expected 1 change after filter, got %d", len(filtered.Changes))
	}
	if filtered.Changes[0].Name != "api" {
		t.Error("Filtered change should be for api service")
	}
}

func TestFilterBySeverity(t *testing.T) {
	report := &models.DiffReport{
		Changes: []models.Change{
			{Severity: models.SeverityInfo},
			{Severity: models.SeverityWarning},
			{Severity: models.SeverityBreaking},
		},
	}

	filtered := FilterBySeverity(report, "warning")

	if len(filtered.Changes) != 2 {
		t.Errorf("Expected 2 changes at warning+, got %d", len(filtered.Changes))
	}
}

func ptrStr(s string) *string {
	return &s
}
