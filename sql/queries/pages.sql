-- name: GetPageBySlug :one
SELECT id, slug, title, body, created_at, updated_at
FROM pages
WHERE slug = ?;

-- name: ListPages :many
SELECT id, slug, title, body, created_at, updated_at
FROM pages
ORDER BY updated_at DESC;
