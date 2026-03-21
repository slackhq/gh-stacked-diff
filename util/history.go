package util

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type HistoricalData struct {
	filename   string
	maxHistory int
}

/*
maxHistory Max history to store, or -1 for no max.
*/
func NewHistoricalData(filename string, maxHistory int) HistoricalData {
	return HistoricalData{filename: filename, maxHistory: maxHistory}
}

/*
history items are returned as:
[0] least recent
[last element] most recent
*/
func (d HistoricalData) ReadHistory() []string {
	appConfig := GetAppConfig()
	dataBytes, err := os.ReadFile(getHistoryFile(d.filename))
	if err != nil {
		return []string{}
	}
	data := string(dataBytes)
	if appConfig.DemoMode {
		// To support writing history file manuallly when demo'ing on windows.
		return strings.Fields(data)
	} else {
		return strings.Split(data, "\n")
	}
}

// Add a most recently used item to history.
func (d HistoricalData) AddToHistory(newHistoryItem string) {
	history := d.ReadHistory()
	// remove any duplicates
	history = slices.DeleteFunc(history, func(next string) bool {
		return next == newHistoryItem
	})
	history = append(history, newHistoryItem)
	d.SetHistory(history)
}

// Add a most recently used item to history.
func (d HistoricalData) SetHistory(history []string) {
	if d.maxHistory != -1 && len(history) > d.maxHistory {
		history = history[len(history)-d.maxHistory:]
	}
	data := strings.Join(history, "\n")
	if writeErr := os.WriteFile(getHistoryFile(d.filename), []byte(data), os.ModePerm); writeErr != nil {
		panic("Could not write file: " + writeErr.Error())
	}
}

func getHistoryFile(historyFilename string) string {
	appConfig := GetAppConfig()
	appCacheDir := filepath.Join(appConfig.UserCacheDir, "gh-stacked-diff", GetRepoName())
	ExecuteOrDie(ExecuteOptions{}, "mkdir", "-p", appCacheDir)
	return filepath.Join(appCacheDir, historyFilename)
}
