package validation

import (
	"fmt"
	"strings"
)

// CheckEventSequence verifies that events appear in the given order.
// Events in expected may be a subset; order must be preserved.
func CheckEventSequence(collected []CollectedEvent, expected []string) (bool, string, map[string]interface{}) {
	if len(expected) == 0 {
		return true, "no events to check", nil
	}
	collectedTypes := make([]string, 0, len(collected))
	for _, e := range collected {
		collectedTypes = append(collectedTypes, e.Type)
	}
	idx := 0
	for _, want := range expected {
		found := false
		for idx < len(collectedTypes) {
			if collectedTypes[idx] == want {
				found = true
				idx++
				break
			}
			idx++
		}
		if !found {
			return false, fmt.Sprintf("event %s not found in expected order (expected: %v)", want, expected),
				map[string]interface{}{
					"expected":     expected,
					"collected":    collectedTypes,
					"missing":      want,
				}
		}
	}
	return true, fmt.Sprintf("event sequence correct: %v", expected),
		map[string]interface{}{"events": collectedTypes}
}

// CheckEventPresent verifies that at least one event of the given type was emitted.
func CheckEventPresent(collected []CollectedEvent, eventType string) (bool, string, map[string]interface{}) {
	eventType = strings.TrimSpace(eventType)
	for _, e := range collected {
		if e.Type == eventType {
			return true, fmt.Sprintf("event %s present", eventType),
				map[string]interface{}{"event_type": eventType}
		}
	}
	return false, fmt.Sprintf("event %s not found", eventType),
		map[string]interface{}{
			"event_type": eventType,
			"collected":  eventTypes(collected),
		}
}

func eventTypes(collected []CollectedEvent) []string {
	out := make([]string, 0, len(collected))
	for _, e := range collected {
		out = append(out, e.Type)
	}
	return out
}
