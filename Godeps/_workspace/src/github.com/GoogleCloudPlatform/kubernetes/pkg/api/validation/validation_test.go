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

package validation

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/kubernetes/pkg/api"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/resource"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/capabilities"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/registry/registrytest"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/util"
	utilerrors "github.com/GoogleCloudPlatform/kubernetes/pkg/util/errors"
)

func expectPrefix(t *testing.T, prefix string, errs errors.ValidationErrorList) {
	for i := range errs {
		if f, p := errs[i].(*errors.ValidationError).Field, prefix; !strings.HasPrefix(f, p) {
			t.Errorf("expected prefix '%s' for field '%s' (%v)", p, f, errs[i])
		}
	}
}

// Ensure custom name functions are allowed
func TestValidateObjectMetaCustomName(t *testing.T) {
	errs := ValidateObjectMeta(&api.ObjectMeta{Name: "test", GenerateName: "foo"}, false, func(s string, prefix bool) (bool, string) {
		if s == "test" {
			return true, ""
		}
		return false, "name-gen"
	})
	if len(errs) != 1 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if !strings.Contains(errs[0].Error(), "name-gen") {
		t.Errorf("unexpected error message: %v", errs)
	}
}

// Ensure trailing slash is allowed in generate name
func TestValidateObjectMetaTrimsTrailingSlash(t *testing.T) {
	errs := ValidateObjectMeta(&api.ObjectMeta{Name: "test", GenerateName: "foo-"}, false, nameIsDNSSubdomain)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateLabels(t *testing.T) {
	successCases := []map[string]string{
		{"simple": "bar"},
		{"now-with-dashes": "bar"},
		{"1-starts-with-num": "bar"},
		{"1234": "bar"},
		{"simple/simple": "bar"},
		{"now-with-dashes/simple": "bar"},
		{"now-with-dashes/now-with-dashes": "bar"},
		{"now.with.dots/simple": "bar"},
		{"now-with.dashes-and.dots/simple": "bar"},
		{"1-num.2-num/3-num": "bar"},
		{"1234/5678": "bar"},
		{"1.2.3.4/5678": "bar"},
	}
	for i := range successCases {
		errs := ValidateLabels(successCases[i], "field")
		if len(errs) != 0 {
			t.Errorf("case[%d] expected success, got %#v", i, errs)
		}
	}

	errorCases := []map[string]string{
		{"NoUppercase123": "bar"},
		{"nospecialchars^=@": "bar"},
		{"cantendwithadash-": "bar"},
		{"only/one/slash": "bar"},
		{strings.Repeat("a", 254): "bar"},
	}
	for i := range errorCases {
		errs := ValidateLabels(errorCases[i], "field")
		if len(errs) != 1 {
			t.Errorf("case[%d] expected failure", i)
		} else {
			detail := errs[0].(*errors.ValidationError).Detail
			if detail != qualifiedNameErrorMsg {
				t.Errorf("error detail %s should be equal %s", detail, qualifiedNameErrorMsg)
			}
		}
	}
}

func TestValidateAnnotations(t *testing.T) {
	successCases := []map[string]string{
		{"simple": "bar"},
		{"now-with-dashes": "bar"},
		{"1-starts-with-num": "bar"},
		{"1234": "bar"},
		{"simple/simple": "bar"},
		{"now-with-dashes/simple": "bar"},
		{"now-with-dashes/now-with-dashes": "bar"},
		{"now.with.dots/simple": "bar"},
		{"now-with.dashes-and.dots/simple": "bar"},
		{"1-num.2-num/3-num": "bar"},
		{"1234/5678": "bar"},
		{"1.2.3.4/5678": "bar"},
		{"UpperCase123": "bar"},
	}
	for i := range successCases {
		errs := ValidateAnnotations(successCases[i], "field")
		if len(errs) != 0 {
			t.Errorf("case[%d] expected success, got %#v", i, errs)
		}
	}

	errorCases := []map[string]string{
		{"nospecialchars^=@": "bar"},
		{"cantendwithadash-": "bar"},
		{"only/one/slash": "bar"},
		{strings.Repeat("a", 254): "bar"},
	}
	for i := range errorCases {
		errs := ValidateAnnotations(errorCases[i], "field")
		if len(errs) != 1 {
			t.Errorf("case[%d] expected failure", i)
		} else {
			detail := errs[0].(*errors.ValidationError).Detail
			if detail != qualifiedNameErrorMsg {
				t.Errorf("error detail %s should be equal %s", detail, qualifiedNameErrorMsg)
			}
		}
	}
}

func TestValidateVolumes(t *testing.T) {
	successCase := []api.Volume{
		{Name: "abc", Source: api.VolumeSource{HostPath: &api.HostPathVolumeSource{"/mnt/path1"}}},
		{Name: "123", Source: api.VolumeSource{HostPath: &api.HostPathVolumeSource{"/mnt/path2"}}},
		{Name: "abc-123", Source: api.VolumeSource{HostPath: &api.HostPathVolumeSource{"/mnt/path3"}}},
		{Name: "empty", Source: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}},
		{Name: "gcepd", Source: api.VolumeSource{GCEPersistentDisk: &api.GCEPersistentDiskVolumeSource{"my-PD", "ext4", 1, false}}},
		{Name: "gitrepo", Source: api.VolumeSource{GitRepo: &api.GitRepoVolumeSource{"my-repo", "hashstring"}}},
		{Name: "secret", Source: api.VolumeSource{Secret: &api.SecretVolumeSource{api.ObjectReference{Namespace: api.NamespaceDefault, Name: "my-secret", Kind: "Secret"}}}},
	}
	names, errs := validateVolumes(successCase)
	if len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}
	if len(names) != len(successCase) || !names.HasAll("abc", "123", "abc-123", "empty", "gcepd", "gitrepo", "secret") {
		t.Errorf("wrong names result: %v", names)
	}
	emptyVS := api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}
	errorCases := map[string]struct {
		V []api.Volume
		T errors.ValidationErrorType
		F string
	}{
		"zero-length name":     {[]api.Volume{{Name: "", Source: emptyVS}}, errors.ValidationErrorTypeRequired, "[0].name"},
		"name > 63 characters": {[]api.Volume{{Name: strings.Repeat("a", 64), Source: emptyVS}}, errors.ValidationErrorTypeInvalid, "[0].name"},
		"name not a DNS label": {[]api.Volume{{Name: "a.b.c", Source: emptyVS}}, errors.ValidationErrorTypeInvalid, "[0].name"},
		"name not unique":      {[]api.Volume{{Name: "abc", Source: emptyVS}, {Name: "abc", Source: emptyVS}}, errors.ValidationErrorTypeDuplicate, "[1].name"},
	}
	for k, v := range errorCases {
		_, errs := validateVolumes(v.V)
		if len(errs) == 0 {
			t.Errorf("expected failure %s for %v", k, v.V)
			continue
		}
		for i := range errs {
			if errs[i].(*errors.ValidationError).Type != v.T {
				t.Errorf("%s: expected errors to have type %s: %v", k, v.T, errs[i])
			}
			if errs[i].(*errors.ValidationError).Field != v.F {
				t.Errorf("%s: expected errors to have field %s: %v", k, v.F, errs[i])
			}
			detail := errs[i].(*errors.ValidationError).Detail
			if detail != "" && detail != dnsLabelErrorMsg {
				t.Errorf("%s: expected error detail either empty or %s, got %s", k, dnsLabelErrorMsg, detail)
			}
		}
	}
}

