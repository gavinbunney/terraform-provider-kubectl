package kubernetes

import (
	"fmt"
	"log"
	"reflect"
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

func compareObjsInternal(originalObj, returnedObj reflect.Value, builder *strings.Builder) error {
	originalObType := originalObj.Type()
	if originalObType.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, but got %+v: %+v", originalObj.Kind(), originalObj)
	}

	returnedObjType := returnedObj.Type()
	if returnedObjType.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, but got %+v: %+v", returnedObj.Kind(), returnedObj)
	}

	// Loop through all fields on the original Obj
	// for each field on the original get it's value on the returned obj
	// and use this to build a hash
	for iO := 0; iO < originalObType.NumField(); iO++ {
		originalField := originalObType.Field(iO)

		for iR := 0; iR < returnedObjType.NumField(); iR++ {
			returnedField := returnedObjType.Field(iR)

			// Check we're comparing the right field
			if returnedField.Name != originalField.Name {
				log.Printf("[COMPARE] Skipping: %#v %#v", returnedField, originalField)
				continue
			}

			// Skip any fields we want to ignore
			if shouldSkip(returnedField.Name, originalField, returnedField) {
				continue
			}

			// Get the value of the field and pull value
			// out if the field is a ptr
			originalValue := originalObj.Field(iO)
			if originalValue.Kind() == reflect.Ptr {
				if originalValue.IsNil() {
					log.Printf("[COMPARE] Skipping as is nil ptr: %#v %#v", returnedField, originalField)
					continue
				}
				originalValue = originalValue.Elem()
			}
			returnedValue := returnedObj.Field(iO)
			if returnedValue.Kind() == reflect.Ptr {
				if returnedValue.IsNil() {
					log.Printf("[COMPARE] Skipping as is nil ptr: %#v %#v", returnedField, originalField)
					continue
				}
				returnedValue = returnedValue.Elem()
			}

			log.Printf("[COMPARE] Found matching field: %#v, %#v", returnedField.Name, returnedValue.Type().Kind().String())

			// Recurse into the struct to compare it's fields
			if returnedValue.Type().Kind() == reflect.Struct {
				log.Printf("[COMPARE] Found struct recurrsing: %#v", returnedField)

				err := compareObjsInternal(originalValue, returnedValue, builder)
				if err != nil {
					return err
				}
				continue
			}

			// Skip unneeded fields
			k := returnedValue.Kind()
			switch k {
			case reflect.String:
				if returnedValue.String() == "" {
					log.Printf("[COMPARE] Skipping empty string value: %#v %#v", returnedField, originalField)
					continue
				}
			case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:

				// We can check if these are nil
				if returnedValue.IsNil() {
					log.Printf("[COMPARE] Skipping nil value: %#v %#v", returnedField, originalField)
					continue
				}
			}

			// We can do a more detailed comparison on map fields
			if k == reflect.Map {
				log.Printf("[COMPARE] Comparing map: %#v %#v", returnedField, originalField)

				returnedKeys := returnedValue.MapKeys()
				originalKeys := originalValue.MapKeys()

				for _, oKey := range originalKeys {
					oKeyString := oKey.String()
					oValue := originalValue.MapIndex(oKey)
					for _, rKey := range returnedKeys {
						if oKey.String() == rKey.String() {
							rValue := returnedValue.MapIndex(rKey)
							// rValueKind := rValue.Kind()

							// Skip any fields we want to ignore
							if shouldSkip(oKeyString, oValue, rValue) {
								log.Println(oKeyString)
								continue
							}

							builder.WriteString(fmt.Sprintf("fieldName:%s,fieldValue:%v", rKey.String(), rValue.Interface()))
						}
					}
				}

				return nil
			}

			// We can do a more detailed comparison for arrays too
			if k == reflect.Slice {
				log.Printf("[COMPARE] Comparing slice: %#v %#v", returnedField, originalField)

				oSliceLen := originalValue.Len()
				rSliceLen := returnedValue.Len()
				if rSliceLen < oSliceLen {
					//Todo: what do we do here?
					panic("wrong size")
				}

				for i := 0; i < oSliceLen; i++ {
					log.Printf("[COMPARE] Recurse for Array/slice item: %#v %#v", returnedField, originalField)

					// Handle case in which this is an array of ints or strings NOT structs
					oValueSlice := originalValue.Index(i)
					rValueSlice := returnedValue.Index(i)
					rValueSliceKind := rValueSlice.Kind()
					if rValueSliceKind == reflect.String || rValueSliceKind == reflect.Int {
						builder.WriteString(fmt.Sprintf("fieldName:%s,fieldValue:%v", returnedField.Name+string(i), returnedValue.Interface()))
					} else {
						err := compareObjsInternal(oValueSlice, rValueSlice, builder)
						if err != nil {
							return err
						}
					}
				}

				return nil
			}

			if returnedValue.CanInterface() {
				log.Printf("[COMPARE] Found value writing to string builder: %s->%#v  (%#v)", returnedField.Name, returnedValue.Interface(), returnedValue.Kind().String())
				builder.WriteString(fmt.Sprintf("fieldName:%s,fieldValue:%v", returnedField.Name, returnedValue.Interface()))
			} else {
				log.Printf("[COMPARE] Found unsettable field :(: %#v", returnedField.Name)
			}
		}
	}

	return nil
}
