# SAI AUROSY Go SDK

Go client for the SAI AUROSY Control Plane API.

## Installation

```bash
go get github.com/sai-aurosy/platform/sdk/go
```

## Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/sai-aurosy/platform/sdk/go"
)

func main() {
    client := sdk.New("http://localhost:8080", "your-api-key")

    // List robots
    robots, err := client.ListRobots(context.Background(), "")
    if err != nil {
        log.Fatal(err)
    }
    for _, r := range robots {
        fmt.Printf("Robot %s: %s %s\n", r.ID, r.Vendor, r.Model)
    }

    // Create a task
    task, err := client.CreateTask(context.Background(), sdk.CreateTaskRequest{
        RobotID:    "r1",
        ScenarioID: "patrol",
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Created task %s\n", task.ID)

    // List tasks
    tasks, err := client.ListTasks(context.Background(), &sdk.ListTasksOptions{
        RobotID: "r1",
    })
    if err != nil {
        log.Fatal(err)
    }
    for _, t := range tasks {
        fmt.Printf("Task %s: %s\n", t.ID, t.Status)
    }
}
```

## Multi-Tenant

Pass `tenantID` to `ListRobots` and `ListTasks` when using an administrator key to filter by tenant:

```go
robots, err := client.ListRobots(ctx, "tenant-123")
tasks, err := client.ListTasks(ctx, &sdk.ListTasksOptions{TenantID: "tenant-123"})
```

## See Also

- [Integration Guide](../../docs/integration/README.md)
- [API Reference](../../docs/integration/api-reference.md)
