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
	defer func() {
		repoName = oldRepoName
		cachedMinChecks = oldCachedMinChecks
		globalExecutor = oldExecutor
	}()

	globalExecutor = DefaultExecutor{}
	repoName = "test-repo"

	appConfig := AppConfig{
		UserCacheDir: tmpDir,
	}

	// Write min checks = 3 to history via setMinChecksToHistory.
	cachedMinChecks = 3
	setMinChecksToHistory(appConfig, 3)

	result := getMinChecksFromHistory(appConfig)
	assert.Equal(3, result,
		"getMinChecksFromHistory should return the cached value")
}
