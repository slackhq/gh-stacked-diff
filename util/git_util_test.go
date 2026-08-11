package util

import "testing"

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/slackhq/gh-stacked-diff.git", "slackhq/gh-stacked-diff"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"git@github.com:slackhq/gh-stacked-diff.git", "slackhq/gh-stacked-diff"},
		{"git@github.com:owner/repo", "owner/repo"},
		{"http://github.com/org/project.git", "org/project"},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := repoNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}
