package gitutil

import (
	"log/slog"
	"testing"

	"github.com/slackhq/gh-stacked-diff/v2/testutil"
	"github.com/stretchr/testify/assert"
)

func TestGetMinChecksFromHistory_WhenHistorySet_ReturnsCachedValue(t *testing.T) {
	testutil.InitTest(t, slog.LevelError)
	assert := assert.New(t)

	setMinChecksToHistory(3)

	result := getMinChecksFromHistory()
	assert.Equal(3, result,
		"getMinChecksFromHistory should return the cached value")
}