func TestValidatePorts(t *testing.T) {
	successCase := []api.Port{
		{Name: "abc", ContainerPort: 80, HostPort: 80, Protocol: "TCP"},
		{Name: "easy", ContainerPort: 82, Protocol: "TCP"},
		{Name: "as", ContainerPort: 83, Protocol: "UDP"},
		{Name: "do-re-me", ContainerPort: 84, Protocol: "UDP"},
		{Name: "baby-you-and-me", ContainerPort: 82, Protocol: "tcp"},
		{ContainerPort: 85, Protocol: "TCP"},
	}
	if errs := validatePorts(successCase); len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	nonCanonicalCase := []api.Port{
		{ContainerPort: 80, Protocol: "TCP"},
	}
	if errs := validatePorts(nonCanonicalCase); len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string]struct {
		P []api.Port
		T errors.ValidationErrorType
		F string
		D string
	}{
		"name > 63 characters": {[]api.Port{{Name: strings.Repeat("a", 64), ContainerPort: 80, Protocol: "TCP"}}, errors.ValidationErrorTypeInvalid, "[0].name", dnsLabelErrorMsg},
		"name not a DNS label": {[]api.Port{{Name: "a.b.c", ContainerPort: 80, Protocol: "TCP"}}, errors.ValidationErrorTypeInvalid, "[0].name", dnsLabelErrorMsg},
		"name not unique": {[]api.Port{
			{Name: "abc", ContainerPort: 80, Protocol: "TCP"},
			{Name: "abc", ContainerPort: 81, Protocol: "TCP"},
		}, errors.ValidationErrorTypeDuplicate, "[1].name", ""},
		"zero container port":    {[]api.Port{{ContainerPort: 0, Protocol: "TCP"}}, errors.ValidationErrorTypeInvalid, "[0].containerPort", portRangeErrorMsg},
		"invalid container port": {[]api.Port{{ContainerPort: 65536, Protocol: "TCP"}}, errors.ValidationErrorTypeInvalid, "[0].containerPort", portRangeErrorMsg},
		"invalid host port":      {[]api.Port{{ContainerPort: 80, HostPort: 65536, Protocol: "TCP"}}, errors.ValidationErrorTypeInvalid, "[0].hostPort", portRangeErrorMsg},
		"invalid protocol":       {[]api.Port{{ContainerPort: 80, Protocol: "ICMP"}}, errors.ValidationErrorTypeNotSupported, "[0].protocol", ""},
		"protocol required":      {[]api.Port{{Name: "abc", ContainerPort: 80}}, errors.ValidationErrorTypeRequired, "[0].protocol", ""},
	}
	for k, v := range errorCases {
		errs := validatePorts(v.P)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
		for i := range errs {
			if errs[i].(*errors.ValidationError).Type != v.T {
				t.Errorf("%s: expected errors to have type %s: %v", k, v.T, errs[i])
			}
			if errs[i].(*errors.ValidationError).Field != v.F {
				t.Errorf("%s: expected errors to have field %s: %v", k, v.F, errs[i])
			}
			detail := errs[i].(*errors.ValidationError).Detail
			if detail != v.D {
				t.Errorf("%s: expected error detail either empty or %s, got %s", k, v.D, detail)
			}
		}
	}
}

func TestValidateEnv(t *testing.T) {
	successCase := []api.EnvVar{
		{Name: "abc", Value: "value"},
		{Name: "ABC", Value: "value"},
		{Name: "AbC_123", Value: "value"},
		{Name: "abc", Value: ""},
	}
	if errs := validateEnv(successCase); len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string][]api.EnvVar{
		"zero-length name":        {{Name: ""}},
		"name not a C identifier": {{Name: "a.b.c"}},
	}
	for k, v := range errorCases {
		if errs := validateEnv(v); len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		} else {
			for i := range errs {
				detail := errs[i].(*errors.ValidationError).Detail
				if detail != "" && detail != cIdentifierErrorMsg {
					t.Errorf("%s: expected error detail either empty or %s, got %s", k, cIdentifierErrorMsg, detail)
				}
			}
		}
	}
}

func TestValidateVolumeMounts(t *testing.T) {
	volumes := util.NewStringSet("abc", "123", "abc-123")

	successCase := []api.VolumeMount{
		{Name: "abc", MountPath: "/foo"},
		{Name: "123", MountPath: "/foo"},
		{Name: "abc-123", MountPath: "/bar"},
	}
	if errs := validateVolumeMounts(successCase, volumes); len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string][]api.VolumeMount{
		"empty name":      {{Name: "", MountPath: "/foo"}},
		"name not found":  {{Name: "", MountPath: "/foo"}},
		"empty mountpath": {{Name: "abc", MountPath: ""}},
	}
	for k, v := range errorCases {
		if errs := validateVolumeMounts(v, volumes); len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
	}
}

