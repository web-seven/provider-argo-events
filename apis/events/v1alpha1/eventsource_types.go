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

package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	esv1alpha1 "github.com/argoproj/argo-events/pkg/apis/eventsource/v1alpha1"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
)

// EventSourceObservation are the observable fields of a EventSource.
type EventSourceObservation struct {
	ObservableField string `json:"observableField,omitempty"`
}

// A EventSourceSpec defines the desired state of a EventSource.
type EventSourceSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       esv1alpha1.EventSourceSpec `json:"forProvider"`
}

// A EventSourceStatus represents the observed state of a EventSource.
type EventSourceStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          EventSourceObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true

// A EventSource is an example API type.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,managed,argo}
type EventSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventSourceSpec   `json:"spec"`
	Status EventSourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EventSourceList contains a list of EventSource
type EventSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EventSource `json:"items"`
}

// EventSource type metadata.
var (
	EventSourceKind             = reflect.TypeOf(EventSource{}).Name()
	EventSourceGroupKind        = schema.GroupKind{Group: Group, Kind: EventSourceKind}.String()
	EventSourceKindAPIVersion   = EventSourceKind + "." + SchemeGroupVersion.String()
	EventSourceGroupVersionKind = SchemeGroupVersion.WithKind(EventSourceKind)
)

func init() {
	SchemeBuilder.Register(&EventSource{}, &EventSourceList{})
}
