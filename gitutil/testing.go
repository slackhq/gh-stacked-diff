package gitutil

import (
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/util"
)

func init() {
	util.RegisterTestInitHook(func(_ *testing.T) {
		ResetCacheForTesting()
	})
}

// ResetCacheForTesting clears all cached values. For use in tests only.
func ResetCacheForTesting() {
	cache = &gitCache{}
}
