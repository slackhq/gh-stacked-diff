package gitutil

import (
	"os"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/util"
	"github.com/stretchr/testify/assert"
)

func TestGetMinChecksFromHistory_WhenHistorySet_ReturnsCachedValue(t *testing.T) {
	assert := assert.New(t)

	tmpDir := t.TempDir()

	// Save and restore global state.
	oldCachedMinChecks := cachedMinChecks
	oldExecutor := util.GetGlobalExecutor()
	oldAppConfig := util.GetAppConfig()
	defer func() {
		cachedMinChecks = oldCachedMinChecks
		util.SetGlobalExecutor(oldExecutor)
		util.SetAppConfig(oldAppConfig)
	}()

	util.SetGlobalExecutor(util.DefaultExecutor{})
	util.SetAppConfig(util.AppConfig{
		UserCacheDir: tmpDir,
	})
	// Init a git repo so GetRepoName() works (called by history file path).
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	util.ExecuteOrDie(util.ExecuteOptions{}, "git", "init")

	// Write min checks = 3 to history via setMinChecksToHistory.
	cachedMinChecks = 3
	setMinChecksToHistory(3)

	result := getMinChecksFromHistory()
	assert.Equal(3, result,
		"getMinChecksFromHistory should return the cached value")
}