func TestValidateProbe(t *testing.T) {
	handler := api.Handler{Exec: &api.ExecAction{Command: []string{"echo"}}}
	successCases := []*api.Probe{
		nil,
		{TimeoutSeconds: 10, InitialDelaySeconds: 0, Handler: handler},
		{TimeoutSeconds: 0, InitialDelaySeconds: 10, Handler: handler},
	}
	for _, p := range successCases {
		if errs := validateProbe(p); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := []*api.Probe{
		{TimeoutSeconds: 10, InitialDelaySeconds: 10},
		{TimeoutSeconds: 10, InitialDelaySeconds: -10, Handler: handler},
		{TimeoutSeconds: -10, InitialDelaySeconds: 10, Handler: handler},
		{TimeoutSeconds: -10, InitialDelaySeconds: -10, Handler: handler},
	}
	for _, p := range errorCases {
		if errs := validateProbe(p); len(errs) == 0 {
			t.Errorf("expected failure for %v", p)
		}
	}
}

func TestValidatePullPolicy(t *testing.T) {
	type T struct {
		Container      api.Container
		ExpectedPolicy api.PullPolicy
	}
	testCases := map[string]T{
		"NotPresent1": {
			api.Container{Name: "abc", Image: "image:latest", ImagePullPolicy: "IfNotPresent"},
			api.PullIfNotPresent,
		},
		"NotPresent2": {
			api.Container{Name: "abc1", Image: "image", ImagePullPolicy: "IfNotPresent"},
			api.PullIfNotPresent,
		},
		"Always1": {
			api.Container{Name: "123", Image: "image:latest", ImagePullPolicy: "Always"},
			api.PullAlways,
		},
		"Always2": {
			api.Container{Name: "1234", Image: "image", ImagePullPolicy: "Always"},
			api.PullAlways,
		},
		"Never1": {
			api.Container{Name: "abc-123", Image: "image:latest", ImagePullPolicy: "Never"},
			api.PullNever,
		},
		"Never2": {
			api.Container{Name: "abc-1234", Image: "image", ImagePullPolicy: "Never"},
			api.PullNever,
		},
	}
	for k, v := range testCases {
		ctr := &v.Container
		errs := validatePullPolicy(ctr)
		if len(errs) != 0 {
			t.Errorf("case[%s] expected success, got %#v", k, errs)
		}
		if ctr.ImagePullPolicy != v.ExpectedPolicy {
			t.Errorf("case[%s] expected policy %v, got %v", k, v.ExpectedPolicy, ctr.ImagePullPolicy)
		}
	}

}

func getResourceLimits(cpu, memory string) api.ResourceList {
	res := api.ResourceList{}
	res[api.ResourceCPU] = resource.MustParse(cpu)
	res[api.ResourceMemory] = resource.MustParse(memory)
	return res
}

func TestValidateContainers(t *testing.T) {
	volumes := util.StringSet{}
	capabilities.SetForTests(capabilities.Capabilities{
		AllowPrivileged: true,
	})

	successCase := []api.Container{
		{Name: "abc", Image: "image", ImagePullPolicy: "IfNotPresent"},
		{Name: "123", Image: "image", ImagePullPolicy: "IfNotPresent"},
		{Name: "abc-123", Image: "image", ImagePullPolicy: "IfNotPresent"},
		{
			Name:  "life-123",
			Image: "image",
			Lifecycle: &api.Lifecycle{
				PreStop: &api.Handler{
					Exec: &api.ExecAction{Command: []string{"ls", "-l"}},
				},
			},
			ImagePullPolicy: "IfNotPresent",
		},
		{
			Name:  "resources-test",
			Image: "image",
			Resources: api.ResourceRequirements{
				Limits: api.ResourceList{
					api.ResourceName(api.ResourceCPU):    resource.MustParse("10"),
					api.ResourceName(api.ResourceMemory): resource.MustParse("10G"),
					api.ResourceName("my.org/resource"):  resource.MustParse("10m"),
				},
			},
			ImagePullPolicy: "IfNotPresent",
		},
		{Name: "abc-1234", Image: "image", Privileged: true, ImagePullPolicy: "IfNotPresent"},
	}
	if errs := validateContainers(successCase, volumes); len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	capabilities.SetForTests(capabilities.Capabilities{
		AllowPrivileged: false,
	})
	errorCases := map[string][]api.Container{
		"zero-length name":     {{Name: "", Image: "image", ImagePullPolicy: "IfNotPresent"}},
		"name > 63 characters": {{Name: strings.Repeat("a", 64), Image: "image", ImagePullPolicy: "IfNotPresent"}},
		"name not a DNS label": {{Name: "a.b.c", Image: "image", ImagePullPolicy: "IfNotPresent"}},
		"name not unique": {
			{Name: "abc", Image: "image", ImagePullPolicy: "IfNotPresent"},
			{Name: "abc", Image: "image", ImagePullPolicy: "IfNotPresent"},
		},
		"zero-length image": {{Name: "abc", Image: "", ImagePullPolicy: "IfNotPresent"}},
		"host port not unique": {
			{Name: "abc", Image: "image", Ports: []api.Port{{ContainerPort: 80, HostPort: 80, Protocol: "TCP"}},
				ImagePullPolicy: "IfNotPresent"},
			{Name: "def", Image: "image", Ports: []api.Port{{ContainerPort: 81, HostPort: 80, Protocol: "TCP"}},
				ImagePullPolicy: "IfNotPresent"},
		},
		"invalid env var name": {
			{Name: "abc", Image: "image", Env: []api.EnvVar{{Name: "ev.1"}}, ImagePullPolicy: "IfNotPresent"},
		},
		"unknown volume name": {
			{Name: "abc", Image: "image", VolumeMounts: []api.VolumeMount{{Name: "anything", MountPath: "/foo"}},
				ImagePullPolicy: "IfNotPresent"},
		},
		"invalid lifecycle, no exec command.": {
			{
				Name:  "life-123",
				Image: "image",
				Lifecycle: &api.Lifecycle{
					PreStop: &api.Handler{
						Exec: &api.ExecAction{},
					},
				},
				ImagePullPolicy: "IfNotPresent",
			},
		},
		"invalid lifecycle, no http path.": {
			{
				Name:  "life-123",
				Image: "image",
				Lifecycle: &api.Lifecycle{
					PreStop: &api.Handler{
						HTTPGet: &api.HTTPGetAction{},
					},
				},
				ImagePullPolicy: "IfNotPresent",
			},
		},
		"invalid lifecycle, no action.": {
			{
				Name:  "life-123",
				Image: "image",
				Lifecycle: &api.Lifecycle{
					PreStop: &api.Handler{},
				},
				ImagePullPolicy: "IfNotPresent",
			},
		},
		"privilege disabled": {
			{Name: "abc", Image: "image", Privileged: true},
		},
		"invalid compute resource": {
			{
				Name:  "abc-123",
				Image: "image",
				Resources: api.ResourceRequirements{
					Limits: api.ResourceList{
						"disk": resource.MustParse("10G"),
					},
				},
				ImagePullPolicy: "IfNotPresent",
			},
		},
		"Resource CPU invalid": {
			{
				Name:  "abc-123",
				Image: "image",
				Resources: api.ResourceRequirements{
					Limits: getResourceLimits("-10", "0"),
				},
				ImagePullPolicy: "IfNotPresent",
			},
		},
		"Resource Memory invalid": {
			{
				Name:  "abc-123",
				Image: "image",
				Resources: api.ResourceRequirements{
					Limits: getResourceLimits("0", "-10"),
				},
				ImagePullPolicy: "IfNotPresent",
			},
		},
	}
	for k, v := range errorCases {
		if errs := validateContainers(v, volumes); len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
	}
}

func TestValidateRestartPolicy(t *testing.T) {
	successCases := []api.RestartPolicy{
		{Always: &api.RestartPolicyAlways{}},
		{OnFailure: &api.RestartPolicyOnFailure{}},
		{Never: &api.RestartPolicyNever{}},
	}
	for _, policy := range successCases {
		if errs := validateRestartPolicy(&policy); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := []api.RestartPolicy{
		{},
		{Always: &api.RestartPolicyAlways{}, Never: &api.RestartPolicyNever{}},
		{Never: &api.RestartPolicyNever{}, OnFailure: &api.RestartPolicyOnFailure{}},
	}
	for k, policy := range errorCases {
		if errs := validateRestartPolicy(&policy); len(errs) == 0 {
			t.Errorf("expected failure for %d", k)
		}
	}
}

func TestValidateDNSPolicy(t *testing.T) {
	successCases := []api.DNSPolicy{api.DNSClusterFirst, api.DNSDefault, api.DNSPolicy(api.DNSClusterFirst)}
	for _, policy := range successCases {
		if errs := validateDNSPolicy(&policy); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := []api.DNSPolicy{api.DNSPolicy("invalid")}
	for _, policy := range errorCases {
		if errs := validateDNSPolicy(&policy); len(errs) == 0 {
			t.Errorf("expected failure for %v", policy)
		}
	}
}

func TestValidateManifest(t *testing.T) {
	successCases := []api.ContainerManifest{
		{Version: "v1beta1", ID: "abc", RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy: api.DNSClusterFirst},
		{Version: "v1beta2", ID: "123", RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy: api.DNSClusterFirst},
		{Version: "V1BETA1", ID: "abc.123.do-re-mi",
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}}, DNSPolicy: api.DNSClusterFirst},
		{
			Version: "v1beta1",
			ID:      "abc",
			Volumes: []api.Volume{{Name: "vol1", Source: api.VolumeSource{HostPath: &api.HostPathVolumeSource{"/mnt/vol1"}}},
				{Name: "vol2", Source: api.VolumeSource{HostPath: &api.HostPathVolumeSource{"/mnt/vol2"}}}},
			Containers: []api.Container{
				{
					Name:       "abc",
					Image:      "image",
					Command:    []string{"foo", "bar"},
					WorkingDir: "/tmp",
					Resources: api.ResourceRequirements{
						Limits: api.ResourceList{
							"cpu":    resource.MustParse("1"),
							"memory": resource.MustParse("1"),
						},
					},
					Ports: []api.Port{
						{Name: "p1", ContainerPort: 80, HostPort: 8080, Protocol: "TCP"},
						{Name: "p2", ContainerPort: 81, Protocol: "TCP"},
						{ContainerPort: 82, Protocol: "TCP"},
					},
					Env: []api.EnvVar{
						{Name: "ev1", Value: "val1"},
						{Name: "ev2", Value: "val2"},
						{Name: "EV3", Value: "val3"},
					},
					VolumeMounts: []api.VolumeMount{
						{Name: "vol1", MountPath: "/foo"},
						{Name: "vol1", MountPath: "/bar"},
					},
					ImagePullPolicy: "IfNotPresent",
				},
			},
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy:     api.DNSClusterFirst,
		},
	}
	for _, manifest := range successCases {
		if errs := ValidateManifest(&manifest); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]api.ContainerManifest{
		"empty version": {Version: "", ID: "abc",
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy:     api.DNSClusterFirst},
		"invalid version": {Version: "bogus", ID: "abc",
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy:     api.DNSClusterFirst},
		"invalid volume name": {
			Version:       "v1beta1",
			ID:            "abc",
			Volumes:       []api.Volume{{Name: "vol.1", Source: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}}},
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy:     api.DNSClusterFirst,
		},
		"invalid container name": {
			Version: "v1beta1",
			ID:      "abc",
			Containers: []api.Container{{Name: "ctr.1", Image: "image", ImagePullPolicy: "IfNotPresent",
				TerminationMessagePath: "/foo/bar"}},
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy:     api.DNSClusterFirst,
		},
	}
	for k, v := range errorCases {
		if errs := ValidateManifest(&v); len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
	}
}

