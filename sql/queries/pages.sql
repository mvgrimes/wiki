-- name: CreatePage :one
INSERT INTO pages (slug, title)
VALUES (?, ?)
RETURNING id, slug, title, created_at, updated_at;

-- name: GetPageBySlug :one
SELECT id, slug, title, created_at, updated_at
FROM pages
WHERE slug = ?;

-- name: ListPages :many
SELECT id, slug, title, created_at, updated_at
FROM pages
ORDER BY updated_at DESC;

-- name: CreateRevision :one
INSERT INTO revisions (page_id, body, author, checksum)
VALUES (?, ?, ?, ?)
RETURNING id, page_id, body, author, checksum, created_at;

-- name: ListRevisionsByPage :many
SELECT id, page_id, body, author, checksum, created_at
FROM revisions
WHERE page_id = ?
ORDER BY created_at DESC;

-- name: CreateAttachment :one
INSERT INTO attachments (page_id, revision_id, file_name, content_type, size_bytes, storage_key)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, page_id, revision_id, file_name, content_type, size_bytes, storage_key, created_at;

-- name: ListAttachmentsByPage :many
SELECT id, page_id, revision_id, file_name, content_type, size_bytes, storage_key, created_at
FROM attachments
WHERE page_id = ?
ORDER BY created_at DESC;
