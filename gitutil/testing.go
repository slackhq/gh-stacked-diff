package gitutil

import "sync"

// ResetCacheForTesting clears all cached values. For use in tests only.
func ResetCacheForTesting() {
	remoteMainBranch = ""
	remoteMainBranchOnce = new(sync.Once)
	localMainBranch = ""
	localMainBranchOnce = new(sync.Once)
	mainBranchNameForHelp = ""
	userEmail = ""
}