func TestValidatePodSpec(t *testing.T) {
	successCases := []api.PodSpec{
		{ // Populate basic fields, leave defaults for most.
			Volumes:       []api.Volume{{Name: "vol", Source: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}}},
			Containers:    []api.Container{{Name: "ctr", Image: "image", ImagePullPolicy: "IfNotPresent"}},
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy:     api.DNSClusterFirst,
		},
		{ // Populate all fields.
			Volumes: []api.Volume{
				{Name: "vol", Source: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}},
			},
			Containers:    []api.Container{{Name: "ctr", Image: "image", ImagePullPolicy: "IfNotPresent"}},
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			NodeSelector: map[string]string{
				"key": "value",
			},
			Host:      "foobar",
			DNSPolicy: api.DNSClusterFirst,
		},
	}
	for i := range successCases {
		if errs := ValidatePodSpec(&successCases[i]); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	failureCases := map[string]api.PodSpec{
		"bad volume": {
			Volumes:       []api.Volume{{}},
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy:     api.DNSClusterFirst,
		},
		"bad container": {
			Containers:    []api.Container{{}},
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
			DNSPolicy:     api.DNSClusterFirst,
		},
		"bad DNS policy": {
			DNSPolicy:     api.DNSPolicy("invalid"),
			RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
		},
		"bad restart policy": {
			RestartPolicy: api.RestartPolicy{},
			DNSPolicy:     api.DNSClusterFirst,
		},
	}
	for k, v := range failureCases {
		if errs := ValidatePodSpec(&v); len(errs) == 0 {
			t.Errorf("expected failure for %q", k)
		}
	}
}

func TestValidatePod(t *testing.T) {
	successCases := []api.Pod{
		{ // Basic fields.
			ObjectMeta: api.ObjectMeta{Name: "123", Namespace: "ns"},
			Spec: api.PodSpec{
				Volumes:       []api.Volume{{Name: "vol", Source: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}}},
				Containers:    []api.Container{{Name: "ctr", Image: "image", ImagePullPolicy: "IfNotPresent"}},
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		{ // Just about everything.
			ObjectMeta: api.ObjectMeta{Name: "abc.123.do-re-mi", Namespace: "ns"},
			Spec: api.PodSpec{
				Volumes: []api.Volume{
					{Name: "vol", Source: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}},
				},
				Containers:    []api.Container{{Name: "ctr", Image: "image", ImagePullPolicy: "IfNotPresent"}},
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
				NodeSelector: map[string]string{
					"key": "value",
				},
				Host: "foobar",
			},
		},
	}
	for _, pod := range successCases {
		if errs := ValidatePod(&pod); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]api.Pod{
		"bad name": {
			ObjectMeta: api.ObjectMeta{Name: "", Namespace: "ns"},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		"bad namespace": {
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: ""},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		"bad spec": {
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: "ns"},
			Spec: api.PodSpec{
				Containers: []api.Container{{}},
			},
		},
		"bad label": {
			ObjectMeta: api.ObjectMeta{
				Name:      "abc",
				Namespace: "ns",
				Labels: map[string]string{
					"NoUppercaseOrSpecialCharsLike=Equals": "bar",
				},
			},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		"bad annotation": {
			ObjectMeta: api.ObjectMeta{
				Name:      "abc",
				Namespace: "ns",
				Annotations: map[string]string{
					"NoUppercaseOrSpecialCharsLike=Equals": "bar",
				},
			},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
	}
	for k, v := range errorCases {
		if errs := ValidatePod(&v); len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
	}
}

func TestValidatePodUpdate(t *testing.T) {
	tests := []struct {
		a       api.Pod
		b       api.Pod
		isValid bool
		test    string
	}{
		{api.Pod{}, api.Pod{}, true, "nothing"},
		{
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
			},
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "bar"},
			},
			false,
			"ids",
		},
		{
			api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						"bar": "foo",
					},
				},
			},
			true,
			"labels",
		},
		{
			api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
			},
			api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
					Annotations: map[string]string{
						"bar": "foo",
					},
				},
			},
			true,
			"annotations",
		},
		{
			api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Image: "foo:V1",
						},
					},
				},
			},
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Image: "foo:V2",
						},
						{
							Image: "bar:V2",
						},
					},
				},
			},
			false,
			"more containers",
		},
		{
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Image: "foo:V1",
						},
					},
				},
			},
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Image: "foo:V2",
						},
					},
				},
			},
			true,
			"image change",
		},
		{
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Image: "foo:V1",
							Resources: api.ResourceRequirements{
								Limits: getResourceLimits("100m", "0"),
							},
						},
					},
				},
			},
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Image: "foo:V2",
							Resources: api.ResourceRequirements{
								Limits: getResourceLimits("1000m", "0"),
							},
						},
					},
				},
			},
			false,
			"cpu change",
		},
		{
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Image: "foo:V1",
							Ports: []api.Port{
								{HostPort: 8080, ContainerPort: 80},
							},
						},
					},
				},
			},
			api.Pod{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Image: "foo:V2",
							Ports: []api.Port{
								{HostPort: 8000, ContainerPort: 80},
							},
						},
					},
				},
			},
			false,
			"port change",
		},
		{
			api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						"foo": "bar",
					},
				},
			},
			api.Pod{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
					Labels: map[string]string{
						"Bar": "foo",
					},
				},
			},
			true,
			"bad label change",
		},
	}

	for _, test := range tests {
		errs := ValidatePodUpdate(&test.a, &test.b)
		if test.isValid {
			if len(errs) != 0 {
				t.Errorf("unexpected invalid: %s %v, %v", test.test, test.a, test.b)
			}
		} else {
			if len(errs) == 0 {
				t.Errorf("unexpected valid: %s %v, %v", test.test, test.a, test.b)
			}
		}
	}
}

