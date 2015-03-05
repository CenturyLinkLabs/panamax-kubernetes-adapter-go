/*
Copyright 2014 Google Inc. All rights reserved.

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

package client

import (
	"errors"
	"fmt"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/runtime"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/watch"
)

// EventNamespacer can return an EventInterface for the given namespace.
type EventNamespacer interface {
	Events(namespace string) EventInterface
}

// EventInterface has methods to work with Event resources
type EventInterface interface {
	Create(event *api.Event) (*api.Event, error)
	Update(event *api.Event) (*api.Event, error)
	List(label, field labels.Selector) (*api.EventList, error)
	Get(name string) (*api.Event, error)
	Watch(label, field labels.Selector, resourceVersion string) (watch.Interface, error)
	// Search finds events about the specified object
	Search(objOrRef runtime.Object) (*api.EventList, error)
}

// events implements Events interface
type events struct {
	client    *Client
	namespace string
}

// newEvents returns a new events object.
func newEvents(c *Client, ns string) *events {
	return &events{
		client:    c,
		namespace: ns,
	}
}

// Create makes a new event. Returns the copy of the event the server returns,
// or an error. The namespace to create the event within is deduced from the
// event; it must either match this event client's namespace, or this event
// client must have been created with the "" namespace.
func (e *events) Create(event *api.Event) (*api.Event, error) {
	if e.namespace != "" && event.Namespace != e.namespace {
		return nil, fmt.Errorf("can't create an event with namespace '%v' in namespace '%v'", event.Namespace, e.namespace)
	}
	result := &api.Event{}
	err := e.client.Post().
		NamespaceIfScoped(event.Namespace, len(event.Namespace) > 0).
		Resource("events").
		Body(event).
		Do().
		Into(result)
	return result, err
}

// Update modifies an existing event. It returns the copy of the event that the server returns,
// or an error. The namespace and key to update the event within is deduced from the event. The
// namespace must either match this event client's namespace, or this event client must have been
// created with the "" namespace. Update also requires the ResourceVersion to be set in the event
// object.
func (e *events) Update(event *api.Event) (*api.Event, error) {
	if len(event.ResourceVersion) == 0 {
		return nil, fmt.Errorf("invalid event update object, missing resource version: %#v", event)
	}
	result := &api.Event{}
	err := e.client.Put().
		NamespaceIfScoped(event.Namespace, len(event.Namespace) > 0).
		Resource("events").
		Name(event.Name).
		Body(event).
		Do().
		Into(result)
	return result, err
}

// List returns a list of events matching the selectors.
func (e *events) List(label, field labels.Selector) (*api.EventList, error) {
	result := &api.EventList{}
	err := e.client.Get().
		NamespaceIfScoped(e.namespace, len(e.namespace) > 0).
		Resource("events").
		SelectorParam("labels", label).
		SelectorParam("fields", field).
		Do().
		Into(result)
	return result, err
}

// Get returns the given event, or an error.
func (e *events) Get(name string) (*api.Event, error) {
	if len(name) == 0 {
		return nil, errors.New("name is required parameter to Get")
	}

	result := &api.Event{}
	err := e.client.Get().
		NamespaceIfScoped(e.namespace, len(e.namespace) > 0).
		Resource("events").
		Name(name).
		Do().
		Into(result)
	return result, err
}

// Watch starts watching for events matching the given selectors.
func (e *events) Watch(label, field labels.Selector, resourceVersion string) (watch.Interface, error) {
	return e.client.Get().
		Prefix("watch").
		NamespaceIfScoped(e.namespace, len(e.namespace) > 0).
		Resource("events").
		Param("resourceVersion", resourceVersion).
		SelectorParam("labels", label).
		SelectorParam("fields", field).
		Watch()
}

// Search finds events about the specified object. The namespace of the
// object must match this event's client namespace unless the event client
// was made with the "" namespace.
func (e *events) Search(objOrRef runtime.Object) (*api.EventList, error) {
	ref, err := api.GetReference(objOrRef)
	if err != nil {
		return nil, err
	}
	if e.namespace != "" && ref.Namespace != e.namespace {
		return nil, fmt.Errorf("won't be able to find any events of namespace '%v' in namespace '%v'", ref.Namespace, e.namespace)
	}
	fields := labels.Set{}
	if ref.Kind != "" {
		fields["involvedObject.kind"] = ref.Kind
	}
	if ref.Namespace != "" {
		fields["involvedObject.namespace"] = ref.Namespace
	}
	if ref.Name != "" {
		fields["involvedObject.name"] = ref.Name
	}
	if ref.UID != "" {
		fields["involvedObject.uid"] = string(ref.UID)
	}
	return e.List(labels.Everything(), fields.AsSelector())
}
