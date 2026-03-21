package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMinChecksFromHistory_WhenHistorySet_ReturnsCachedValue(t *testing.T) {
	assert := assert.New(t)

	tmpDir := t.TempDir()

	// Save and restore global state.
	oldRepoName := repoName
	oldCachedMinChecks := cachedMinChecks
	oldExecutor := globalExecutor
	oldAppConfig := GetAppConfig()
	defer func() {
		repoName = oldRepoName
		cachedMinChecks = oldCachedMinChecks
		globalExecutor = oldExecutor
		SetAppConfig(oldAppConfig)
	}()

	globalExecutor = DefaultExecutor{}
	repoName = "test-repo"

	SetAppConfig(AppConfig{
		UserCacheDir: tmpDir,
	})

	// Write min checks = 3 to history via setMinChecksToHistory.
	cachedMinChecks = 3
	setMinChecksToHistory(3)

	result := getMinChecksFromHistory()
	assert.Equal(3, result,
		"getMinChecksFromHistory should return the cached value")
}
