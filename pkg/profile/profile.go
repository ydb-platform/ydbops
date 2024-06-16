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

const (
	activeProfileKeyName = "active_profile"
	profileArrayKeyName  = "profiles"
)

func FillDefaultsFromActiveProfile(configFile, profileName string) error {
	if configFile == "" && profileName == "" {
		return nil
	}

	if configFile == "" && profileName != "" {
		return fmt.Errorf("specified --profile, but unspecified --config-path")
	}

	data := make(map[string]any)
	fileContent, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("failed to read the config file on path %s: %w", configFile, err)
	}

	err = yaml.Unmarshal(fileContent, &data)
	if err != nil {
		return err
	}

	if currentProfile, present := data[activeProfileKeyName]; profileName == "" && present {
		profileName = currentProfile.(string)
		delete(data, activeProfileKeyName)
	} else if profileName == "" {
		return fmt.Errorf(
			"failed to get current profile: field `%s` absent in config, and --profile flag unspecified",
			activeProfileKeyName,
		)
	}

	profilesArray, ok := data[profileArrayKeyName]
	if !ok {
		return fmt.Errorf("config file is malformed, `%s` field not found", profileArrayKeyName)
	}

	profile, ok := profilesArray.(map[any]any)[profileName]
	if !ok {
		return fmt.Errorf("profile `%s` not found in your profile file", profileName)
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
