package eventsource

import (
	"strings"

	apiclient "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned"
	v1alpha1 "github.com/argoproj/argo-events/pkg/client/eventsource/clientset/versioned/typed/eventsource/v1alpha1"
	clients "github.com/web-seven/provider-argo-events/internal/client"
)

const (
	errorNotFound = "code = NotFound desc = repo"
)

// ServiceClient wraps the functions to connect to argocd repositories
type ServiceClient interface {
	v1alpha1.EventSourceInterface
}

// NewEventSourceServiceClient creates a new API client from a set of config options, or fails fatally if the new client creation fails.
func NewEventSourceServiceClient(config *clients.ClientOptions) ServiceClient {
	repoIf := apiclient.NewForConfigOrDie(&config.Client).
		ArgoprojV1alpha1().
		EventSources(config.Namespace).(ServiceClient)
	return repoIf
}

// IsErrorEventSourceNotFound helper function to test for errorNotFound error.
func IsErrorEventSourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), errorNotFound)
}
