package validation

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadScenario loads a scenario from a YAML file.
func LoadScenario(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scenario: %w", err)
	}
	var s Scenario
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse scenario: %w", err)
	}
	if s.Name == "" {
		s.Name = filepath.Base(path)
	}
	if s.RobotID == "" {
		s.RobotID = "sim-001"
	}
	return &s, nil
}

// LoadScenariosFromDir loads all scenarios from a directory.
func LoadScenariosFromDir(dir string) ([]*Scenario, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	var scenarios []*Scenario
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := filepath.Ext(e.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		s, err := LoadScenario(path)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", e.Name(), err)
		}
		scenarios = append(scenarios, s)
	}
	return scenarios, nil
}

// ScenarioNames returns the list of scenario names for a given directory.
func ScenarioNames(dir string) ([]string, error) {
	scenarios, err := LoadScenariosFromDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(scenarios))
	for _, s := range scenarios {
		names = append(names, s.Name)
	}
	return names, nil
}
