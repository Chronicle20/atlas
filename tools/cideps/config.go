package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type ServiceEntry struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Path              string `json:"path"`
	ModulePath        string `json:"module_path,omitempty"`
	DockerImage       string `json:"docker_image,omitempty"`
	DockerContext     string `json:"docker_context,omitempty"`
}

type LibraryEntry struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	ModulePath        string `json:"module_path"`
	CoverageThreshold int    `json:"coverage_threshold,omitempty"`
}

type Config struct {
	Services  []ServiceEntry `json:"services"`
	Libraries []LibraryEntry `json:"libraries"`

	warnings []string
}

func (c *Config) Warnings(dst *[]string) {
	*dst = append(*dst, c.warnings...)
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &cfg, nil
}

// GoServiceRow matches the matrix shape consumed by test-go-services.
type GoServiceRow struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	ModulePath  string `json:"module_path"`
	DockerImage string `json:"docker_image,omitempty"`
}

// GoLibraryRow matches the matrix shape consumed by test-go-libraries.
type GoLibraryRow struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	ModulePath        string `json:"module_path"`
	CoverageThreshold int    `json:"coverage_threshold"`
}

// DockerServiceRow matches the matrix shape consumed by build-docker.
type DockerServiceRow struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	DockerContext string `json:"docker_context"`
	DockerImage   string `json:"docker_image"`
}

func (c *Config) EnrichGoServices(names []string) []GoServiceRow {
	byName := make(map[string]ServiceEntry, len(c.Services))
	for _, s := range c.Services {
		byName[s.Name] = s
	}
	sort.Strings(names)
	out := make([]GoServiceRow, 0, len(names))
	for _, n := range names {
		s, ok := byName[n]
		if !ok {
			c.warnings = append(c.warnings, fmt.Sprintf("services.json has no entry for %q", n))
			continue
		}
		if s.Type != "go-service" {
			continue
		}
		out = append(out, GoServiceRow{
			Name: s.Name, Path: s.Path, ModulePath: s.ModulePath, DockerImage: s.DockerImage,
		})
	}
	return out
}

func (c *Config) EnrichGoLibraries(names []string) []GoLibraryRow {
	byName := make(map[string]LibraryEntry, len(c.Libraries))
	for _, l := range c.Libraries {
		byName[l.Name] = l
	}
	sort.Strings(names)
	out := make([]GoLibraryRow, 0, len(names))
	for _, n := range names {
		l, ok := byName[n]
		if !ok {
			c.warnings = append(c.warnings, fmt.Sprintf("services.json has no entry for lib %q", n))
			continue
		}
		out = append(out, GoLibraryRow{
			Name: l.Name, Path: l.Path, ModulePath: l.ModulePath, CoverageThreshold: l.CoverageThreshold,
		})
	}
	return out
}

func (c *Config) EnrichDockerServices(names []string) []DockerServiceRow {
	byName := make(map[string]ServiceEntry, len(c.Services))
	for _, s := range c.Services {
		byName[s.Name] = s
	}
	sort.Strings(names)
	out := make([]DockerServiceRow, 0, len(names))
	for _, n := range names {
		s, ok := byName[n]
		if !ok {
			c.warnings = append(c.warnings, fmt.Sprintf("services.json has no entry for docker service %q", n))
			continue
		}
		if s.DockerImage == "" {
			continue
		}
		ctx := s.DockerContext
		if ctx == "" {
			ctx = s.Path
		}
		out = append(out, DockerServiceRow{
			Name: s.Name, Path: s.Path, DockerContext: ctx, DockerImage: s.DockerImage,
		})
	}
	return out
}
