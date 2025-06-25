package runtime

import (
	"context"
	"net/http"
)

type Service interface {
	// Returns the service name (e.g. "storage", "bigquery")
	Name() string

	// Initializes the service and its dependencies
	Initialize(ctx context.Context) error

	// Starts the service
	Start(ctx context.Context) error

	// Gracefully shuts down service
	Stop(ctx context.Context) error

	// Returns the service's health status
	Health(ctx context.Context) error

	// Returns the http handler for this service
	Handler() http.Handler

	// Returns all the url patterns this service handles
	Routes() []string
}

// Manages all registered services
type ServiceRegistry struct {
	services map[string]Service
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]Service),
	}
}

func (sr *ServiceRegistry) Register(service Service) {
	sr.services[service.Name()] = service
}

func (sr *ServiceRegistry) Get(name string) (Service, bool) {
	service, exists := sr.services[name]
	return service, exists
}

func (sr *ServiceRegistry) All() map[string]Service {
	return sr.services
}
