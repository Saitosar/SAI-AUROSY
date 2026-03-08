# MVP V1 Architecture

Архитектура минимального сценария «Status + Safe Stop».

## Diagram

```mermaid
flowchart TB
    subgraph Console [Operator Console]
        UI[React SPA]
    end

    subgraph ControlPlane [Control Plane]
        API[API Server]
        Registry[Fleet Registry]
        Arbiter[Command Arbiter]
        Safety[Safety Supervisor]
    end

    subgraph NATS [NATS]
        TelemetryTopic[telemetry.robots.*]
        CommandTopic[commands.robots.*]
    end

    subgraph Adapter [AGIBOT Adapter]
        Bridge[ROS2 / Mock Bridge]
    end

    subgraph Robot [AGIBOT X1]
        X1[X1 Infer / AimRT]
    end

    UI -->|GET /robots| API
    UI -->|POST /command| API
    UI -->|SSE /telemetry/stream| API
    API --> Registry
    API --> Arbiter
    Arbiter --> Safety
    API -->|publish| CommandTopic
    API -->|subscribe| TelemetryTopic
    Bridge -->|subscribe| CommandTopic
    Bridge -->|publish| TelemetryTopic
    Bridge <-->|ROS2| X1
```

## Data Flow

### Telemetry
1. X1 Infer публикует `/joint_states`, `/imu/data` в ROS2
2. AGIBOT Adapter подписывается, маппит в нормализованную телеметрию
3. Adapter публикует в NATS `telemetry.robots.x1-001`
4. Control Plane подписан на `telemetry.robots.>`, стримит в SSE
5. Operator Console получает SSE, обновляет UI

### Safe Stop
1. Оператор нажимает Safe Stop в Console
2. Console: POST /robots/x1-001/command { "command": "safe_stop" }
3. API → Safety Supervisor (allow) → Publish в NATS `commands.robots.x1-001`
4. AGIBOT Adapter получает, публикует в ROS2 `/start_control`
5. X1 Infer RL Control переводит робота в idle
