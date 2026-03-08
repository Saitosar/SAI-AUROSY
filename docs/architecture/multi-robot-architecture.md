# Multi-Robot Architecture

## Purpose

This document describes how the SAI AUROSY platform supports robots from multiple vendors through a unified architecture.

The goal is to allow heterogeneous robots — including AGIBOT, UNITREE, ROS-based robots, and future robotics vendors — to be managed as a single operational fleet.

The platform abstracts vendor-specific runtimes and exposes a standardized interface for enterprise applications and workflows.

---

# Scope

This document covers:

- multi-vendor robot support
- robot abstraction strategy
- adapter-based integration
- unified command model
- unified telemetry model
- capability normalization

This document does NOT cover:

- robot hardware internals
- vendor firmware
- robot training pipelines
- low-level motor control
- vendor robotics operating systems

---

# Problem Statement

Robotics vendors provide different SDKs, runtimes, communication protocols, and data models.

Examples include:

- ROS / ROS2
- proprietary SDKs
- DDS-based communication
- vendor-specific APIs

Without a unifying architecture, enterprise systems must integrate separately with each robot vendor.

This leads to:

- high integration complexity
- vendor lock-in
- fragmented monitoring
- inconsistent command models
- difficult scaling of robot fleets

SAI AUROSY solves this by introducing a multi-robot abstraction architecture.

---

# Architecture Goal

The architecture allows heterogeneous robots to appear as standardized operational resources within the SAI AUROSY platform.

The platform exposes a unified model for:

- robot identity
- robot status
- robot capabilities
- robot telemetry
- robot commands
- robot events
- robot tasks

---

# High-Level Architecture

```mermaid
flowchart TB

subgraph Platform
A[Business Applications]
B[Workforce Platform]
C[Control Plane]
D[Robot Abstraction Layer]
end

subgraph Adapters
E1[AGIBOT Adapter]
E2[UNITREE Adapter]
E3[ROS Adapter]
E4[Future Adapter]
end

subgraph Robots
F1[AGIBOT Robots]
F2[UNITREE Robots]
F3[ROS Robots]
F4[Future Robots]
end

A --> B
B --> C
C --> D
D --> E1
D --> E2
D --> E3
D --> E4

E1 --> F1
E2 --> F2
E3 --> F3
E4 --> F4