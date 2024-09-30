package options

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var AvailabilityModes = []string{"strong", "weak", "force"}

type StartedTime struct {
	Timestamp time.Time
	Direction rune
}

type MajorMinorPatchVersion struct {
	Sign  string
	Major int
	Minor int
	Patch int
}

type RawVersion struct {
	Sign string
	Raw  string
}

type VersionSpec interface {
	Satisfies(otherVersion string) (bool, error)
	String() string
}

func compareMajorMinorPatch(sign string, nodeVersion, userVersion [3]int) bool {
	res := 0
	for i := 0; i < 3; i++ {
		if nodeVersion[i] < userVersion[i] {
			res = -1
			break
		} else if nodeVersion[i] > userVersion[i] {
			res = 1
			break
		}
	}

	switch sign {
	case "==":
		return res == 0
	case "<":
		return res == -1
	case ">":
		return res == 1
	case "!=":
		return res != 0
	}
	return false
}

func compareRaw(sign string, nodeVersion, userVersion string) bool {
	switch sign {
	case "==":
		return nodeVersion == userVersion
	case "!=":
		return nodeVersion != userVersion
	}
	return false
}

func tryParseWith(reString, version string) (int, int, int, bool) {
	re := regexp.MustCompile(reString)
	matches := re.FindStringSubmatch(version)
	if len(matches) == 4 {
		num1, _ := strconv.Atoi(matches[1])
		num2, _ := strconv.Atoi(matches[2])
		num3, _ := strconv.Atoi(matches[3])
		return num1, num2, num3, true
	}
	return 0, 0, 0, false
}

func parseNodeVersion(version string) (int, int, int, error) {
	pattern1 := `^ydb-stable-(\d+)-(\d+)-(\d+).*$`
	major, minor, patch, parsed := tryParseWith(pattern1, version)
	if parsed {
		return major, minor, patch, nil
	}

	pattern2 := `^(\d+)\.(\d+)\.(\d+).*$`
	major, minor, patch, parsed = tryParseWith(pattern2, version)
	if parsed {
		return major, minor, patch, nil
	}

	return 0, 0, 0, fmt.Errorf("failed to parse the version number in any of the known patterns")
}

func (v MajorMinorPatchVersion) Satisfies(otherVersion string) (bool, error) {
	major, minor, patch, err := parseNodeVersion(otherVersion)
	if err != nil {
		return false, fmt.Errorf("Failed to extract major.minor.patch from version %s", otherVersion)
	}

	return compareMajorMinorPatch(
		v.Sign,
		[3]int{major, minor, patch},
		[3]int{v.Major, v.Minor, v.Patch},
	), nil
}

func (v MajorMinorPatchVersion) String() string {
	return fmt.Sprintf("%s%v.%v.%v", v.Sign, v.Major, v.Minor, v.Patch)
}

func (v RawVersion) Satisfies(otherVersion string) (bool, error) {
	return compareRaw(
		v.Sign,
		otherVersion,
		v.Raw,
	), nil
}

func (v RawVersion) String() string {
	return fmt.Sprintf("%s%v", v.Sign, v.Raw)
}
