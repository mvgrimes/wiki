---
id: wik-wunu
status: closed
deps: [wik-4jn1, wik-txfk, wik-3rsb]
links: []
created: 2026-05-12T01:55:29Z
type: feature
priority: 1
tags: [s3, uploads, attachments]
---
# Implement S3-backed file/image upload endpoint and attachment metadata

Add upload API for drag/drop usage; store file metadata in sqlite and object data in S3-compatible storage.

## Acceptance Criteria

- Upload endpoint validates size/type and auth\n- Files stored with deterministic/object-safe keys\n- Metadata persisted and associated to page/revision\n- Errors are surfaced cleanly for client JS\n- Tests include success/failure paths and storage mock coverage


## Notes

**2026-05-12T01:56:02Z**

Acceptance criteria clarification: upload endpoint validates size/type/auth; files stored with deterministic safe keys; metadata persisted and associated with page/revision; errors surfaced cleanly for JS; tests cover success/failure and storage mocks.
