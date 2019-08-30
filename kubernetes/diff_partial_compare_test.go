package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPartialCompare(t *testing.T) {
	testCases := []struct {
		description    string
		expectedString string
		original       map[string]interface{}
		returned       map[string]interface{}
		ignored        []string
	}{
		{
			description: "Simple map with string value",
			original: map[string]interface{}{
				"test1": "test2",
			},
			returned: map[string]interface{}{
				"test1": "test2",
			},
			expectedString: "fieldName:test1,fieldValue:test2",
		},
		{
			// Ensure skippable fields are skipped
			description: "Simple map with string value and Skippable fields",
			original: map[string]interface{}{
				"test1":           "test2",
				"resourceVersion": "1245",
			},
			returned: map[string]interface{}{
				"test1":           "test2",
				"resourceVersion": "1245",
			},
			expectedString: "fieldName:test1,fieldValue:test2",
		},
		{
			// Ensure ignored fields are skipped
			description: "Simple map with string value and ignored fields",
			original: map[string]interface{}{
				"test1":           "test2",
				"ignoreThis": "1245",
			},
			returned: map[string]interface{}{
				"test1":           "test2",
				"ignoreThis": "1245",
			},
			expectedString: "fieldName:test1,fieldValue:test2",
			ignored: []string{"ignoreThis"},
		},
		{
			// Ensure nested `map[string]string` are supported
			description: "Map with nested map[string]string",
			original: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob": "bill",
				},
			},
			returned: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob": "bill",
				},
			},
			expectedString: "fieldName:bob,fieldValue:billfieldName:test1,fieldValue:test2",
		},
		{
			// Ensure nested `map[string]string` with different ordering are supported
			description: "Map with nested map[string]string with different ordering",
			original: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob1": "bill",
					"bob2": "bill",
					"bob3": "bill",
				},
			},
			returned: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"bob2": "bill",
					"bob1": "bill",
					"bob3": "bill",
				},
			},
			expectedString: "fieldName:bob1,fieldValue:billfieldName:bob2,fieldValue:billfieldName:bob3,fieldValue:billfieldName:test1,fieldValue:test2",
		},
		{
			description: "Map with nested map[string]string with nested array",
			original: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						"a",
						"b",
						"c",
					},
				},
			},
			returned: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						"c",
						"b",
						"a",
					},
				},
			},
			expectedString: "fieldName:bob1[0],fieldValue:cfieldName:bob1[1],fieldValue:bfieldName:bob1[2],fieldValue:afieldName:test1,fieldValue:test2",
		},
		{
			description: "Map with nested map[string]string with nested array and nested map",
			original: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						map[string]string{
							"1": "1",
							"2": "2",
							"3": "3",
						},
						map[string]interface{}{
							"1": 1,
							"2": 2,
							"3": 3,
						},
					},
				},
			},
			returned: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]interface{}{
					"bob1": []interface{}{
						map[string]string{
							"2": "2",
							"1": "1",
							"3": "3",
						},
						map[string]interface{}{
							"2": 2,
							"1": 1,
							"3": 3,
						},
					},
				},
			},
			expectedString: "fieldName:1,fieldValue:1fieldName:1,fieldValue:1fieldName:2,fieldValue:2fieldName:2,fieldValue:2fieldName:3,fieldValue:3fieldName:3,fieldValue:3fieldName:test1,fieldValue:test2",
		},
		{
			// Ensure ordering of the fields doesn't affect matching
			description: "Different Ordering",
			original: map[string]interface{}{
				"ztest1": "test2",
				"afield": "test2",
			},
			returned: map[string]interface{}{
				"afield": "test2",
				"ztest1": "test2",
			},
			expectedString: "fieldName:afield,fieldValue:test2fieldName:ztest1,fieldValue:test2",
		},
		{
			// Ensure nested arrays are handled
			description: "Nested Array",
			original: map[string]interface{}{
				"ztest1": []string{
					"1", "2",
				},
				"afield": "test2",
			},
			returned: map[string]interface{}{
				"afield": "test2",
				"ztest1": []string{
					"1", "2",
				},
			},
			expectedString: "fieldName:afield,fieldValue:test2fieldName:ztest1,fieldValue:[1 2]",
		},
		{
			// Ensure fields added to the `returned` which aren't present in the `originl` are ignored
			description: "Ignore additional fields",
			original: map[string]interface{}{
				"afield": "test2",
			},
			returned: map[string]interface{}{
				"afield": "test2",
				"ztest1": []string{
					"1", "2",
				},
			},
			expectedString: "fieldName:afield,fieldValue:test2",
		},
		{
			// Ensure that fields present in the `original` but missing in the `returned` are skipped
			description: "Handle removed fields",
			original: map[string]interface{}{
				"afield":   "test2",
				"igetlost": "test2",
			},
			returned: map[string]interface{}{
				"afield": "test2",
			},
			expectedString: "fieldName:afield,fieldValue:test2",
		},
		{
			description: "Handle integers",
			original: map[string]interface{}{
				"afield": 1,
			},
			returned: map[string]interface{}{
				"afield": 1,
			},
			expectedString: "fieldName:afield,fieldValue:1",
		},
		{
			// Ensure that the updated value for `afield` on the `returned` object is taken
			description: "Handle updated field. Expect returned value to be shown",
			original: map[string]interface{}{
				"afield": 1,
			},
			returned: map[string]interface{}{
				"afield": 2,
			},
			expectedString: "fieldName:afield,fieldValue:2",
		},
		{
			// Ensure that the updated value fo the `returned` object is taken for the `willchange` field
			description: "Map with nested map[string]string with updated field",
			original: map[string]interface{}{
				"test1": "test2",
				"nest": map[string]string{
					"willchange": "bill",
				},
			},
			returned: map[string]interface{}{
				"nest": map[string]string{
					"willchange": "updatedbill",
				},
			},
			expectedString: "fieldName:willchange,fieldValue:updatedbill",
		},
		{
			// Ensure that both fields are tracked in the output
			description: "Handle duplicate name fields in nested maps",
			original: map[string]interface{}{
				"atest": "test",
				"nest": map[string]string{
					"atest": "bill",
				},
			},
			returned: map[string]interface{}{
				"atest": "test",
				"nest": map[string]string{
					"atest": "bill",
				},
			},
			expectedString: "fieldName:atest,fieldValue:billfieldName:atest,fieldValue:test",
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.description, func(t *testing.T) {
			result, err := compareMaps(tcase.original, tcase.returned, tcase.ignored)
			assert.NoError(t, err, "Expect compareMaps to succeed")

			assert.Equal(t, tcase.expectedString, result, "Expect the builder output to match")
		})
	}
}
