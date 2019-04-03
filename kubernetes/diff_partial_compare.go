package kubernetes

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/conversion"
)

func compareObjs(original, returned interface{}, builder *strings.Builder) error {
	// Check originalObj is valid
	originalObj, err := conversion.EnforcePtr(original)
	if err != nil {
		return err
	}

	// Check returnedObj is valid
	returnedObj, err := conversion.EnforcePtr(returned)
	if err != nil {
		return err
	}
	return compareObjsInternal(originalObj, returnedObj, builder)
}

var skipFields = map[string]bool{
	"Status":            true,
	"Finalizers":        true,
	"Initializers":      true,
	"OwnerReferences":   true,
	"CreationTimestamp": true,
	"Generation":        true,
	"ResourceVersion":   true,
	"resourceVersion":   true,
	"creationTimestamp": true,
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
