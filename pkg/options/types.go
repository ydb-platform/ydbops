package options

import (
	"fmt"
	"time"

	"github.com/ydb-platform/ydbops/pkg/utils"
)

const (
	equalSign       = "=="
	notEqualSign    = "!="
	lessThanSign    = "<"
	greaterThanSign = ">"
)

var AvailabilityModes = []string{"strong", "weak", "force", "smart"}

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
	case equalSign:
		return res == 0
	case lessThanSign:
		return res == -1
	case greaterThanSign:
		return res == 1
	case notEqualSign:
		return res != 0
	}
	return false
}

func compareRaw(sign, nodeVersion, userVersion string) bool {
	switch sign {
	case equalSign:
		return nodeVersion == userVersion
	case notEqualSign:
		return nodeVersion != userVersion
	}
	return false
}

func (v MajorMinorPatchVersion) Satisfies(otherVersion string) (bool, error) {
	major, minor, patch, err := utils.ParseMajorMinorPatchFromVersion(otherVersion)
	if err != nil {
		return false, fmt.Errorf("failed to extract major.minor.patch from version %s", otherVersion)
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
