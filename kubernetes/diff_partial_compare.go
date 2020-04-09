package kubernetes

import (
	"fmt"
	"log"
	"sort"
	"strings"
)

func compareMaps(original, returned map[string]interface{}, ignoreFields []string) (string, error) {
	fields, err := getReturnedValueForOriginalFields(original, returned, ignoreFields)
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
// and then build a list of field values for those set on the original object.
func getReturnedValueForOriginalFields(original, returned map[string]interface{}, ignoreFields []string) ([]string, error) {
	fields := []string{}
	for oKeyTop, oValueTop := range original {
		for rKeyTop, rValueTop := range returned {
			// Skip if we're not looking at the same key
			if oKeyTop != rKeyTop {
				continue
			}

			if len(ignoreFields) > 0 {
				var shouldIgnore = false
				for _, fieldToIgnore := range ignoreFields {
					if fieldToIgnore == oKeyTop {
						log.Printf("[TRACE] Skipping as in ignoreFields [%v]: %#v %#v", oKeyTop, original, returned)
						shouldIgnore = true
						break
					}
				}

				if shouldIgnore {
					continue
				}
			}

			// Skip if it's an ignored field
			if shouldSkip(oKeyTop, oValueTop, rValueTop) {
				continue
			}

			// If we're looking at a nested map then recurse into it
			fieldsFound, foundMaps, err := handleMaps(oValueTop, rValueTop, ignoreFields)
			if err != nil {
				return []string{}, err
			}
			if foundMaps {
				// this one was a map and we've handled it.
				fields = append(fields, fieldsFound...)
				continue
			}

			// Handle array returned types
			// Todo: probably needs to be more recursive
			if arrayReturned, ok := rValueTop.([]interface{}); ok {
				for i, _ := range arrayReturned {

					if oValueTop == nil {
						continue
					}

					// check if we are outside bounds when array is added on either side that's not in the other
					oValueArray := oValueTop.([]interface{})
					if len(oValueArray)-1 < i || len(arrayReturned)-1 < i {
						if len(arrayReturned) > i {
							fields = append(fields, fmt.Sprintf("fieldName:%s,fieldValue:%+v", fmt.Sprintf("%v[%v]", oKeyTop, i), arrayReturned[i]))
						} else {
							fields = append(fields, fmt.Sprintf("fieldName:%s,fieldValue:%+v", fmt.Sprintf("%v[%v]", oKeyTop, i), ""))
						}
					} else {
						// Again if we're looking at a nested map then recurse into it
						fieldsFound, foundMaps, err := handleMaps(oValueArray[i], arrayReturned[i], ignoreFields)
						if err != nil {
							return []string{}, err
						}
						if foundMaps {
							// this one was a map and we've handled it.
							fields = append(fields, fieldsFound...)
							continue
						}

						// Otherwise it's probably something else so can be printed
						fields = append(fields, fmt.Sprintf("fieldName:%s,fieldValue:%+v", fmt.Sprintf("%v[%v]", oKeyTop, i), arrayReturned[i]))
					}
				}

				continue
			}

			// Check for simple types
			fields = append(fields, fmt.Sprintf("fieldName:%s,fieldValue:%+v", oKeyTop, rValueTop))
		}
	}

	return fields, nil
}

func handleMaps(oValue, rValue interface{}, ignoreFields []string) ([]string, bool, error) {
	fields := []string{}

	// If we're looking at a nested map then recurse into it
	if _, ok := oValue.(map[string]interface{}); ok {
		newFields, err := getReturnedValueForOriginalFields(oValue.(map[string]interface{}), rValue.(map[string]interface{}), ignoreFields)
		if err != nil {
			return []string{}, false, err
		}
		fields = append(fields, newFields...)
		return fields, true, nil
	}

	// If it's a map[string]string convert then recurse
	if _, ok := oValue.(map[string]string); ok {

		newFields, err := getReturnedValueForOriginalFields(convertToMapStringInterface(oValue), convertToMapStringInterface(rValue), ignoreFields)
		if err != nil {
			return []string{}, false, err
		}
		fields = append(fields, newFields...)
		return fields, true, nil
	}

	return []string{}, false, nil
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
		log.Printf("[TRACE] Skipping as in SkipFields: %#v %#v", original, returned)
		return true
	}
	return false
}