func TestValidateBoundPods(t *testing.T) {
	successCases := []api.BoundPod{
		{ // Mostly empty.
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: "ns"},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		{ // Basic fields.
			ObjectMeta: api.ObjectMeta{Name: "123", Namespace: "ns"},
			Spec: api.PodSpec{
				Volumes:       []api.Volume{{Name: "vol", Source: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}}},
				Containers:    []api.Container{{Name: "ctr", Image: "image", ImagePullPolicy: "IfNotPresent"}},
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		{ // Just about everything.
			ObjectMeta: api.ObjectMeta{Name: "abc.123.do-re-mi", Namespace: "ns"},
			Spec: api.PodSpec{
				Volumes: []api.Volume{
					{Name: "vol", Source: api.VolumeSource{EmptyDir: &api.EmptyDirVolumeSource{}}},
				},
				Containers:    []api.Container{{Name: "ctr", Image: "image", ImagePullPolicy: "IfNotPresent"}},
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
				NodeSelector: map[string]string{
					"key": "value",
				},
				Host: "foobar",
			},
		},
	}
	for _, pod := range successCases {
		if errs := ValidateBoundPod(&pod); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]api.Pod{
		"zero-length name": {
			ObjectMeta: api.ObjectMeta{Name: "", Namespace: "ns"},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		"bad namespace": {
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: ""},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		"bad spec": {
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: "ns"},
			Spec: api.PodSpec{
				Containers:    []api.Container{{Name: "name", ImagePullPolicy: "IfNotPresent"}},
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		"name > 253 characters": {
			ObjectMeta: api.ObjectMeta{Name: strings.Repeat("a", 254), Namespace: "ns"},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		"name not a DNS subdomain": {
			ObjectMeta: api.ObjectMeta{Name: "a..b.c", Namespace: "ns"},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
		"name with underscore": {
			ObjectMeta: api.ObjectMeta{Name: "a_b_c", Namespace: "ns"},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
	}
	for k, v := range errorCases {
		if errs := ValidatePod(&v); len(errs) != 1 {
			t.Errorf("expected one failure for %s; got %d: %s", k, len(errs), errs)
		}
	}
}

func TestValidateService(t *testing.T) {
	testCases := []struct {
		name     string
		svc      api.Service
		existing api.ServiceList
		numErrs  int
	}{
		{
			name: "missing id",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			// Should fail because the ID is missing.
			numErrs: 1,
		},
		{
			name: "missing protocol",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					SessionAffinity: "None",
				},
			},
			// Should fail because protocol is missing.
			numErrs: 1,
		},
		{
			name: "missing session affinity",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:     8675,
					Selector: map[string]string{"foo": "bar"},
					Protocol: "TCP",
				},
			},
			// Should fail because the session affinity is missing.
			numErrs: 1,
		},
		{
			name: "missing namespace",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "foo"},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			// Should fail because the Namespace is missing.
			numErrs: 1,
		},
		{
			name: "invalid id",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "-123abc", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			// Should fail because the ID is invalid.
			numErrs: 1,
		},
		{
			name: "invalid generate.base",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:         "valid",
					GenerateName: "-123abc",
					Namespace:    api.NamespaceDefault,
				},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			// Should fail because the Base value for generation is invalid
			numErrs: 1,
		},
		{
			name: "invalid generateName",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:         "valid",
					GenerateName: "abc1234567abc1234567abc1234567abc1234567abc1234567abc1234567",
					Namespace:    api.NamespaceDefault,
				},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			// Should fail because the generate name type is invalid.
			numErrs: 1,
		},
		{
			name: "missing port",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			// Should fail because the port number is missing/invalid.
			numErrs: 1,
		},
		{
			name: "invalid port",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            66536,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			// Should fail because the port number is invalid.
			numErrs: 1,
		},
		{
			name: "invalid protocol",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "INVALID",
					SessionAffinity: "None",
				},
			},
			// Should fail because the protocol is invalid.
			numErrs: 1,
		},
		{
			name: "missing selector",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "foo", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            8675,
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			// Should be ok because the selector is missing.
			numErrs: 0,
		},
		{
			name: "valid 1",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			numErrs: 0,
		},
		{
			name: "valid 2",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "UDP",
					SessionAffinity: "None",
				},
			},
			numErrs: 0,
		},
		{
			name: "valid 3",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "UDP",
					SessionAffinity: "None",
				},
			},
			numErrs: 0,
		},
		{
			name: "external port in use",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port: 80,
					CreateExternalLoadBalancer: true,
					Selector:                   map[string]string{"foo": "bar"},
					Protocol:                   "TCP",
					SessionAffinity:            "None",
				},
			},
			existing: api.ServiceList{
				Items: []api.Service{
					{
						ObjectMeta: api.ObjectMeta{Name: "def123", Namespace: api.NamespaceDefault},
						Spec:       api.ServiceSpec{Port: 80, CreateExternalLoadBalancer: true, Protocol: "TCP"},
					},
				},
			},
			numErrs: 0,
		},
		{
			name: "same port in use, but not external",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port: 80,
					CreateExternalLoadBalancer: true,
					Selector:                   map[string]string{"foo": "bar"},
					Protocol:                   "TCP",
					SessionAffinity:            "None",
				},
			},
			existing: api.ServiceList{
				Items: []api.Service{
					{
						ObjectMeta: api.ObjectMeta{Name: "def123", Namespace: api.NamespaceDefault},
						Spec:       api.ServiceSpec{Port: 80, Protocol: "TCP"},
					},
				},
			},
			numErrs: 0,
		},
		{
			name: "same port in use, but not external on input",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            80,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			existing: api.ServiceList{
				Items: []api.Service{
					{
						ObjectMeta: api.ObjectMeta{Name: "def123", Namespace: api.NamespaceDefault},
						Spec:       api.ServiceSpec{Port: 80, CreateExternalLoadBalancer: true, Protocol: "TCP"},
					},
				},
			},
			numErrs: 0,
		},
		{
			name: "same port in use, but neither external",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{Name: "abc123", Namespace: api.NamespaceDefault},
				Spec: api.ServiceSpec{
					Port:            80,
					Selector:        map[string]string{"foo": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			existing: api.ServiceList{
				Items: []api.Service{
					{
						ObjectMeta: api.ObjectMeta{Name: "def123", Namespace: api.NamespaceDefault},
						Spec:       api.ServiceSpec{Port: 80, Protocol: "TCP"},
					},
				},
			},
			numErrs: 0,
		},
		{
			name: "invalid label",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:      "abc123",
					Namespace: api.NamespaceDefault,
					Labels: map[string]string{
						"NoUppercaseOrSpecialCharsLike=Equals": "bar",
					},
				},
				Spec: api.ServiceSpec{
					Port:            8675,
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			numErrs: 1,
		},
		{
			name: "invalid annotation",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:      "abc123",
					Namespace: api.NamespaceDefault,
					Annotations: map[string]string{
						"NoUppercaseOrSpecialCharsLike=Equals": "bar",
					},
				},
				Spec: api.ServiceSpec{
					Port:            8675,
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			numErrs: 1,
		},
		{
			name: "invalid selector",
			svc: api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:      "abc123",
					Namespace: api.NamespaceDefault,
				},
				Spec: api.ServiceSpec{
					Port:            8675,
					Selector:        map[string]string{"foo": "bar", "NoUppercaseOrSpecialCharsLike=Equals": "bar"},
					Protocol:        "TCP",
					SessionAffinity: "None",
				},
			},
			numErrs: 1,
		},
	}

	for _, tc := range testCases {
		registry := registrytest.NewServiceRegistry()
		registry.List = tc.existing
		errs := ValidateService(&tc.svc)
		if len(errs) != tc.numErrs {
			t.Errorf("Unexpected error list for case %q: %v", tc.name, utilerrors.NewAggregate(errs))
		}
	}

	svc := api.Service{
		ObjectMeta: api.ObjectMeta{Name: "foo", Namespace: api.NamespaceDefault},
		Spec: api.ServiceSpec{
			Port:            8675,
			Selector:        map[string]string{"foo": "bar"},
			Protocol:        "TCP",
			SessionAffinity: "None",
		},
	}
	errs := ValidateService(&svc)
	if len(errs) != 0 {
		t.Errorf("Unexpected non-zero error list: %#v", errs)
		for i := range errs {
			t.Errorf("Found error: %s", errs[i].Error())
		}
	}
}

func TestValidateReplicationControllerUpdate(t *testing.T) {
	validSelector := map[string]string{"a": "b"}
	validPodTemplate := api.PodTemplate{
		Spec: api.PodTemplateSpec{
			ObjectMeta: api.ObjectMeta{
				Labels: validSelector,
			},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
	}
	readWriteVolumePodTemplate := api.PodTemplate{
		Spec: api.PodTemplateSpec{
			ObjectMeta: api.ObjectMeta{
				Labels: validSelector,
			},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
				Volumes:       []api.Volume{{Name: "gcepd", Source: api.VolumeSource{GCEPersistentDisk: &api.GCEPersistentDiskVolumeSource{"my-PD", "ext4", 1, false}}}},
			},
		},
	}
	invalidSelector := map[string]string{"NoUppercaseOrSpecialCharsLike=Equals": "b"}
	invalidPodTemplate := api.PodTemplate{
		Spec: api.PodTemplateSpec{
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
			ObjectMeta: api.ObjectMeta{
				Labels: invalidSelector,
			},
		},
	}
	type rcUpdateTest struct {
		old    api.ReplicationController
		update api.ReplicationController
	}
	successCases := []rcUpdateTest{
		{
			old: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Selector: validSelector,
					Template: &validPodTemplate.Spec,
				},
			},
			update: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Replicas: 3,
					Selector: validSelector,
					Template: &validPodTemplate.Spec,
				},
			},
		},
		{
			old: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Selector: validSelector,
					Template: &validPodTemplate.Spec,
				},
			},
			update: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Replicas: 1,
					Selector: validSelector,
					Template: &readWriteVolumePodTemplate.Spec,
				},
			},
		},
	}
	for _, successCase := range successCases {
		if errs := ValidateReplicationControllerUpdate(&successCase.old, &successCase.update); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}
	errorCases := map[string]rcUpdateTest{
		"more than one read/write": {
			old: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Selector: validSelector,
					Template: &validPodTemplate.Spec,
				},
			},
			update: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Replicas: 2,
					Selector: validSelector,
					Template: &readWriteVolumePodTemplate.Spec,
				},
			},
		},
		"invalid selector": {
			old: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Selector: validSelector,
					Template: &validPodTemplate.Spec,
				},
			},
			update: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Replicas: 2,
					Selector: invalidSelector,
					Template: &validPodTemplate.Spec,
				},
			},
		},
		"invalid pod": {
			old: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Selector: validSelector,
					Template: &validPodTemplate.Spec,
				},
			},
			update: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Replicas: 2,
					Selector: validSelector,
					Template: &invalidPodTemplate.Spec,
				},
			},
		},
		"negative replicas": {
			old: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Selector: validSelector,
					Template: &validPodTemplate.Spec,
				},
			},
			update: api.ReplicationController{
				ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
				Spec: api.ReplicationControllerSpec{
					Replicas: -1,
					Selector: validSelector,
					Template: &validPodTemplate.Spec,
				},
			},
		},
	}
	for testName, errorCase := range errorCases {
		if errs := ValidateReplicationControllerUpdate(&errorCase.old, &errorCase.update); len(errs) == 0 {
			t.Errorf("expected failure: %s", testName)
		}
	}

}

