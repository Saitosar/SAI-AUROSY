# SAI AUROSY

Enterprise Multi-Robot Workforce Platform.

SAI AUROSY is a platform designed to transform robots from different vendors into a unified enterprise workforce.

The platform provides:

- fleet management
- robot task orchestration
- enterprise integrations
- analytics and telemetry
- multi-robot coordination

SAI AUROSY supports multiple robotics ecosystems:

- AGIBOT
- UNITREE
- ROS-based robots
- future robotics vendors

---

# Quick Start (MVP V1)

```bash
docker compose up -d
```

Open [http://localhost:3000](http://localhost:3000) — Operator Console. Robot x1-001 appears with real-time telemetry. Click **Safe Stop** to send the command.

See [Status + Safe Stop Runbook](docs/implementation/status-safe-stop.md) and [MVP V1 Overview](docs/implementation/mvp-v1-overview.md).

---

# Documentation

## Product

- [Product overview](docs/product/platform-overview.md)
- [Problem statement](docs/product/problem-statement.md)
- [Value proposition](docs/product/value-proposition.md)
- [Use cases](docs/product/use-cases.md)
- [Roadmap](docs/product/roadmap.md)

---

## Architecture

- [Platform architecture](docs/architecture/platform-architecture.md)
- [MVP V1 architecture](docs/architecture/mvp-v1-architecture.md)

## Implementation

- [MVP V1 overview](docs/implementation/mvp-v1-overview.md)
- [Status + Safe Stop runbook](docs/implementation/status-safe-stop.md)
- [Multi-robot architecture](docs/architecture/multi-robot-architecture.md)
- [Adapter layer](docs/architecture/adapter-layer.md)
- [Deployment model (edge/cloud)](docs/architecture/deployment-model.md)

---

## Robot Vendors

- [AGIBOT integration notes](docs/vendors/agibot.md)
- [UNITREE integration notes](docs/vendors/unitree.md)

---

## Glossary

- [Robotics terminology](docs/glossary/robotics-terms.md)
