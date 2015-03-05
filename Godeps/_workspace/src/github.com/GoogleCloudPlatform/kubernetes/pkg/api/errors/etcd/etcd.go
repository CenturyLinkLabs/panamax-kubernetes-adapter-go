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

package etcd

import (
	"github.com/GoogleCloudPlatform/kubernetes/pkg/api/errors"
	"github.com/GoogleCloudPlatform/kubernetes/pkg/tools"
)

// InterpretGetError converts a generic etcd error on a retrieval
// operation into the appropriate API error.
func InterpretGetError(err error, kind, name string) error {
	switch {
	case tools.IsEtcdNotFound(err):
		return errors.NewNotFound(kind, name)
	default:
		return err
	}
}

// InterpretCreateError converts a generic etcd error on a create
// operation into the appropriate API error.
func InterpretCreateError(err error, kind, name string) error {
	switch {
	case tools.IsEtcdNodeExist(err):
		return errors.NewAlreadyExists(kind, name)
	default:
		return err
	}
}

// InterpretUpdateError converts a generic etcd error on a update
// operation into the appropriate API error.
func InterpretUpdateError(err error, kind, name string) error {
	switch {
	case tools.IsEtcdTestFailed(err), tools.IsEtcdNodeExist(err):
		return errors.NewConflict(kind, name, err)
	default:
		return err
	}
}

// InterpretDeleteError converts a generic etcd error on a delete
// operation into the appropriate API error.
func InterpretDeleteError(err error, kind, name string) error {
	switch {
	case tools.IsEtcdNotFound(err):
		return errors.NewNotFound(kind, name)
	default:
		return err
	}
}
