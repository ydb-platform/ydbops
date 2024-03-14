package blackmagic

import (
	"reflect"

	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/proto"
)

func populateFieldMapFor(tp reflect.Type, value reflect.Value) map[string]reflect.Value {
	fields := make(map[string]reflect.Value)
	for i := 0; i < tp.Elem().NumField(); i++ {
		fields[tp.Elem().Field(i).Name] = value.Elem().Field(i)
	}
	return fields
}

// NOTE this is just an experiment and will be probably gone soon! 
// Please don't judge this code yet. This is an experimental matcher to compare two proto messages 
// with a couple nuances, required for comparing e2e test output with an expected, hand-constructed 
// output, type-safely.
func ExpectPresentFieldsDeepEqual(expected proto.Message, actual proto.Message, values map[string]string) {
	expectedType := reflect.TypeOf(expected)
	actualType := reflect.TypeOf(actual)


	Expect(expectedType).To(Equal(actualType))

	expectedFieldValues := populateFieldMapFor(expectedType, reflect.ValueOf(expected))
	actualFieldValues := populateFieldMapFor(actualType, reflect.ValueOf(actual))

	for structFieldName, expectedFieldValue := range expectedFieldValues {
		if expectedFieldValue.IsValid() && expectedFieldValue.CanSet() {
			actualFieldValue := actualFieldValues[structFieldName]

			if expectedFieldValue.Kind() == reflect.String {
				Expect(actualFieldValue.Kind()).To(Equal(reflect.String))
				expectedFieldString := expectedFieldValue.Interface().(string)
				actualFieldString := actualFieldValue.Interface().(string)
				if _, present := values[expectedFieldString]; !present {
					values[expectedFieldString] = actualFieldString
				}
				Expect(values[expectedFieldString]).To(Equal(actualFieldString))
			}

			if expectedFieldValue.Kind() == reflect.Struct && expectedFieldValue.Interface().(proto.Message) != nil {
				ExpectPresentFieldsDeepEqual(expectedFieldValue.Interface().(proto.Message), actualFieldValue.Interface().(proto.Message), values)
			}

			if expectedFieldValue.Kind() == reflect.Slice {
				Expect(actualFieldValue.Kind()).To(Equal(reflect.Slice))
				Expect(expectedFieldValue.Len()).To(Equal(actualFieldValue.Len()))
				for i := 0; i < expectedFieldValue.Len(); i++ {
					expectedElement := expectedFieldValue.Index(i)
					actualElement := actualFieldValue.Index(i)
					ExpectPresentFieldsDeepEqual(expectedElement.Interface().(proto.Message), actualElement.Interface().(proto.Message), values)
				}
			}
		}
	}
}
