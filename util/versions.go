package util

import (
	_ "embed"
)

//go:embed stable_version.txt
var StableVersion string

//go:embed current_version.txt
var CurrentVersion string
