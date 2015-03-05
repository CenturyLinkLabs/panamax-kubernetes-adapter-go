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
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/resource"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/labels"
)

func TestResourceQuotaCreate(t *testing.T) {
	ns := api.NamespaceDefault
	resourceQuota := &api.ResourceQuota{
		ObjectMeta: api.ObjectMeta{
			Name:      "abc",
			Namespace: "foo",
		},
		Spec: api.ResourceQuotaSpec{
			Hard: api.ResourceList{
				api.ResourceCPU:                    resource.MustParse("100"),
				api.ResourceMemory:                 resource.MustParse("10000"),
				api.ResourcePods:                   resource.MustParse("10"),
				api.ResourceServices:               resource.MustParse("10"),
				api.ResourceReplicationControllers: resource.MustParse("10"),
				api.ResourceQuotas:                 resource.MustParse("10"),
			},
		},
	}
	c := &testClient{
		Request: testRequest{
			Method: "POST",
			Path:   buildResourcePath(ns, "/resourceQuotas"),
			Query:  buildQueryValues(ns, nil),
			Body:   resourceQuota,
		},
		Response: Response{StatusCode: 200, Body: resourceQuota},
	}

	response, err := c.Setup().ResourceQuotas(ns).Create(resourceQuota)
	c.Validate(t, response, err)
}

func TestResourceQuotaGet(t *testing.T) {
	ns := api.NamespaceDefault
	resourceQuota := &api.ResourceQuota{
		ObjectMeta: api.ObjectMeta{
			Name:      "abc",
			Namespace: "foo",
		},
		Spec: api.ResourceQuotaSpec{
			Hard: api.ResourceList{
				api.ResourceCPU:                    resource.MustParse("100"),
				api.ResourceMemory:                 resource.MustParse("10000"),
				api.ResourcePods:                   resource.MustParse("10"),
				api.ResourceServices:               resource.MustParse("10"),
				api.ResourceReplicationControllers: resource.MustParse("10"),
				api.ResourceQuotas:                 resource.MustParse("10"),
			},
		},
	}
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   buildResourcePath(ns, "/resourceQuotas/abc"),
			Query:  buildQueryValues(ns, nil),
			Body:   nil,
		},
		Response: Response{StatusCode: 200, Body: resourceQuota},
	}

	response, err := c.Setup().ResourceQuotas(ns).Get("abc")
	c.Validate(t, response, err)
}

func TestResourceQuotaList(t *testing.T) {
	ns := api.NamespaceDefault

	resourceQuotaList := &api.ResourceQuotaList{
		Items: []api.ResourceQuota{
			{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
			},
		},
	}
	c := &testClient{
		Request: testRequest{
			Method: "GET",
			Path:   buildResourcePath(ns, "/resourceQuotas"),
			Query:  buildQueryValues(ns, nil),
			Body:   nil,
		},
		Response: Response{StatusCode: 200, Body: resourceQuotaList},
	}
	response, err := c.Setup().ResourceQuotas(ns).List(labels.Everything())
	c.Validate(t, response, err)
}

func TestResourceQuotaUpdate(t *testing.T) {
	ns := api.NamespaceDefault
	resourceQuota := &api.ResourceQuota{
		ObjectMeta: api.ObjectMeta{
			Name:            "abc",
			Namespace:       "foo",
			ResourceVersion: "1",
		},
		Spec: api.ResourceQuotaSpec{
			Hard: api.ResourceList{
				api.ResourceCPU:                    resource.MustParse("100"),
				api.ResourceMemory:                 resource.MustParse("10000"),
				api.ResourcePods:                   resource.MustParse("10"),
				api.ResourceServices:               resource.MustParse("10"),
				api.ResourceReplicationControllers: resource.MustParse("10"),
				api.ResourceQuotas:                 resource.MustParse("10"),
			},
		},
	}
	c := &testClient{
		Request:  testRequest{Method: "PUT", Path: buildResourcePath(ns, "/resourceQuotas/abc"), Query: buildQueryValues(ns, nil)},
		Response: Response{StatusCode: 200, Body: resourceQuota},
	}
	response, err := c.Setup().ResourceQuotas(ns).Update(resourceQuota)
	c.Validate(t, response, err)
}

func TestInvalidResourceQuotaUpdate(t *testing.T) {
	ns := api.NamespaceDefault
	resourceQuota := &api.ResourceQuota{
		ObjectMeta: api.ObjectMeta{
			Name:      "abc",
			Namespace: "foo",
		},
		Spec: api.ResourceQuotaSpec{
			Hard: api.ResourceList{
				api.ResourceCPU:                    resource.MustParse("100"),
				api.ResourceMemory:                 resource.MustParse("10000"),
				api.ResourcePods:                   resource.MustParse("10"),
				api.ResourceServices:               resource.MustParse("10"),
				api.ResourceReplicationControllers: resource.MustParse("10"),
				api.ResourceQuotas:                 resource.MustParse("10"),
			},
		},
	}
	c := &testClient{
		Request:  testRequest{Method: "PUT", Path: buildResourcePath(ns, "/resourceQuotas/abc"), Query: buildQueryValues(ns, nil)},
		Response: Response{StatusCode: 200, Body: resourceQuota},
	}
	_, err := c.Setup().ResourceQuotas(ns).Update(resourceQuota)
	if err == nil {
		t.Errorf("Expected an error due to missing ResourceVersion")
	}
}

func TestResourceQuotaDelete(t *testing.T) {
	ns := api.NamespaceDefault
	c := &testClient{
		Request:  testRequest{Method: "DELETE", Path: buildResourcePath(ns, "/resourceQuotas/foo"), Query: buildQueryValues(ns, nil)},
		Response: Response{StatusCode: 200},
	}
	err := c.Setup().ResourceQuotas(ns).Delete("foo")
	c.Validate(t, nil, err)
}
