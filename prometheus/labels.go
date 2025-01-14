// Copyright 2018 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package prometheus

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/prometheus/common/model"
)

// Labels represents a collection of label name -> value mappings. This type is
// commonly used with the With(Labels) and GetMetricWith(Labels) methods of
// metric vector Collectors, e.g.:
//
//	myVec.With(Labels{"code": "404", "method": "GET"}).Add(42)
//
// The other use-case is the specification of constant label pairs in Opts or to
// create a Desc.
type Labels map[string]string

// ConstrainedLabels represents a label name and its constrain function
// to normalize label values. This type is commonly used when constructing
// metric vector Collectors.
type ConstrainedLabel struct {
	Name       string
	Constraint func(string) string
}

func (cl ConstrainedLabel) Constrain(v string) string {
	if cl.Constraint == nil {
		return v
	}
	return cl.Constraint(v)
}

// ConstrainableLabels is an interface that allows creating of labels that can
// be optionally constrained.
//
//	prometheus.V2().NewCounterVec(CounterVecOpts{
//	  CounterOpts: {...}, // Usual CounterOpts fields
//	  VariableLabels: []ConstrainedLabels{
//	    {Name: "A"},
//	    {Name: "B", Constraint: func(v string) string { ... }},
//	  },
//	})
type ConstrainableLabels interface {
	constrainedLabels() ConstrainedLabels
	labelNames() []string
}

// ConstrainedLabels represents a collection of label name -> constrain function
// to normalize label values. This type is commonly used when constructing
// metric vector Collectors.
type ConstrainedLabels []ConstrainedLabel

func (cls ConstrainedLabels) constrainedLabels() ConstrainedLabels {
	return cls
}

func (cls ConstrainedLabels) labelNames() []string {
	names := make([]string, len(cls))
	for i, label := range cls {
		names[i] = label.Name
	}
	return names
}

func (cls ConstrainedLabels) indexOfLabelNames(target string) (int, bool) {
	for i, label := range cls {
		if label.Name == target {
			return i, true
		}
	}
	return len(cls), false
}

// UnconstrainedLabels represents collection of label without any constraint on
// their value. Thus, it is simply a collection of label names.
//
//	UnconstrainedLabels([]string{ "A", "B" })
//
// is equivalent to
//
//	ConstrainedLabels {
//	  { Name: "A" },
//	  { Name: "B" },
//	}
type UnconstrainedLabels []string

func (uls UnconstrainedLabels) constrainedLabels() ConstrainedLabels {
	constrainedLabels := make([]ConstrainedLabel, len(uls))
	for i, l := range uls {
		constrainedLabels[i] = ConstrainedLabel{Name: l}
	}
	return constrainedLabels
}

func (uls UnconstrainedLabels) labelNames() []string {
	return uls
}

// reservedLabelPrefix is a prefix which is not legal in user-supplied
// label names.
const reservedLabelPrefix = "__"

var errInconsistentCardinality = errors.New("inconsistent label cardinality")

func makeInconsistentCardinalityError(fqName string, labels, labelValues []string) error {
	return fmt.Errorf(
		"%w: %q has %d variable labels named %q but %d values %q were provided",
		errInconsistentCardinality, fqName,
		len(labels), labels,
		len(labelValues), labelValues,
	)
}

func validateValuesInLabels(labels Labels, expectedNumberOfValues int) error {
	if len(labels) != expectedNumberOfValues {
		return fmt.Errorf(
			"%w: expected %d label values but got %d in %#v",
			errInconsistentCardinality, expectedNumberOfValues,
			len(labels), labels,
		)
	}

	for name, val := range labels {
		if !utf8.ValidString(val) {
			return fmt.Errorf("label %s: value %q is not valid UTF-8", name, val)
		}
	}

	return nil
}

func validateLabelValues(vals []string, expectedNumberOfValues int) error {
	if len(vals) != expectedNumberOfValues {
		return fmt.Errorf(
			"%w: expected %d label values but got %d in %#v",
			errInconsistentCardinality, expectedNumberOfValues,
			len(vals), vals,
		)
	}

	for _, val := range vals {
		if !utf8.ValidString(val) {
			return fmt.Errorf("label value %q is not valid UTF-8", val)
		}
	}

	return nil
}

func checkLabelName(l string) bool {
	return model.LabelName(l).IsValid() && !strings.HasPrefix(l, reservedLabelPrefix)
}
