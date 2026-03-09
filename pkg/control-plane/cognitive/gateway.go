package cognitive

import (
	"context"
)

// Gateway is the abstraction for AI services (navigation, recognition, planning).
type Gateway interface {
	// Navigate computes a path from A to B for the given robot.
	Navigate(ctx context.Context, req NavigateRequest) (*NavigateResult, error)
	// Recognize detects objects in sensor data.
	Recognize(ctx context.Context, req RecognizeRequest) (*RecognizeResult, error)
	// Plan generates a sequence of steps for a task type.
	Plan(ctx context.Context, req PlanRequest) (*PlanResult, error)
}
