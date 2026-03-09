package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadScenario_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	content := `
name: test_scenario
description: "Test scenario"
robot_id: sim-001
steps:
  - action: start_mall_assistant
  - action: submit_visitor_request
    text: "Where is Nike?"
assertions:
  - type: final_robot_state
    state: IDLE
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	s, err := LoadScenario(path)
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}
	if s.Name != "test_scenario" {
		t.Errorf("expected name test_scenario, got %s", s.Name)
	}
	if s.RobotID != "sim-001" {
		t.Errorf("expected robot_id sim-001, got %s", s.RobotID)
	}
	if len(s.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(s.Steps))
	}
	if len(s.Assertions) != 1 {
		t.Errorf("expected 1 assertion, got %d", len(s.Assertions))
	}
}

func TestLoadScenario_DefaultRobotID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	content := `
name: minimal
steps:
  - action: wait
    timeout_sec: 1
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	s, err := LoadScenario(path)
	if err != nil {
		t.Fatalf("LoadScenario: %v", err)
	}
	if s.RobotID != "sim-001" {
		t.Errorf("expected default robot_id sim-001, got %s", s.RobotID)
	}
}

func TestLoadScenario_NotFound(t *testing.T) {
	_, err := LoadScenario("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadScenariosFromDir(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.yaml", "b.yaml", "skip.txt"} {
		path := filepath.Join(dir, name)
		content := "name: " + name[:len(name)-5] + "\nsteps: []\n"
		if filepath.Ext(name) != ".yaml" {
			content = "not yaml"
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	scenarios, err := LoadScenariosFromDir(dir)
	if err != nil {
		t.Fatalf("LoadScenariosFromDir: %v", err)
	}
	if len(scenarios) != 2 {
		t.Errorf("expected 2 yaml scenarios, got %d", len(scenarios))
	}
}

func TestScenarioNames(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.yaml")
	if err := os.WriteFile(path, []byte("name: x\nsteps: []\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	names, err := ScenarioNames(dir)
	if err != nil {
		t.Fatalf("ScenarioNames: %v", err)
	}
	if len(names) != 1 || names[0] != "x" {
		t.Errorf("expected [x], got %v", names)
	}
}
