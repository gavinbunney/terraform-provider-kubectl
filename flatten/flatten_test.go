package flatten

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFlattenMap(t *testing.T) {
	testCases := []struct {
		description string
		test        map[string]interface{}
		expected    map[string]string
	}{
		{
			description: "Simple map with string value",
			test: map[string]interface{}{
				"test1": "test2",
			},
			expected: map[string]string{
				"test1": "test2",
			},
		},
		{
			description: "All primitive types",
			test: map[string]interface{}{
				"a_string":  "test2",
				"a_boolean": true,
				"a_number":  123,
				"a_float":   123.456,
			},
			expected: map[string]string{
				"a_boolean": "true",
				"a_float":   "123.456",
				"a_number":  "123",
				"a_string":  "test2",
			},
		},
		{
			description: "Map with empty keys",
			test: map[string]interface{}{
				"": "",
			},
			expected: map[string]string{},
		},
		{
			description: "Empty map",
			test:        map[string]interface{}{},
			expected:    map[string]string{},
		},
		{
			description: "Nil map",
			test:        nil,
			expected:    map[string]string{},
		},
		{
			description: "One level map",
			test: map[string]interface{}{
				"atest": "test",
				"meta": map[string]interface{}{
					"annotations": map[string]string{
						"helm.sh/hook": "crd-install",
					},
				},
			},
			expected: map[string]string{
				"atest":                         "test",
				"meta.annotations.helm.sh/hook": "crd-install",
			},
		},
		{
			description: "One level map empty value",
			test: map[string]interface{}{
				"atest": "test",
				"meta":  map[string]interface{}{},
			},
			expected: map[string]string{
				"atest": "test",
			},
		},
		{
			description: "One level map, nil value",
			test: map[string]interface{}{
				"atest": "test",
				"meta":  nil,
			},
			expected: map[string]string{
				"atest": "test",
			},
		},
		{
			description: "One level slice",
			test: map[string]interface{}{
				"my-slice": []string{"first", "second"},
			},
			expected: map[string]string{
				"my-slice.#": "2",
				"my-slice.0": "first",
				"my-slice.1": "second",
			},
		},
		{
			description: "Map with slice elements",
			test: map[string]interface{}{
				"meta": map[string]interface{}{
					"my-slice": []string{"first", "second"},
				},
			},
			expected: map[string]string{
				"meta.my-slice.#": "2",
				"meta.my-slice.0": "first",
				"meta.my-slice.1": "second",
			},
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.description, func(t *testing.T) {
			result := Flatten(tcase.test)
			assert.Equal(t, tcase.expected, result)
		})
	}
}
