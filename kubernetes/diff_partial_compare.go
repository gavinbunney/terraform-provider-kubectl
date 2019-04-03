package kubernetes

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

func compareMaps(original, returned map[string]interface{}) (string, error) {
	fields, err := getReturnedValueForOriginalFields(original, returned)
	if err != nil {
		return "", err
	}

	// As the orginal and returned object may have fields set in different orders
	// we use sort here to ensure the same fields always produce the same output
	// no matter what the order
	sort.Strings(fields)

	builder := strings.Builder{}
	for _, f := range fields {
		builder.WriteString(f)
	}

	return builder.String(), nil
}

// getReturnedValueForOriginalFields loops over all fields set in the origin item and returns the
// value that field now holds in the returned item.
// This is necessary as mutating admissions controllers may manipulate the values of items in the cluster
// and these mutations should not be flagged as a change in TF. So we take the returned value from the cluster
// and then build a list of field values for those set on the orignal object.
func getReturnedValueForOriginalFields(original, returned map[string]interface{}) ([]string, error) {
	fields := []string{}
	for oKeyTop, oValueTop := range original {
		for rKeyTop, rValueTop := range returned {
			// Skip if we're not looking at the same key
			if oKeyTop != rKeyTop {
				continue
			}

			// Skip if it's an ignored field
			if shouldSkip(oKeyTop, oValueTop, rValueTop) {
				continue
			}

			// If we're looking at a nested map then recurse into it
			if _, ok := oValueTop.(map[string]interface{}); ok {
				newFields, err := getReturnedValueForOriginalFields(oValueTop.(map[string]interface{}), rValueTop.(map[string]interface{}))
				if err != nil {
					return nil, err
				}
				fields = append(fields, newFields...)
				continue
			}

			// If it's a map[string]string convert then recurse
			if _, ok := oValueTop.(map[string]string); ok {

				newFields, err := getReturnedValueForOriginalFields(convertToMapStringInterface(oValueTop), convertToMapStringInterface(rValueTop))
				if err != nil {
					return nil, err
				}
				fields = append(fields, newFields...)

				continue
			}

			// Check for simple types
			fields = append(fields, fmt.Sprintf("fieldName:%s,fieldValue:%+v", oKeyTop, rValueTop))
		}
	}

	return fields, nil
}

func convertToMapStringInterface(input interface{}) map[string]interface{} {
	castInput := input.(map[string]string)
	converted := make(map[string]interface{})
	for k, v := range castInput {
		converted[k] = v
	}
	return converted
}

var skipFields = map[string]bool{
	"status":            true,
	"finalizers":        true,
	"initializers":      true,
	"ownerReferences":   true,
	"creationTimestamp": true,
	"generation":        true,
	"resourceVersion":   true,
	"uid":               true,
}

func shouldSkip(fieldName string, original, returned interface{}) bool {
	// Skip any fields we want to ignore
	if _, exists := skipFields[fieldName]; exists {
		log.Printf("[COMPARE] Skipping as in SkipFields: %#v %#v", original, returned)
		return true
	}
	return false
}