func TestValidateReplicationController(t *testing.T) {
	validSelector := map[string]string{"a": "b"}
	validPodTemplate := api.PodTemplate{
		Spec: api.PodTemplateSpec{
			ObjectMeta: api.ObjectMeta{
				Labels: validSelector,
			},
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
	}
	readWriteVolumePodTemplate := api.PodTemplate{
		Spec: api.PodTemplateSpec{
			ObjectMeta: api.ObjectMeta{
				Labels: validSelector,
			},
			Spec: api.PodSpec{
				Volumes:       []api.Volume{{Name: "gcepd", Source: api.VolumeSource{GCEPersistentDisk: &api.GCEPersistentDiskVolumeSource{"my-PD", "ext4", 1, false}}}},
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
		},
	}
	invalidSelector := map[string]string{"NoUppercaseOrSpecialCharsLike=Equals": "b"}
	invalidPodTemplate := api.PodTemplate{
		Spec: api.PodTemplateSpec{
			Spec: api.PodSpec{
				RestartPolicy: api.RestartPolicy{Always: &api.RestartPolicyAlways{}},
				DNSPolicy:     api.DNSClusterFirst,
			},
			ObjectMeta: api.ObjectMeta{
				Labels: invalidSelector,
			},
		},
	}
	successCases := []api.ReplicationController{
		{
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
				Template: &validPodTemplate.Spec,
			},
		},
		{
			ObjectMeta: api.ObjectMeta{Name: "abc-123", Namespace: api.NamespaceDefault},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
				Template: &validPodTemplate.Spec,
			},
		},
		{
			ObjectMeta: api.ObjectMeta{Name: "abc-123", Namespace: api.NamespaceDefault},
			Spec: api.ReplicationControllerSpec{
				Replicas: 1,
				Selector: validSelector,
				Template: &readWriteVolumePodTemplate.Spec,
			},
		},
	}
	for _, successCase := range successCases {
		if errs := ValidateReplicationController(&successCase); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]api.ReplicationController{
		"zero-length ID": {
			ObjectMeta: api.ObjectMeta{Name: "", Namespace: api.NamespaceDefault},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
				Template: &validPodTemplate.Spec,
			},
		},
		"missing-namespace": {
			ObjectMeta: api.ObjectMeta{Name: "abc-123"},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
				Template: &validPodTemplate.Spec,
			},
		},
		"empty selector": {
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
			Spec: api.ReplicationControllerSpec{
				Template: &validPodTemplate.Spec,
			},
		},
		"selector_doesnt_match": {
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
			Spec: api.ReplicationControllerSpec{
				Selector: map[string]string{"foo": "bar"},
				Template: &validPodTemplate.Spec,
			},
		},
		"invalid manifest": {
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
			},
		},
		"read-write persistent disk with > 1 pod": {
			ObjectMeta: api.ObjectMeta{Name: "abc"},
			Spec: api.ReplicationControllerSpec{
				Replicas: 2,
				Selector: validSelector,
				Template: &readWriteVolumePodTemplate.Spec,
			},
		},
		"negative_replicas": {
			ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: api.NamespaceDefault},
			Spec: api.ReplicationControllerSpec{
				Replicas: -1,
				Selector: validSelector,
			},
		},
		"invalid_label": {
			ObjectMeta: api.ObjectMeta{
				Name:      "abc-123",
				Namespace: api.NamespaceDefault,
				Labels: map[string]string{
					"NoUppercaseOrSpecialCharsLike=Equals": "bar",
				},
			},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
				Template: &validPodTemplate.Spec,
			},
		},
		"invalid_label 2": {
			ObjectMeta: api.ObjectMeta{
				Name:      "abc-123",
				Namespace: api.NamespaceDefault,
				Labels: map[string]string{
					"NoUppercaseOrSpecialCharsLike=Equals": "bar",
				},
			},
			Spec: api.ReplicationControllerSpec{
				Template: &invalidPodTemplate.Spec,
			},
		},
		"invalid_annotation": {
			ObjectMeta: api.ObjectMeta{
				Name:      "abc-123",
				Namespace: api.NamespaceDefault,
				Annotations: map[string]string{
					"NoUppercaseOrSpecialCharsLike=Equals": "bar",
				},
			},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
				Template: &validPodTemplate.Spec,
			},
		},
		"invalid restart policy 1": {
			ObjectMeta: api.ObjectMeta{
				Name:      "abc-123",
				Namespace: api.NamespaceDefault,
			},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
				Template: &api.PodTemplateSpec{
					Spec: api.PodSpec{
						RestartPolicy: api.RestartPolicy{
							OnFailure: &api.RestartPolicyOnFailure{},
						},
						DNSPolicy: api.DNSClusterFirst,
					},
					ObjectMeta: api.ObjectMeta{
						Labels: validSelector,
					},
				},
			},
		},
		"invalid restart policy 2": {
			ObjectMeta: api.ObjectMeta{
				Name:      "abc-123",
				Namespace: api.NamespaceDefault,
			},
			Spec: api.ReplicationControllerSpec{
				Selector: validSelector,
				Template: &api.PodTemplateSpec{
					Spec: api.PodSpec{
						RestartPolicy: api.RestartPolicy{
							Never: &api.RestartPolicyNever{},
						},
						DNSPolicy: api.DNSClusterFirst,
					},
					ObjectMeta: api.ObjectMeta{
						Labels: validSelector,
					},
				},
			},
		},
	}
	for k, v := range errorCases {
		errs := ValidateReplicationController(&v)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
		for i := range errs {
			field := errs[i].(*errors.ValidationError).Field
			if !strings.HasPrefix(field, "spec.template.") &&
				field != "metadata.name" &&
				field != "metadata.namespace" &&
				field != "spec.selector" &&
				field != "spec.template" &&
				field != "GCEPersistentDisk.ReadOnly" &&
				field != "spec.replicas" &&
				field != "spec.template.labels" &&
				field != "metadata.annotations" &&
				field != "metadata.labels" {
				t.Errorf("%s: missing prefix for: %v", k, errs[i])
			}
		}
	}
}

