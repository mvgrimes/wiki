---
id: wik-txfk
status: closed
deps: [wik-4jn1, wik-wb14]
links: []
created: 2026-05-12T01:55:29Z
type: feature
priority: 1
tags: [db, sqlite, goose, sqlc]
---
# Add Goose migrations and sqlc schema/query pipeline

Define wiki schema for pages, revisions, and attachments; wire migration embedding and sqlc generation.

## Acceptance Criteria

- Migrations are timestamped and reversible where applicable\n-  works with local sqlite file\n- sqlc generates typed DB package under internal/db\n- Repository compiles with generated code committed\n- Integration tests validate schema creation and key query behavior


## Notes

**2026-05-12T01:56:02Z**

Acceptance criteria clarification: 1) migrations timestamped and reversible where applicable; 2) go run ./... migrate up|down|status works with local sqlite file; 3) sqlc generates typed DB package under internal/db; 4) repository compiles with generated code committed; 5) integration tests validate schema creation and key query behavior.
