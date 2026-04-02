package gitutil

import (
	"sync"
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
	remoteMainBranch = ""
	remoteMainBranchOnce = new(sync.Once)
	localMainBranch = ""
	localMainBranchOnce = new(sync.Once)
	mainBranchNameForHelp = ""
	userEmail = ""
}
