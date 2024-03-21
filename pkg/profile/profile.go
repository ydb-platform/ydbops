package profile

import (
	"fmt"
	"os"
	"unicode"

	"gopkg.in/yaml.v2"
)

type profile map[string]any

type option struct {
	ptr          *string
	defaultValue string
}

var pointersToProgramOptions = make(map[string]option)

func fromCamelToKebabCase(s string) string {
	var result string
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) {
			result += "-"
		}
		result += string(unicode.ToLower(r))
	}
	return result
}

func FillDefaultsFromActiveProfile(profileFile, profileName string) error {
	if profileFile != "" && profileName == "" {
		return fmt.Errorf("specified --profile-path, but unspecified --profile")
	}

	if profileFile == "" && profileName != "" {
		return fmt.Errorf("specified --profile, but unspecified --profile-path")
	}

	if profileFile == "" && profileName == "" {
		return nil
	}

	data := make(map[string]profile)
	fileContent, err := os.ReadFile(profileFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(fileContent, &data)
	if err != nil {
		return err
	}

	// fmt.Printf("%v\n", data)
	// for k, v := range data {
	// 	fmt.Printf("%v %v\n", k, v)
	// }

	profile, ok := data[profileName]
	if !ok {
		return fmt.Errorf("profile %s not found in your profile file", profileName)
	}

	for k, v := range profile {
		optionName := fromCamelToKebabCase(k)
		option := pointersToProgramOptions[optionName]
		if *option.ptr == option.defaultValue {
			*option.ptr = v.(string)
		}
	}

	return nil
}

// func tryLookupInConfig(kebabCaseflagName string, defaultValue any) any {
// 	var flagName = strcase.ToLowerCamel(kebabCaseflagName)

// 	// use the initial default value, provided in DefineFlags function
// 	if _, present := data[flagName]; !present {
// 		return defaultValue
// 	}

// 	// use the initial default value, provided in DefineFlags function
// 	var parsedValue any

// 	switch defaultValue.(type) {
// 	case string:
// 		parsedValue = activeProfile[flagName].(string)
// 	case int:
// 		parsedValue = activeProfile[flagName].(int)
// 	case bool:
// 		parsedValue = activeProfile[flagName].(bool)
// 	case []string:
// 		stringSliceVal := strings.Split(activeProfile[flagName].(string), ",")
// 		parsedValue = stringSliceVal
// 	default:
// 		panic("TODO failed to determine type of the argument")
// 	}

// 	return parsedValue
// }

func PopulateFromProfileLaterP(
	setter func(*string, string, string, string, string),
	ptr *string,
	flagName string,
	shorthand string,
	defaultValue string,
	usage string,
) {
	pointersToProgramOptions[flagName] = option{
		ptr:          ptr,
		defaultValue: defaultValue,
	}
	setter(ptr, flagName, shorthand, defaultValue, usage)
}

func PopulateFromProfileLater(
	setter func(*string, string, string, string),
	ptr *string,
	flagName string,
	defaultValue string,
	usage string,
) {
	pointersToProgramOptions[flagName] = option{
		ptr:          ptr,
		defaultValue: defaultValue,
	}
	setter(ptr, flagName, defaultValue, usage)
}
