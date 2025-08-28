/*
Copyright 2025.

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

package util

// finalizerName is the name of the finalizer used by the litellm operator
const FinalizerName = "litellm-operator.litellm.ai/finalizer"

// ensureMetadata ensures that the metadata contains the managed_by metadata
func EnsureMetadata(metadata map[string]string) map[string]string {
	operatorMetadata := map[string]string{
		"managed_by": "litellm-operator",
	}
	for k, v := range metadata {
		operatorMetadata[k] = v
	}
	return operatorMetadata
}

func StringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func IntPtrOrNil(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

// DerefString returns the value of a *string or "" if nil.
func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// DerefInt returns the value of a *int or 0 if nil.
func DerefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// DerefFloat64 returns the value of a *float64 or 0.0 if nil.
func DerefFloat64(f *float64) float64 {
	if f == nil {
		return 0.0
	}
	return *f
}

// DerefBool returns the value of a *bool or false if nil.
func DerefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
