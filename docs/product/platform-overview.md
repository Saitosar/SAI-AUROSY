# SAI AUROSY Platform Overview

SAI AUROSY is an enterprise platform that enables organizations to deploy, manage, and orchestrate robots from multiple vendors as a unified workforce.

The platform abstracts the complexity of robotics hardware and vendor-specific software stacks and exposes a unified interface for enterprise applications.

The system supports robots from:

- AGIBOT
- UNITREE
- ROS-based robots
- future robotics vendors

---

# Mission

To make robots deployable, manageable, and scalable in enterprise environments.

---

# Platform Capabilities

SAI AUROSY provides the following core capabilities:

Fleet Management  
Managing large fleets of robots across multiple locations.

Robot Task Management  
Defining and executing robot missions and scenarios.

Workforce Orchestration  
Coordinating multiple robots to perform enterprise workflows.

Enterprise Integration  
Integrating robots with CRM, ERP, ticketing systems, and internal APIs. OAuth 2.0 for third-party apps, webhooks for events, REST API documentation, and Go SDK. See [Integration Guide](../integration/README.md).

Analytics  
Collecting and analyzing robot telemetry and operational data.

Multi-Robot Coordination  
Allowing robots from different vendors to operate within the same operational environment.

---

# Platform Concept

The platform sits above vendor robotics stacks.

Robots
↓
Vendor SDK / Runtime
↓
Robot Adapter Layer
↓
SAI AUROSY Control Plane
↓
Workforce Platform
↓
Business Applications


Target Industries

Retail
Shopping malls
Museums
Airports
Logistics centers
Smart cities