func TestValidateMinion(t *testing.T) {
	validSelector := map[string]string{"a": "b"}
	invalidSelector := map[string]string{"NoUppercaseOrSpecialCharsLike=Equals": "b"}
	successCases := []api.Node{
		{
			ObjectMeta: api.ObjectMeta{
				Name:   "abc",
				Labels: validSelector,
			},
			Status: api.NodeStatus{
				HostIP: "something",
			},
		},
		{
			ObjectMeta: api.ObjectMeta{
				Name: "abc",
			},
			Status: api.NodeStatus{
				HostIP: "something",
			},
		},
	}
	for _, successCase := range successCases {
		if errs := ValidateMinion(&successCase); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]api.Node{
		"zero-length Name": {
			ObjectMeta: api.ObjectMeta{
				Name:   "",
				Labels: validSelector,
			},
			Status: api.NodeStatus{
				HostIP: "something",
			},
		},
		"invalid-labels": {
			ObjectMeta: api.ObjectMeta{
				Name:   "abc-123",
				Labels: invalidSelector,
			},
		},
		"invalid-annotations": {
			ObjectMeta: api.ObjectMeta{
				Name:        "abc-123",
				Annotations: invalidSelector,
			},
		},
	}
	for k, v := range errorCases {
		errs := ValidateMinion(&v)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
		for i := range errs {
			field := errs[i].(*errors.ValidationError).Field
			if field != "metadata.name" &&
				field != "metadata.labels" &&
				field != "metadata.annotations" &&
				field != "metadata.namespace" {
				t.Errorf("%s: missing prefix for: %v", k, errs[i])
			}
		}
	}
}

func TestValidateMinionUpdate(t *testing.T) {
	tests := []struct {
		oldMinion api.Node
		minion    api.Node
		valid     bool
	}{
		{api.Node{}, api.Node{}, true},
		{api.Node{
			ObjectMeta: api.ObjectMeta{
				Name: "foo"}},
			api.Node{
				ObjectMeta: api.ObjectMeta{
					Name: "bar"},
			}, false},
		{api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "bar"},
			},
		}, api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "baz"},
			},
		}, true},
		{api.Node{
			ObjectMeta: api.ObjectMeta{
				Name: "foo",
			},
		}, api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "baz"},
			},
		}, true},
		{api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"bar": "foo"},
			},
		}, api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "baz"},
			},
		}, true},
		{api.Node{
			ObjectMeta: api.ObjectMeta{
				Name: "foo",
			},
			Spec: api.NodeSpec{
				Capacity: api.ResourceList{
					api.ResourceCPU:    resource.MustParse("10000"),
					api.ResourceMemory: resource.MustParse("100"),
				},
			},
		}, api.Node{
			ObjectMeta: api.ObjectMeta{
				Name: "foo",
			},
			Spec: api.NodeSpec{
				Capacity: api.ResourceList{
					api.ResourceCPU:    resource.MustParse("100"),
					api.ResourceMemory: resource.MustParse("10000"),
				},
			},
		}, true},
		{api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"bar": "foo"},
			},
			Spec: api.NodeSpec{
				Capacity: api.ResourceList{
					api.ResourceCPU:    resource.MustParse("10000"),
					api.ResourceMemory: resource.MustParse("100"),
				},
			},
		}, api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"bar": "fooobaz"},
			},
			Spec: api.NodeSpec{
				Capacity: api.ResourceList{
					api.ResourceCPU:    resource.MustParse("100"),
					api.ResourceMemory: resource.MustParse("10000"),
				},
			},
		}, true},
		{api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"bar": "foo"},
			},
			Status: api.NodeStatus{
				HostIP: "1.2.3.4",
			},
		}, api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"bar": "fooobaz"},
			},
		}, true},
		{api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "baz"},
			},
		}, api.Node{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"Foo": "baz"},
			},
		}, false},
	}
	for i, test := range tests {
		errs := ValidateMinionUpdate(&test.oldMinion, &test.minion)
		if test.valid && len(errs) > 0 {
			t.Errorf("%d: Unexpected error: %v", i, errs)
			t.Logf("%#v vs %#v", test.oldMinion.ObjectMeta, test.minion.ObjectMeta)
		}
		if !test.valid && len(errs) == 0 {
			t.Errorf("%d: Unexpected non-error", i)
		}
	}
}

func TestValidateServiceUpdate(t *testing.T) {
	tests := []struct {
		oldService api.Service
		service    api.Service
		valid      bool
	}{
		{ // 0
			api.Service{},
			api.Service{},
			true},
		{ // 1
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name: "foo"}},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name: "bar"},
			}, false},
		{ // 2
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"foo": "bar"},
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"foo": "baz"},
				},
			}, true},
		{ // 3
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"foo": "baz"},
				},
			}, true},
		{ // 4
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"bar": "foo"},
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"foo": "baz"},
				},
			}, true},
		{ // 5
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:        "foo",
					Annotations: map[string]string{"bar": "foo"},
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:        "foo",
					Annotations: map[string]string{"foo": "baz"},
				},
			}, true},
		{ // 6
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
				},
				Spec: api.ServiceSpec{
					Selector: map[string]string{"foo": "baz"},
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name: "foo",
				},
				Spec: api.ServiceSpec{
					Selector: map[string]string{"foo": "baz"},
				},
			}, true},
		{ // 7
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"bar": "foo"},
				},
				Spec: api.ServiceSpec{
					PortalIP: "127.0.0.1",
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"bar": "fooobaz"},
				},
				Spec: api.ServiceSpec{
					PortalIP: "new",
				},
			}, false},
		{ // 8
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"bar": "foo"},
				},
				Spec: api.ServiceSpec{
					PortalIP: "127.0.0.1",
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"bar": "fooobaz"},
				},
				Spec: api.ServiceSpec{
					PortalIP: "",
				},
			}, false},
		{ // 9
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"bar": "foo"},
				},
				Spec: api.ServiceSpec{
					PortalIP: "127.0.0.1",
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"bar": "fooobaz"},
				},
				Spec: api.ServiceSpec{
					PortalIP: "127.0.0.2",
				},
			}, false},
		{ // 10
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"foo": "baz"},
				},
			},
			api.Service{
				ObjectMeta: api.ObjectMeta{
					Name:   "foo",
					Labels: map[string]string{"Foo": "baz"},
				},
			}, false},
	}
	for i, test := range tests {
		errs := ValidateServiceUpdate(&test.oldService, &test.service)
		if test.valid && len(errs) > 0 {
			t.Errorf("%d: Unexpected error: %v", i, errs)
			t.Logf("%#v vs %#v", test.oldService.ObjectMeta, test.service.ObjectMeta)
		}
		if !test.valid && len(errs) == 0 {
			t.Errorf("%d: Unexpected non-error", i)
		}
	}
}

func TestValidateResourceNames(t *testing.T) {
	longString := "a"
	for i := 0; i < 6; i++ {
		longString += longString
	}
	table := []struct {
		input   string
		success bool
	}{
		{"memory", true},
		{"cpu", true},
		{"network", false},
		{"disk", false},
		{"", false},
		{".", false},
		{"..", false},
		{"my.favorite.app.co/12345", true},
		{"my.favorite.app.co/_12345", false},
		{"my.favorite.app.co/12345_", false},
		{"kubernetes.io/..", false},
		{"kubernetes.io/" + longString, false},
		{"kubernetes.io//", false},
		{"kubernetes.io", false},
		{"kubernetes.io/will/not/work/", false},
	}
	for k, item := range table {
		err := validateResourceName(item.input, "sth")
		if len(err) != 0 && item.success {
			t.Errorf("expected no failure for input %q", item.input)
		} else if len(err) == 0 && !item.success {
			t.Errorf("expected failure for input %q", item.input)
			for i := range err {
				detail := err[i].(*errors.ValidationError).Detail
				if detail != "" && detail != qualifiedNameErrorMsg {
					t.Errorf("%s: expected error detail either empty or %s, got %s", k, qualifiedNameErrorMsg, detail)
				}
			}
		}
	}
}

