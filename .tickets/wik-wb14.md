---
id: wik-wb14
status: closed
deps: [wik-4jn1]
links: []
created: 2026-05-12T01:55:29Z
type: feature
priority: 1
tags: [cli, go, cobra]
---
# Implement Cobra CLI with server/migrate/bootstrap commands

Add root command and subcommands matching sharefile command shape: server, migrate (up/down/status), bootstrap.

## Acceptance Criteria

-  starts HTTP app\n-  works against sqlite path\n-  runs migrate up + startup checks\n- Command help text is complete and consistent\n- Unit tests cover command argument handling and error paths


## Notes

**2026-05-12T01:56:02Z**

Acceptance criteria clarification: 1) go run ./... server starts HTTP app; 2) go run ./... migrate up|down|status works against sqlite path; 3) go run ./... bootstrap runs migrate up and startup checks; 4) command help text complete and consistent; 5) unit tests cover command argument handling and error paths.
