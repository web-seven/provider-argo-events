/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package eventsource

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	esv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/web-seven/provider-argo-events/apis/events/v1alpha1"
	apisv1alpha1 "github.com/web-seven/provider-argo-events/apis/v1alpha1"
	clients "github.com/web-seven/provider-argo-events/internal/client"
	"github.com/web-seven/provider-argo-events/internal/client/eventsource"
	"github.com/web-seven/provider-argo-events/internal/features"
)

const (
	errTrackPCUsage   = "cannot track ProviderConfig usage"
	errNewClient      = "cannot create new Service"
	errNotEventSource = "managed resource is not a EventSource custom resource"
	errGetPC          = "cannot get ProviderConfig"
	errGetCreds       = "cannot get credentials"
	errListFailed     = "cannot list EventSources"
)

// A NoOpService does nothing.
type NoOpService struct{}

// Setup adds a controller that reconciles EventSource managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.EventSourceGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.EventSourceGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:            mgr.GetClient(),
			usage:           resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newArgoClientFn: eventsource.NewEventSourceServiceClient}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.EventSource{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube            client.Client
	usage           resource.Tracker
	newArgoClientFn func(c *clients.ClientOptions) eventsource.ServiceClient
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	cfg, err := clients.GetConfig(ctx, c.kube, mg)
	if err != nil {
		return nil, err
	}

	argoClient := c.newArgoClientFn(cfg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	fmt.Println("Connect: ")
	fmt.Println("")

	return &external{kube: c.kube, client: argoClient}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube   client.Client
	client eventsource.ServiceClient
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {

	cr, ok := mg.(*v1alpha1.EventSource)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotEventSource)
	}

	var name = meta.GetExternalName(cr)
	if name == "" {
		return managed.ExternalObservation{}, nil
	}

	var evss *esv1alpha1.EventSourceList
	evss, err := c.client.List(ctx, v1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", name),
	})
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errListFailed)
	}
	evs := &esv1alpha1.EventSource{}
	for _, item := range evss.Items {
		if item.Name == name {
			evs = item.DeepCopy()
		}
	}

	res2print, _ := json.MarshalIndent(evs, "", "  ")
	fmt.Println(string(res2print))

	if evs.Name == "" {
		return managed.ExternalObservation{}, nil
	}

	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: true,

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.EventSource)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotEventSource)
	}
	fmt.Printf("Creating: %+v", cr)
	fmt.Println("")
	es := &esv1alpha1.EventSource{}
	es.Spec = cr.Spec.ForProvider
	c.client.Create(ctx, es, v1.CreateOptions{})
	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.EventSource)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotEventSource)
	}

	fmt.Printf("Updating: %+v", cr)
	fmt.Println("")
	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.EventSource)
	if !ok {
		return errors.New(errNotEventSource)
	}

	fmt.Printf("Deleting: %+v", cr)
	fmt.Println("")

	return nil
}
