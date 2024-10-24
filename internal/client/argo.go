package clients

import (
	"context"

	"github.com/pkg/errors"
	"github.com/web-seven/provider-argo-events/apis/events/v1alpha1"
	apisv1alpha1 "github.com/web-seven/provider-argo-events/apis/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

const (
	errNotEventSource = "managed resource is not a EventSource custom resource"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
)

// ClientOptions hold address, security, and other settings for the API client.
type ClientOptions struct {
	Namespace string
	Client    rest.Config
}

// GetConfig constructs a Config that can be used to authenticate to argocd
// API by the argocd Go client
func GetConfig(ctx context.Context, c client.Client, mg resource.Managed) (*ClientOptions, error) {
	switch {
	case mg.GetProviderConfigReference() != nil:
		return UseProviderConfig(ctx, c, mg)
	default:
		return nil, errors.New("providerConfigRef is not given")
	}
}

// UseProviderConfig to produce a config that can be used to authenticate to AWS.
func UseProviderConfig(ctx context.Context, c client.Client, mg resource.Managed) (*ClientOptions, error) {
	cr, ok := mg.(*v1alpha1.EventSource)
	if !ok {
		return nil, errors.New(errNotEventSource)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	// cd := pc.Spec.Credentials
	// data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c, cd.CommonCredentialSelectors)
	// if err != nil {
	// 	return nil, errors.Wrap(err, errGetCreds)
	// }
	return &ClientOptions{
		Namespace: pc.Spec.Namespace,
	}, nil
}
