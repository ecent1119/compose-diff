package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseComposeFile(t *testing.T) {
	// Create temp compose file
	content := `services:
  api:
    image: node:18-alpine
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=development
      - DATABASE_URL=postgres://localhost/db
    volumes:
      - ./src:/app/src
    depends_on:
      - db
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: mydb
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - pgdata:/var/lib/postgresql/data

volumes:
  pgdata:

networks:
  default:
`
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ir, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	// Verify services parsed
	if len(ir.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(ir.Services))
	}

	// Debug: print all service names and images
	for name, svc := range ir.Services {
		img := "<nil>"
		if svc.Image != nil {
			img = *svc.Image
		}
		t.Logf("Found service: %s with image: %s", name, img)
	}

	// Verify api service
	api, ok := ir.Services["api"]
	if !ok {
		t.Fatal("api service not found")
	}
	if api.Image == nil {
		t.Error("api image should not be nil")
	} else if *api.Image != "node:18-alpine" {
		t.Errorf("Expected api image node:18-alpine, got %s", *api.Image)
	}
	if len(api.Ports) != 1 {
		t.Errorf("Expected 1 port, got %d", len(api.Ports))
	}
	if len(api.Env) != 2 {
		t.Errorf("Expected 2 env vars, got %d", len(api.Env))
	}
	if len(api.DependsOn) != 1 || api.DependsOn[0] != "db" {
		t.Errorf("Expected depends_on [db], got %v", api.DependsOn)
	}

	// Verify db service
	db, ok := ir.Services["db"]
	if !ok {
		t.Fatal("db service not found")
	}
	if db.Image == nil || *db.Image != "postgres:16-alpine" {
		t.Errorf("Expected db image postgres:16-alpine, got %v", db.Image)
	}

	// Verify volumes
	if len(ir.Volumes) != 1 {
		t.Errorf("Expected 1 volume, got %d", len(ir.Volumes))
	}
	if _, ok := ir.Volumes["pgdata"]; !ok {
		t.Error("pgdata volume not found")
	}
}

func TestParseEnvironmentListForm(t *testing.T) {
	content := `
services:
  app:
    image: app:latest
    environment:
      - KEY1=value1
      - KEY2=value2
      - KEY3
`
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ir, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	app := ir.Services["app"]
	if len(app.Env) != 3 {
		t.Errorf("Expected 3 env vars, got %d", len(app.Env))
	}

	if app.Env["KEY1"] == nil || *app.Env["KEY1"] != "value1" {
		t.Error("KEY1 should be 'value1'")
	}
	if app.Env["KEY3"] != nil {
		t.Error("KEY3 should be nil (undefined value)")
	}
}

func TestParsePortsShortAndLongForm(t *testing.T) {
	content := `
services:
  app:
    image: app:latest
    ports:
      - "8080:80"
      - "127.0.0.1:9000:9000/udp"
      - target: 443
        published: 8443
        protocol: tcp
`
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ir, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	app := ir.Services["app"]
	if len(app.Ports) != 3 {
		t.Fatalf("Expected 3 ports, got %d", len(app.Ports))
	}

	// Check first port
	p1 := app.Ports[0]
	if p1.HostPort != "8080" || p1.ContainerPort != "80" || p1.Protocol != "tcp" {
		t.Errorf("Port 1 mismatch: %+v", p1)
	}

	// Check second port (with IP and UDP)
	p2 := app.Ports[1]
	if p2.HostIP != "127.0.0.1" || p2.Protocol != "udp" {
		t.Errorf("Port 2 mismatch: %+v", p2)
	}

	// Check third port (long form)
	p3 := app.Ports[2]
	if p3.ContainerPort != "443" || p3.HostPort != "8443" {
		t.Errorf("Port 3 mismatch: %+v", p3)
	}
}

func TestParseVolumesShortAndLongForm(t *testing.T) {
	content := `
services:
  app:
    image: app:latest
    volumes:
      - ./src:/app/src
      - data:/app/data:ro
      - type: bind
        source: ./config
        target: /app/config
        read_only: true
`
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ir, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("ParseComposeFile failed: %v", err)
	}

	app := ir.Services["app"]
	if len(app.Volumes) != 3 {
		t.Fatalf("Expected 3 volumes, got %d", len(app.Volumes))
	}

	// Check bind mount
	v1 := app.Volumes[0]
	if v1.Type != "bind" || v1.Source != "./src" || v1.Target != "/app/src" {
		t.Errorf("Volume 1 mismatch: %+v", v1)
	}

	// Check named volume with ro flag
	v2 := app.Volumes[1]
	if v2.Type != "volume" || !v2.ReadOnly {
		t.Errorf("Volume 2 mismatch: %+v", v2)
	}

	// Check long form
	v3 := app.Volumes[2]
	if v3.Type != "bind" || !v3.ReadOnly || v3.Target != "/app/config" {
		t.Errorf("Volume 3 mismatch: %+v", v3)
	}
}
