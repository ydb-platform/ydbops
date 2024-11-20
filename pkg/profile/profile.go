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

// TODO @jorres make a generic solution, at this moment only string values are supported

// NOTE: the whole profile solution is probably a hack, and there must be a better way of doing it.
// Currently, this map, for every key that is supported by a profile (e.g. 'kubeconfig'), holds
// pointers to all string variables for this key across all commands (e.g. for 'kubeconfig' profile
// option, there are multiple commands which have RestartOptions and thus need filling
// KubeconfigPath in profile: run, maintenance, restart).
var pointersToProgramOptions = make(map[string][]option)

const (
	activeProfileKeyName = "current-profile"
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

	for optionName, valueFromFile := range profile.(map[any]any) {
		options, ok := pointersToProgramOptions[optionName.(string)]
		if !ok {
			return fmt.Errorf("profile `%s` contains unsupported field `%s`", profileName, optionName)
		}
		for _, option := range options {
			if *option.ptr == option.defaultValue {
				*option.ptr = valueFromFile.(string)
			}
		}
	}

	return nil
}

func appendToOptions(flagName string, ptr *string, defaultValue string) {
	if pointersToProgramOptions[flagName] == nil {
		pointersToProgramOptions[flagName] = []option{}
	}

	pointersToProgramOptions[flagName] = append(pointersToProgramOptions[flagName], option{
		ptr:          ptr,
		defaultValue: defaultValue,
	})
}

func PopulateFromProfileLaterP(
	setter func(*string, string, string, string, string),
	ptr *string,
	flagName string,
	shorthand string,
	defaultValue string,
	usage string,
) {
	appendToOptions(flagName, ptr, defaultValue)
	setter(ptr, flagName, shorthand, defaultValue, usage)
}

func PopulateFromProfileLater(
	setter func(*string, string, string, string),
	ptr *string,
	flagName string,
	defaultValue string,
	usage string,
) {
	appendToOptions(flagName, ptr, defaultValue)
	setter(ptr, flagName, defaultValue, usage)
}