func TestValidateLimitRange(t *testing.T) {
	spec := api.LimitRangeSpec{
		Limits: []api.LimitRangeItem{
			{
				Type: api.LimitTypePod,
				Max: api.ResourceList{
					api.ResourceCPU:    resource.MustParse("100"),
					api.ResourceMemory: resource.MustParse("10000"),
				},
				Min: api.ResourceList{
					api.ResourceCPU:    resource.MustParse("0"),
					api.ResourceMemory: resource.MustParse("100"),
				},
			},
		},
	}

	successCases := []api.LimitRange{
		{
			ObjectMeta: api.ObjectMeta{
				Name:      "abc",
				Namespace: "foo",
			},
			Spec: spec,
		},
	}

	for _, successCase := range successCases {
		if errs := ValidateLimitRange(&successCase); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]struct {
		R api.LimitRange
		D string
	}{
		"zero-length Name": {
			api.LimitRange{ObjectMeta: api.ObjectMeta{Name: "", Namespace: "foo"}, Spec: spec},
			"",
		},
		"zero-length-namespace": {
			api.LimitRange{ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: ""}, Spec: spec},
			"",
		},
		"invalid Name": {
			api.LimitRange{ObjectMeta: api.ObjectMeta{Name: "^Invalid", Namespace: "foo"}, Spec: spec},
			dnsSubdomainErrorMsg,
		},
		"invalid Namespace": {
			api.LimitRange{ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: "^Invalid"}, Spec: spec},
			dnsSubdomainErrorMsg,
		},
	}
	for k, v := range errorCases {
		errs := ValidateLimitRange(&v.R)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
		for i := range errs {
			field := errs[i].(*errors.ValidationError).Field
			detail := errs[i].(*errors.ValidationError).Detail
			if field != "metadata.name" && field != "metadata.namespace" {
				t.Errorf("%s: missing prefix for: %v", k, errs[i])
			}
			if detail != v.D {
				t.Errorf("%s: expected error detail either empty or %s, got %s", k, v.D, detail)
			}
		}
	}
}

func TestValidateResourceQuota(t *testing.T) {
	spec := api.ResourceQuotaSpec{
		Hard: api.ResourceList{
			api.ResourceCPU:                    resource.MustParse("100"),
			api.ResourceMemory:                 resource.MustParse("10000"),
			api.ResourcePods:                   resource.MustParse("10"),
			api.ResourceServices:               resource.MustParse("10"),
			api.ResourceReplicationControllers: resource.MustParse("10"),
			api.ResourceQuotas:                 resource.MustParse("10"),
		},
	}

	successCases := []api.ResourceQuota{
		{
			ObjectMeta: api.ObjectMeta{
				Name:      "abc",
				Namespace: "foo",
			},
			Spec: spec,
		},
	}

	for _, successCase := range successCases {
		if errs := ValidateResourceQuota(&successCase); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}

	errorCases := map[string]struct {
		R api.ResourceQuota
		D string
	}{
		"zero-length Name": {
			api.ResourceQuota{ObjectMeta: api.ObjectMeta{Name: "", Namespace: "foo"}, Spec: spec},
			"",
		},
		"zero-length Namespace": {
			api.ResourceQuota{ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: ""}, Spec: spec},
			"",
		},
		"invalid Name": {
			api.ResourceQuota{ObjectMeta: api.ObjectMeta{Name: "^Invalid", Namespace: "foo"}, Spec: spec},
			dnsSubdomainErrorMsg,
		},
		"invalid Namespace": {
			api.ResourceQuota{ObjectMeta: api.ObjectMeta{Name: "abc", Namespace: "^Invalid"}, Spec: spec},
			dnsSubdomainErrorMsg,
		},
	}
	for k, v := range errorCases {
		errs := ValidateResourceQuota(&v.R)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
		for i := range errs {
			field := errs[i].(*errors.ValidationError).Field
			detail := errs[i].(*errors.ValidationError).Detail
			if field != "metadata.name" && field != "metadata.namespace" {
				t.Errorf("%s: missing prefix for: %v", k, errs[i])
			}
			if detail != v.D {
				t.Errorf("%s: expected error detail either empty or %s, got %s", k, v.D, detail)
			}
		}
	}
}

func TestValidateNamespace(t *testing.T) {
	validLabels := map[string]string{"a": "b"}
	invalidLabels := map[string]string{"NoUppercaseOrSpecialCharsLike=Equals": "b"}
	successCases := []api.Namespace{
		{
			ObjectMeta: api.ObjectMeta{Name: "abc", Labels: validLabels},
		},
		{
			ObjectMeta: api.ObjectMeta{Name: "abc-123"},
		},
	}
	for _, successCase := range successCases {
		if errs := ValidateNamespace(&successCase); len(errs) != 0 {
			t.Errorf("expected success: %v", errs)
		}
	}
	errorCases := map[string]struct {
		R api.Namespace
		D string
	}{
		"zero-length name": {
			api.Namespace{ObjectMeta: api.ObjectMeta{Name: ""}},
			"",
		},
		"defined-namespace": {
			api.Namespace{ObjectMeta: api.ObjectMeta{Name: "abc-123", Namespace: "makesnosense"}},
			"",
		},
		"invalid-labels": {
			api.Namespace{ObjectMeta: api.ObjectMeta{Name: "abc", Labels: invalidLabels}},
			"",
		},
	}
	for k, v := range errorCases {
		errs := ValidateNamespace(&v.R)
		if len(errs) == 0 {
			t.Errorf("expected failure for %s", k)
		}
	}
}

func TestValidateNamespaceUpdate(t *testing.T) {
	tests := []struct {
		oldNamespace api.Namespace
		namespace    api.Namespace
		valid        bool
	}{
		{api.Namespace{}, api.Namespace{}, true},
		{api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name: "foo"}},
			api.Namespace{
				ObjectMeta: api.ObjectMeta{
					Name: "bar"},
			}, false},
		{api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "bar"},
			},
		}, api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "baz"},
			},
		}, true},
		{api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name: "foo",
			},
		}, api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "baz"},
			},
		}, true},
		{api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"bar": "foo"},
			},
		}, api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "baz"},
			},
		}, true},
		{api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"foo": "baz"},
			},
		}, api.Namespace{
			ObjectMeta: api.ObjectMeta{
				Name:   "foo",
				Labels: map[string]string{"Foo": "baz"},
			},
		}, false},
	}
	for i, test := range tests {
		errs := ValidateNamespaceUpdate(&test.oldNamespace, &test.namespace)
		if test.valid && len(errs) > 0 {
			t.Errorf("%d: Unexpected error: %v", i, errs)
			t.Logf("%#v vs %#v", test.oldNamespace.ObjectMeta, test.namespace.ObjectMeta)
		}
		if !test.valid && len(errs) == 0 {
			t.Errorf("%d: Unexpected non-error", i)
		}
	}
}

func TestValidateSecret(t *testing.T) {
	validSecret := func() api.Secret {
		return api.Secret{
			ObjectMeta: api.ObjectMeta{Name: "foo", Namespace: "bar"},
			Data: map[string][]byte{
				"data-1": []byte("bar"),
			},
		}
	}

	var (
		emptyName   = validSecret()
		invalidName = validSecret()
		emptyNs     = validSecret()
		invalidNs   = validSecret()
		overMaxSize = validSecret()
		invalidKey  = validSecret()
	)

	emptyName.Name = ""
	invalidName.Name = "NoUppercaseOrSpecialCharsLike=Equals"
	emptyNs.Namespace = ""
	invalidNs.Namespace = "NoUppercaseOrSpecialCharsLike=Equals"
	overMaxSize.Data = map[string][]byte{
		"over": make([]byte, api.MaxSecretSize+1),
	}
	invalidKey.Data["a..b"] = []byte("whoops")

	tests := map[string]struct {
		secret api.Secret
		valid  bool
	}{
		"valid":             {validSecret(), true},
		"empty name":        {emptyName, false},
		"invalid name":      {invalidName, false},
		"empty namespace":   {emptyNs, false},
		"invalid namespace": {invalidNs, false},
		"over max size":     {overMaxSize, false},
		"invalid key":       {invalidKey, false},
	}

	for name, tc := range tests {
		errs := ValidateSecret(&tc.secret)
		if tc.valid && len(errs) > 0 {
			t.Errorf("%v: Unexpected error: %v", name, errs)
		}
		if !tc.valid && len(errs) == 0 {
			t.Errorf("%v: Unexpected non-error", name)
		}
	}
}
