// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package charm_test

import (
	"bytes"
	"sort"

	gc "gopkg.in/check.v1"

	"gopkg.in/juju/charm.v4"
)

// Keys returns a list of all defined metrics keys.
func Keys(m *charm.Metrics) []string {
	result := make([]string, 0, len(m.Metrics))

	for name := range m.Metrics {
		result = append(result, name)

	}
	sort.Strings(result)
	return result
}

type MetricsSuite struct{}

var _ = gc.Suite(&MetricsSuite{})

func (s *MetricsSuite) TestReadEmpty(c *gc.C) {
	metrics, err := charm.ReadMetrics(bytes.NewBuffer([]byte{}))
	c.Assert(err, gc.IsNil)
	c.Assert(metrics, gc.NotNil)
	c.Assert(Keys(metrics), gc.HasLen, 0)
}

func (s *MetricsSuite) TestReadAlmostEmpty(c *gc.C) {
	metrics, err := charm.ReadMetrics(bytes.NewBuffer([]byte(`
metrics:
`)))
	c.Assert(err, gc.IsNil)
	c.Assert(metrics, gc.NotNil)
	c.Assert(Keys(metrics), gc.HasLen, 0)
}

func (s *MetricsSuite) TestNoDescription(c *gc.C) {
	metrics, err := charm.ReadMetrics(bytes.NewBuffer([]byte(`
metrics:
  some-metric:
    type: gauge
`)))
	c.Assert(err, gc.ErrorMatches, "invalid metrics declaration: metric \"some-metric\" lacks description")
	c.Assert(metrics, gc.IsNil)
}

func (s *MetricsSuite) TestIncorrectType(c *gc.C) {
	metrics, err := charm.ReadMetrics(bytes.NewBuffer([]byte(`
metrics:
  some-metric:
    type: not-a-type
    description: Some description.
`)))
	c.Assert(err, gc.ErrorMatches, "invalid metrics declaration: metric \"some-metric\" has unknown type \"not-a-type\"")
	c.Assert(metrics, gc.IsNil)
}

func (s *MetricsSuite) TestValidYaml(c *gc.C) {
	metrics, err := charm.ReadMetrics(bytes.NewBuffer([]byte(`
metrics:
  blips:
    type: absolute
    description: An absolute metric.
  blops:
    type: gauge
    description: A gauge metric.
  juju-unit-time:
    type: gauge
    description: Unit time.
`)))
	c.Assert(err, gc.IsNil)
	c.Assert(metrics, gc.NotNil)
	c.Assert(Keys(metrics), gc.DeepEquals, []string{"blips", "blops", "juju-unit-time"})

	testCases := []struct {
		about string
		name  string
		value string
		err   string
	}{{
		about: "valid gauge metric",
		name:  "blops",
		value: "1",
		err:   "",
	}, {
		about: "valid absolute metric",
		name:  "blips",
		value: "0",
		err:   "",
	}, {
		about: "valid gauge metric, float value",
		name:  "blops",
		value: "0.15",
		err:   "",
	}, {
		about: "valid absolute metric, float value",
		name:  "blips",
		value: "6.015e15",
		err:   "",
	}, {
		about: "undeclared metric",
		name:  "undeclared",
		value: "6.015e15",
		err:   "metric \"undeclared\" not defined",
	}, {
		about: "invalid type for gauge metric",
		name:  "blops",
		value: "true",
		err:   "invalid value type: expected float, got \"true\"",
	}}

	for i, t := range testCases {
		c.Logf("test %d: %s", i, t.about)
		err := metrics.ValidateMetric(t.name, t.value)
		if t.err == "" {
			c.Check(err, gc.IsNil)
		} else {
			c.Check(err, gc.ErrorMatches, t.err)
		}
	}

}
