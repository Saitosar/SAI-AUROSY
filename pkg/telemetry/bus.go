package telemetry

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/sai-aurosy/platform/pkg/hal"
)

const (
	TopicTelemetryPrefix = "telemetry.robots"
	TopicCommandsPrefix  = "commands.robots"
)

// Bus is the Telemetry/Event bus over NATS.
type Bus struct {
	nc *nats.Conn
}

// NewBus creates a new NATS-based telemetry bus.
func NewBus(url string) (*Bus, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	return &Bus{nc: nc}, nil
}

// PublishTelemetry publishes telemetry for a robot.
func (b *Bus) PublishTelemetry(t *hal.Telemetry) error {
	topic := fmt.Sprintf("%s.%s", TopicTelemetryPrefix, t.RobotID)
	data, err := json.Marshal(t)
	if err != nil {
		return err
	}
	return b.nc.Publish(topic, data)
}

// SubscribeTelemetry subscribes to telemetry for a robot.
func (b *Bus) SubscribeTelemetry(robotID string, handler func(*hal.Telemetry)) (*nats.Subscription, error) {
	topic := fmt.Sprintf("%s.%s", TopicTelemetryPrefix, robotID)
	return b.nc.Subscribe(topic, func(msg *nats.Msg) {
		var t hal.Telemetry
		if err := json.Unmarshal(msg.Data, &t); err != nil {
			return
		}
		handler(&t)
	})
}

// SubscribeAllTelemetry subscribes to telemetry for all robots.
func (b *Bus) SubscribeAllTelemetry(handler func(*hal.Telemetry)) (*nats.Subscription, error) {
	topic := fmt.Sprintf("%s.>", TopicTelemetryPrefix)
	return b.nc.Subscribe(topic, func(msg *nats.Msg) {
		var t hal.Telemetry
		if err := json.Unmarshal(msg.Data, &t); err != nil {
			return
		}
		handler(&t)
	})
}

// PublishCommand publishes a command to a robot.
func (b *Bus) PublishCommand(cmd *hal.Command) error {
	topic := fmt.Sprintf("%s.%s", TopicCommandsPrefix, cmd.RobotID)
	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	return b.nc.Publish(topic, data)
}

// SubscribeCommands subscribes to commands for a robot.
func (b *Bus) SubscribeCommands(robotID string, handler func(*hal.Command)) (*nats.Subscription, error) {
	topic := fmt.Sprintf("%s.%s", TopicCommandsPrefix, robotID)
	return b.nc.Subscribe(topic, func(msg *nats.Msg) {
		var c hal.Command
		if err := json.Unmarshal(msg.Data, &c); err != nil {
			return
		}
		handler(&c)
	})
}

// SubscribeAllCommands subscribes to commands for all robots (commands.robots.>).
func (b *Bus) SubscribeAllCommands(handler func(*hal.Command)) (*nats.Subscription, error) {
	topic := fmt.Sprintf("%s.>", TopicCommandsPrefix)
	return b.nc.Subscribe(topic, func(msg *nats.Msg) {
		var c hal.Command
		if err := json.Unmarshal(msg.Data, &c); err != nil {
			return
		}
		handler(&c)
	})
}

// Close closes the NATS connection.
func (b *Bus) Close() {
	b.nc.Close()
}

// IsConnected returns true if the NATS connection is active.
func (b *Bus) IsConnected() bool {
	return b.nc.Status() == nats.CONNECTED
}
