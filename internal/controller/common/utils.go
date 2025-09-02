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

package common

import (
	"errors"
	"strconv"
	"strings"
)

// Shared constants used across multiple controllers
const (
	// Model source tags to indicate where models originate from
	ModelTagCRD  = "-[crd]"  // Models created from CRD resources
	ModelTagInst = "-[inst]" // Models created from LiteLLMInstance resources
)

// ParseAndAssign parses a string field and assigns the float64 value to target.
// Used for parsing cost and budget fields across multiple controllers.
func ParseAndAssign(field *string, target *float64, fieldName string) error {
	if field != nil && *field != "" {
		value, err := strconv.ParseFloat(*field, 64)
		if err != nil {
			return errors.New(fieldName + " not parsable to float")
		}
		if target != nil {
			*target = value
		}
	}
	return nil
}

// AppendModelSourceTag appends a short tag to the provided modelName if not already present.
// Used to differentiate models created from different sources (CRD vs LiteLLMInstance).
func AppendModelSourceTag(modelName string, tag string) string {
	if strings.HasSuffix(modelName, tag) {
		return modelName
	}
	return modelName + tag
}
