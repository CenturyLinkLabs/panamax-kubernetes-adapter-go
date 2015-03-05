/*
Copyright 2015 Google Inc. All rights reserved.

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

package probe

type Result int

// Status values must be one of these constants.
const (
	Success Result = iota
	Failure
	Unknown
)

func (s Result) String() string {
	switch s {
	case Success:
		return "success"
	case Failure:
		return "failure"
	default:
		return "unknown"
	}
}
