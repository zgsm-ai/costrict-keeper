package utils

import (
	"strconv"
	"strings"
)

/**
 * Parse version string into VersionNumber struct
 * @param {string} versionStr - Version string in "major.minor.micro" format (e.g. "1.2.3")
 * @returns {*VersionNumber} Pointer to VersionNumber struct if parse succeeds, nil on failure
 * @description
 * - Splits version string by dots and converts to integers
 * - Returns nil if input format is invalid or contains non-numeric parts
 * @example
 * ver := ParseVersionNumber("1.2.3")  // returns VersionNumber{Major:1, Minor:2, Micro:3}
 * ver := ParseVersionNumber("invalid") // returns nil
 */
func ParseVersionNumber(versionStr string) *VersionNumber {
	vers := strings.Split(versionStr, ".")
	if len(vers) != 3 {
		return nil
	}

	var ver VersionNumber
	var err error
	ver.Major, err = strconv.Atoi(vers[0])
	if err != nil {
		return nil
	}
	ver.Minor, err = strconv.Atoi(vers[1])
	if err != nil {
		return nil
	}
	ver.Micro, err = strconv.Atoi(vers[2])
	if err != nil {
		return nil
	}
	return &ver
}
