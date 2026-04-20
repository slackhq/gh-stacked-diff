package util

import (
	_ "embed"
	"strings"
)

//go:embed stable_version.txt
var stableVersion string

//go:embed current_version.txt
var CurrentVersion string

// VersionSuffix returns " (stable)" or " (preview)" based on whether CurrentVersion matches stableVersion.
func VersionSuffix() string {
	if strings.TrimSpace(CurrentVersion) == strings.TrimSpace(stableVersion) {
		return " (stable)"
	}
	return " (preview)"
}
