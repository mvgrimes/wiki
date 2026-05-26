package server

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"wiki/internal/db"
)

const defaultUploadMaxBytes int64 = 10 << 20

var safeObjectChars = regexp.MustCompile(`[^a-z0-9._-]+`)

var errAttachmentNotFound = errors.New("attachment not found")

type ObjectStore interface {
	PutObject(
		ctx context.Context,
		bucket, key, contentType string,
		body io.Reader,
		size int64,
	) error
	GetObject(ctx context.Context, bucket, key string) (AttachmentObject, error)
}

type AttachmentObject struct {
	Body        io.ReadCloser
	ContentType string
}

func uploadHandler(cfg Config) echo.HandlerFunc {
	maxBytes := cfg.UploadMaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultUploadMaxBytes
	}

	return func(c echo.Context) error {
		if cfg.Queries == nil || cfg.UploadStore == nil ||
			strings.TrimSpace(cfg.UploadBucket) == "" {
			return uploadError(c, http.StatusServiceUnavailable, "upload service is not configured", nil)
		}

		if err := c.Request().ParseMultipartForm(maxBytes); err != nil {
			return uploadError(c, http.StatusBadRequest, "invalid multipart form", err)
		}

		slug, err := normalizeWikiSlug(c.FormValue("page_slug"))
		if err != nil {
			return uploadError(c, http.StatusBadRequest, "page_slug is required", err)
		}

		page, err := cfg.Queries.GetPageBySlug(c.Request().Context(), slug)
		if errors.Is(err, sql.ErrNoRows) {
			page, err = cfg.Queries.CreatePage(
				c.Request().Context(),
				db.CreatePageParams{Slug: slug, Title: titleFromSlug(slug)},
			)
		}
		if err != nil {
			return err
		}

		fileHeader, err := uploadFormFile(c)
		if err != nil {
			return uploadError(c, http.StatusBadRequest, "file is required", err)
		}

		contentType, data, err := readAndValidateUpload(fileHeader, maxBytes)
		if err != nil {
			return uploadError(c, http.StatusBadRequest, err.Error(), err)
		}

		key := buildObjectKey(slug, fileHeader.Filename, data)
		if err := cfg.UploadStore.PutObject(
			c.Request().Context(),
			cfg.UploadBucket,
			key,
			contentType,
			bytes.NewReader(data),
			int64(len(data)),
		); err != nil {
			return uploadError(
				c,
				http.StatusBadGateway,
				"failed to store upload",
				err,
				map[string]any{
					"upload_bucket": cfg.UploadBucket,
					"upload_key":    key,
				},
			)
		}

		revisionID, err := revisionIDFromForm(c.FormValue("revision_id"))
		if err != nil {
			return uploadError(c, http.StatusBadRequest, err.Error(), err)
		}

		attachment, err := cfg.Queries.CreateAttachment(
			c.Request().Context(),
			db.CreateAttachmentParams{
				PageID:      page.ID,
				RevisionID:  revisionID,
				FileName:    sanitizeFilename(fileHeader.Filename),
				ContentType: contentType,
				SizeBytes:   int64(len(data)),
				StorageKey:  key,
			},
		)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusCreated, map[string]any{
			"success":      true,
			"image_path":   attachmentPath(attachment.StorageKey),
			"id":           attachment.ID,
			"page_id":      attachment.PageID,
			"revision_id":  nullableInt64(attachment.RevisionID),
			"file_name":    attachment.FileName,
			"content_type": attachment.ContentType,
			"size_bytes":   attachment.SizeBytes,
			"storage_key":  attachment.StorageKey,
		})
	}
}

func uploadFormFile(c echo.Context) (*multipart.FileHeader, error) {
	fileHeader, err := c.FormFile("file")
	if err == nil {
		return fileHeader, nil
	}

	fileHeader, imageErr := c.FormFile("image_file")
	if imageErr == nil {
		return fileHeader, nil
	}

	return nil, err
}

func readAndValidateUpload(header *multipart.FileHeader, maxBytes int64) (string, []byte, error) {
	f, err := header.Open()
	if err != nil {
		return "", nil, fmt.Errorf("failed to open upload")
	}
	defer f.Close()

	limited := io.LimitReader(f, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read upload")
	}
	if len(data) == 0 {
		return "", nil, fmt.Errorf("file is empty")
	}
	if int64(len(data)) > maxBytes {
		return "", nil, fmt.Errorf("file exceeds %d bytes", maxBytes)
	}

	contentType := http.DetectContentType(data)
	if !allowedContentType(contentType) {
		return "", nil, fmt.Errorf("file type is not allowed")
	}

	return contentType, data, nil
}

func allowedContentType(contentType string) bool {
	switch contentType {
	case "image/png",
		"image/jpeg",
		"image/webp",
		"image/gif",
		"application/pdf",
		"text/plain; charset=utf-8":
		return true
	default:
		return false
	}
}

func buildObjectKey(slug, fileName string, data []byte) string {
	checksum := sha256.Sum256(data)
	ext := strings.ToLower(filepath.Ext(fileName))
	cleanSlug := sanitizePathPart(slug)
	if cleanSlug == "" {
		cleanSlug = "page"
	}
	return fmt.Sprintf("attachments/%s/%s%s", cleanSlug, hex.EncodeToString(checksum[:]), ext)
}

func sanitizePathPart(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "/", "-")
	raw = safeObjectChars.ReplaceAllString(raw, "-")
	raw = strings.Trim(raw, "-.")
	return raw
}

func sanitizeFilename(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == "" {
		return "file"
	}
	return base
}

func revisionIDFromForm(raw string) (sql.NullInt64, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return sql.NullInt64{}, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n <= 0 {
		return sql.NullInt64{}, fmt.Errorf("revision_id must be a positive integer")
	}
	return sql.NullInt64{Int64: n, Valid: true}, nil
}

func nullableInt64(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

func attachmentPath(storageKey string) string {
	trimmed := strings.TrimSpace(storageKey)
	if trimmed == "" {
		return ""
	}
	return "/" + strings.TrimPrefix(trimmed, "/")
}

func uploadError(c echo.Context, code int, message string, cause error, contextFields ...map[string]any) error {
	var fields map[string]any
	if len(contextFields) > 0 {
		fields = contextFields[0]
	}
	setRequestError(c, cause, fields)
	return c.JSON(code, map[string]string{"error": message})
}
