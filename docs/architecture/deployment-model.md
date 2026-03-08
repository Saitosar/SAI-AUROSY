## Deployment Model

SAI AUROSY follows a hybrid architecture:

Edge Layer
Runs on the customer site and provides low-latency robot control.

Includes:
- robot adapters
- safety supervisor
- local event broker
- teleoperation relay

Cloud Control Plane
Runs in the cloud and provides platform management.

Includes:
- fleet management
- orchestration
- analytics
- enterprise integrations

This architecture allows robots to operate even in unstable network environments.