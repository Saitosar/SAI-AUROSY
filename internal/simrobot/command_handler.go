package simrobot

import (
	"encoding/json"
	"log/slog"
	"math"
	"strconv"
	"strings"

	"github.com/sai-aurosy/platform/pkg/hal"
)

// navigateToPayload matches the navigate_to command payload structure.
type navigateToPayload struct {
	TargetCoordinates  string   `json:"target_coordinates"`
	StoreName          string   `json:"store_name"`
	DestinationNodeID  string   `json:"destination_node_id"`
	Route              []string `json:"route"`
	EstimatedDistance  float64  `json:"estimated_distance"`
}

// speakPayload matches the speak command payload (if used).
type speakPayload struct {
	Text string `json:"text"`
}

// HandleCommand processes a command and updates the robot state.
func HandleCommand(robot *SimRobot, cmd *hal.Command) {
	if cmd == nil || cmd.RobotID != robot.ID() {
		return
	}

	robot.mu.Lock()
	defer robot.mu.Unlock()

	robot.state.LastCommand = cmd.Command
	robot.state.UpdatedAt = cmd.Timestamp

	switch cmd.Command {
	case "navigate_to":
		handleNavigateTo(robot, cmd)
	case "safe_stop":
		handleSafeStop(robot)
	case "release_control":
		handleReleaseControl(robot)
	case "walk_mode":
		handleWalkMode(robot)
	case "stand_mode":
		handleStandMode(robot)
	case "zero_mode":
		handleStandMode(robot) // treat as stand for sim
	case "speak":
		handleSpeak(robot, cmd)
	case "cmd_vel":
		// Simulator accepts cmd_vel but does not change navigation target
		robot.state.Mode = ModeWalk
	default:
		slog.Debug("simrobot unknown command", "robot_id", robot.id, "command", cmd.Command)
	}
}

func handleNavigateTo(robot *SimRobot, cmd *hal.Command) {
	var p navigateToPayload
	if len(cmd.Payload) > 0 {
		_ = json.Unmarshal(cmd.Payload, &p)
	}
	if p.TargetCoordinates == "" {
		slog.Warn("simrobot navigate_to missing target_coordinates", "robot_id", robot.id)
		return
	}

	tx, ty, err := parseCoordinates(p.TargetCoordinates)
	if err != nil {
		slog.Warn("simrobot navigate_to invalid coordinates", "robot_id", robot.id, "coords", p.TargetCoordinates, "error", err)
		return
	}

	robot.state.TargetPosition = Position{X: tx, Y: ty}
	robot.state.RouteNodes = p.Route
	robot.state.RouteIndex = 0
	robot.state.Mode = ModeNavigating
	robot.state.DistanceToTarget = distance(robot.state.Position, robot.state.TargetPosition)
	slog.Info("simrobot navigate_to", "robot_id", robot.id, "target", p.TargetCoordinates)
}

func handleSafeStop(robot *SimRobot) {
	robot.state.Mode = ModeSafeStop
	robot.state.TargetPosition = Position{}
	robot.state.RouteNodes = nil
	robot.state.DistanceToTarget = 0
	slog.Info("simrobot safe_stop", "robot_id", robot.id)
}

func handleReleaseControl(robot *SimRobot) {
	robot.state.Mode = ModeIdle
	robot.state.TargetPosition = Position{}
	robot.state.DistanceToTarget = 0
}

func handleWalkMode(robot *SimRobot) {
	if robot.state.Mode != ModeNavigating && robot.state.Mode != ModeReturning {
		robot.state.Mode = ModeWalk
	}
}

func handleStandMode(robot *SimRobot) {
	if robot.state.Mode != ModeNavigating && robot.state.Mode != ModeReturning {
		robot.state.Mode = ModeStand
	}
}

func handleSpeak(robot *SimRobot, cmd *hal.Command) {
	var p speakPayload
	if len(cmd.Payload) > 0 {
		_ = json.Unmarshal(cmd.Payload, &p)
	}
	robot.state.LastSpokenText = p.Text
	slog.Debug("simrobot speak", "robot_id", robot.id, "text", p.Text)
}

// distance returns Euclidean distance between two positions.
func distance(a, b Position) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// parseCoordinates parses "x,y,z" or "x,y" format.
func parseCoordinates(s string) (x, y float64, err error) {
	parts := strings.Split(strings.TrimSpace(s), ",")
	if len(parts) < 2 {
		return 0, 0, strconv.ErrSyntax
	}
	x, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return 0, 0, err
	}
	y, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}
