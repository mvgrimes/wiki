---
id: wik-g882
status: closed
deps: [wik-4jn1, wik-wb14, wik-dvk8, wik-3rsb]
links: []
created: 2026-05-12T01:55:29Z
type: feature
priority: 1
tags: [server, templates, echo]
---
# Build Echo server, template rendering, and dev/prod template FS strategy

Add server wiring with html/template, embedded templates for prod, and disk-based template loading during development.

## Acceptance Criteria

- Echo server starts with middleware and health endpoints\n- Templates render correctly in development and production modes\n- URL-to-template organization is clear and documented\n- Routing and render error paths tested\n- Basic request logging and panic recovery enabled


## Notes

**2026-05-12T01:56:02Z**

Acceptance criteria clarification: Echo server starts with middleware and health endpoints; templates render in development and production modes; URL-to-template organization is documented; routing/render errors are tested; request logging and panic recovery enabled.
