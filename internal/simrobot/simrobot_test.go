package simrobot

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sai-aurosy/platform/pkg/control-plane/registry"
	"github.com/sai-aurosy/platform/pkg/hal"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

func TestBuildTelemetryFromState(t *testing.T) {
	s := SimState{
		RobotID:          "sim-001",
		Online:           true,
		Mode:             ModeNavigating,
		Position:         Position{X: 5, Y: 3},
		TargetPosition:   Position{X: 10, Y: 5},
		DistanceToTarget: 5.4,
	}
	tel := buildTelemetryFromState(s)
	if tel.RobotID != "sim-001" {
		t.Errorf("RobotID: got %s", tel.RobotID)
	}
	if !tel.Online {
		t.Error("expected Online true")
	}
	if tel.Position != "5.00,3.00,0" {
		t.Errorf("Position: got %s", tel.Position)
	}
	if tel.TargetPosition != "10.00,5.00,0" {
		t.Errorf("TargetPosition: got %s", tel.TargetPosition)
	}
	if tel.DistanceToTarget == nil || *tel.DistanceToTarget != 5.4 {
		t.Errorf("DistanceToTarget: got %v", tel.DistanceToTarget)
	}
	if tel.CurrentTask != "walk" {
		t.Errorf("CurrentTask: got %s", tel.CurrentTask)
	}
	if !tel.MockMode {
		t.Error("expected MockMode true")
	}
}

func TestParseCoordinates(t *testing.T) {
	tests := []struct {
		in    string
		x, y  float64
		valid bool
	}{
		{"15.00,5.00,0", 15, 5, true},
		{"0,0", 0, 0, true},
		{"1.5,2.5", 1.5, 2.5, true},
		{"", 0, 0, false},
		{"x,y", 0, 0, false},
	}
	for _, tt := range tests {
		x, y, err := parseCoordinates(tt.in)
		valid := err == nil
		if valid != tt.valid {
			t.Errorf("parseCoordinates(%q): valid=%v, want %v", tt.in, valid, tt.valid)
		}
		if valid && (x != tt.x || y != tt.y) {
			t.Errorf("parseCoordinates(%q): got (%.2f,%.2f), want (%.2f,%.2f)", tt.in, x, y, tt.x, tt.y)
		}
	}
}

func TestDistance(t *testing.T) {
	a := Position{X: 0, Y: 0}
	b := Position{X: 3, Y: 4}
	if d := distance(a, b); d != 5 {
		t.Errorf("distance(0,0, 3,4): got %.2f, want 5", d)
	}
}

func TestNewSimRobot_RegistersInFleet(t *testing.T) {
	reg := registry.NewMemoryStore()
	bus, err := telemetry.NewBus("nats://localhost:4222")
	if err != nil {
		t.Skipf("NATS not available: %v", err)
	}
	defer bus.Close()

	robot, err := NewSimRobot(CreateRobotOpts{RobotID: "sim-test", TenantID: "default"}, bus, reg)
	if err != nil {
		t.Fatalf("NewSimRobot: %v", err)
	}
	if robot.ID() != "sim-test" {
		t.Errorf("ID: got %s", robot.ID())
	}

	r := reg.Get("sim-test")
	if r == nil {
		t.Fatal("robot not in registry")
	}
	if r.Vendor != "simulated" {
		t.Errorf("Vendor: got %s", r.Vendor)
	}
	if !hal.HasCapability(r, []string{hal.CapNavigation, hal.CapSpeech}) {
		t.Error("missing capabilities")
	}
}

func TestReplayScript_LoadAndStructure(t *testing.T) {
	script := &ReplayScript{
		Scenario: "test",
		Ticks: []ReplayTick{
			{Position: Position{X: 0, Y: 0}, DistanceToTarget: 10, Online: true},
			{Position: Position{X: 5, Y: 0}, DistanceToTarget: 5, Online: true},
		},
	}
	data, err := json.Marshal(script)
	if err != nil {
		t.Fatal(err)
	}
	var loaded ReplayScript
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}
	if loaded.Scenario != script.Scenario || len(loaded.Ticks) != len(script.Ticks) {
		t.Errorf("roundtrip: got %+v", loaded)
	}
}

func TestSimRobot_StartStopReset(t *testing.T) {
	reg := registry.NewMemoryStore()
	bus, err := telemetry.NewBus("nats://localhost:4222")
	if err != nil {
		t.Skipf("NATS not available: %v", err)
	}
	defer bus.Close()

	robot, err := NewSimRobot(CreateRobotOpts{RobotID: "sim-lifecycle", TenantID: "default"}, bus, reg)
	if err != nil {
		t.Fatalf("NewSimRobot: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := robot.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	robot.Stop()

	robot.Reset()
	s := robot.State()
	if s.Mode != ModeIdle || s.Position.X != 0 || s.Position.Y != 0 {
		t.Errorf("Reset: got mode=%s pos=(%.1f,%.1f)", s.Mode, s.Position.X, s.Position.Y)
	}
}
