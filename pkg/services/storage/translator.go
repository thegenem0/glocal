package storage

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type APITranslator struct {
	logger *zap.Logger
}

func NewAPITranslator(logger *zap.Logger) *APITranslator {
	return &APITranslator{
		logger: logger,
	}
}

func (t *APITranslator) TranslateRequest(r *http.Request) error {
	origPath := r.URL.Path

	switch {
	case strings.HasPrefix(r.URL.Path, "/storage/v1/b/"):
		t.translateBucketAPI(r)
	case strings.HasPrefix(r.URL.Path, "/upload/storage/v1/b/"):
		t.translateUploadAPI(r)

	case strings.HasPrefix(r.URL.Path, "/batch/storage/v1/"):
		t.translateBatchAPI(r)

	default:
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/storage")
	}

	t.translateHeaders(r)

	t.logger.Debug("Request translated",
		zap.String("original_path", origPath),
		zap.String("translated_path", r.URL.Path))

	return nil
}

func (t *APITranslator) translateBucketAPI(r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/storage/v1/b/")
	parts := strings.SplitN(path, "/", 3)

	if len(parts) >= 1 {
		bucketName := parts[0]

		if len(parts) >= 3 && parts[1] == "o" {
			objectName := parts[2]
			r.URL.Path = "/" + bucketName + "/" + objectName
		} else {
			r.URL.Path = "/" + bucketName
		}
	}
}

func (t *APITranslator) translateUploadAPI(r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/upload/storage/v1/b")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) >= 1 {
		bucketName := parts[0]
		r.URL.Path = "/" + bucketName + "/"
	}
}

func (t *APITranslator) translateBatchAPI(r *http.Request) {
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/batch/storage/v1")
}

func (t *APITranslator) translateHeaders(r *http.Request) {
	r.Header.Del("X-Goog-API-Key")
	r.Header.Del("X-Goog-User-Project")

	if auth := r.Header.Get("Authorization"); auth != "" {
		r.Header.Del("Authorization")
	}
}
