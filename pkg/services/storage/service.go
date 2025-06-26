package storage

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/thegenem0/glocal/pkg/config"
	"github.com/thegenem0/glocal/pkg/containers"
	"github.com/thegenem0/glocal/pkg/services/base"
	"go.uber.org/zap"
)

type StorageService struct {
	*base.ContainerService
	proxy      *httputil.ReverseProxy
	translator *APITranslator
}

func NewStorageService(
	containerMgr *containers.ContainerManager,
	containerConfig config.ContainerConfig,
	logger *zap.Logger,
) *StorageService {
	contaierService := base.NewContainerServcie(
		"storage",
		"minio",
		containerConfig,
		containerMgr,
		logger,
	)

	service := &StorageService{
		ContainerService: contaierService,
		translator:       NewAPITranslator(logger),
	}

	service.SetRoutes([]string{
		"/storage/*path",
		"/upload/storage/*path",
		"/batch/storage/*path",
	})

	return service
}

func (s *StorageService) Initialize(ctx context.Context) error {
	if err := s.ContainerService.Initialize(ctx); err != nil {
		return err
	}

	minioEndpoint, err := s.GetContainerEndpoint(9000)
	if err != nil {
		return fmt.Errorf("failed to get MinIO endpoint: %w", err)

	}

	// s.logger.Info("MinIO endpoint ready", zap.String("endpoint", minioEndpoint))

	target, err := url.Parse(minioEndpoint)
	if err != nil {
		return fmt.Errorf("failed to parse MinIO URL: %w", err)
	}

	s.proxy = httputil.NewSingleHostReverseProxy(target)
	s.proxy.Director = s.createProxyDirector(target)
	s.proxy.ErrorHandler = s.proxyErrorHandler

	return nil
}

func (s *StorageService) Handler() http.Handler {
	return http.HandlerFunc(s.handleRequest)
}

func (s *StorageService) handleRequest(w http.ResponseWriter, r *http.Request) {
	// s.logger.Debug("Handling storage request",
	// 	zap.String("method", r.Method),
	// 	zap.String("path", r.URL.Path),
	// 	zap.String("query", r.URL.RawQuery))

	if err := s.translator.TranslateRequest(r); err != nil {
		//s.logger.Error("Failed to translate request", zap.Error(err))
		http.Error(w, "Failed to translate request", http.StatusInternalServerError)
		return
	}

	s.proxy.ServeHTTP(w, r)
}

func (s *StorageService) createProxyDirector(target *url.URL) func(*http.Request) {
	return func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host

		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/storage")
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
	}
}

func (s *StorageService) proxyErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	// s.logger.Error("Proxy error", zap.Error(err))
	http.Error(w, "Service temporarily unavailable", http.StatusServiceUnavailable)
}
