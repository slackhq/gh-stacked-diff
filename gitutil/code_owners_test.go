package gitutil

import (
	"slices"
	"testing"
)

func TestParseCodeOwnersTeams_OnlyReturnsTeams(t *testing.T) {
	content := "# comment @org/ignored\n" +
		"* @org/team-b @user user@example.com\n" +
		"/docs/ @org/team-a # trailing @org/ignored\n" +
		"*.go @org/team-b\n"
	teams := parseCodeOwnersTeams(content)
	expected := []string{"org/team-a", "org/team-b"}
	if !slices.Equal(teams, expected) {
		t.Errorf("expected %v, got %v", expected, teams)
	}
}
