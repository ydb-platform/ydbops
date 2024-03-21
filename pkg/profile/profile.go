package profile

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type option struct {
	ptr          *string
	defaultValue string
}

// TODO make a generic solution, at this moment only string values are supported
var pointersToProgramOptions = make(map[string]option)

func FillDefaultsFromActiveProfile(profileFile, profileName string) error {
	if profileFile == "" && profileName == "" {
		return nil
	}

	if profileFile == "" && profileName != "" {
		return fmt.Errorf("specified --profile, but unspecified --profile-path")
	}

	data := make(map[string]any)
	fileContent, err := os.ReadFile(profileFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(fileContent, &data)
	if err != nil {
		return err
	}

	if currentProfile, present := data["current-profile"]; profileName == "" && present {
		profileName = currentProfile.(string)
		delete(data, "current-profile")
	} else if profileName == "" {
		return fmt.Errorf("failed to get current profile: field `current-profile` absent in config, and --profile flag unspecified")
	}

	profile, ok := data[profileName]
	if !ok {
		return fmt.Errorf("profile %s not found in your profile file", profileName)
	}

	for optionName, defValue := range profile.(map[any]any) {
		option := pointersToProgramOptions[optionName.(string)]
		if *option.ptr == option.defaultValue {
			*option.ptr = defValue.(string)
		}
	}

	return nil
}

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
