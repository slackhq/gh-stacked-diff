package util

import (
	_ "embed"
	"strings"
)

//go:embed stable_version.txt
var StableVersion string

//go:embed current_version.txt
var CurrentVersion string

// VersionSuffix returns " (stable)" or " (preview)" based on whether CurrentVersion matches StableVersion.
func VersionSuffix() string {
	if strings.TrimSpace(CurrentVersion) == strings.TrimSpace(StableVersion) {
		return " (stable)"
	}
	return " (preview)"
}
