# Backend

API and services for the Lia event-discovery MVP.

Planned stack: **Go modular monolith** with PostgreSQL + PostGIS, Redis, and S3 object storage, as described in [`../docs/event_discovery_mvp_technical_stack.md`](../docs/event_discovery_mvp_technical_stack.md).

## Service template

Build the backend from the **`go-microservice-template`** (`/Users/dodonovpavel/gateway_fm/go-microservice-template`) — it provides Cobra/Viper CLI wiring, ldflags versioning, logrus logging, HTTP + gRPC scaffolding, Makefile targets, tests, and CI/CD.

> **Use it as a single modular monolith, not as a microservice.** Despite the template's name, we run one Go service with clearly separated domain modules (per the technical stack doc), not a fleet of microservices. Start with `make rename NEW_NAME=...`, then add domain modules (`events`, `organizers`, `search`, `notifications`, `ai-assistant`, …) inside the one service; only split out a module into its own service later if real load or team structure demands it.

> Scaffolding to be added.
