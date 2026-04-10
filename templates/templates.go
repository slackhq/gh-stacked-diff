package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/slackhq/gh-stacked-diff/v2/gitutil"
	"github.com/slackhq/gh-stacked-diff/v2/util"
)

//go:embed config/branch_name.template
var branchNameTemplateText string

//go:embed config/pr_title.template
var prTitleTemplateText string

//go:embed config/pr_description.template
var prDescriptionTemplateText string

type PullRequestText struct {
	Title       string
	Description string
}

type branchTemplateData struct {
	UsernameCleaned      string
	CommitSummaryCleaned string
}

type templateData struct {
	TicketNumber               string
	TicketUrlPattern           string
	Username                   string
	CommitBody                 string
	CommitSummary              string
	CommitSummaryCleaned       string
	CommitSummaryWithoutTicket string
	FeatureFlag                string
}

// Enum for what commitIndicator represents.
type IndicatorType string

const (
	// commitIndicator is a commit hash.
	IndicatorTypeCommit IndicatorType = "commit"
	// commitIndicator is a PR number.
	IndicatorTypePr IndicatorType = "pr"
	// commitIndicator is a list index from log (1 based).
	IndicatorTypeList IndicatorType = "list"
	// Guess based on length of commitIndicator and whether it is all numeric.
	IndicatorTypeGuess IndicatorType = "guess"
)

// Returns weather the indicator type is of a known type.
func (indicator IndicatorType) IsValid() bool {
	switch indicator {
	case IndicatorTypeCommit, IndicatorTypePr, IndicatorTypeList, IndicatorTypeGuess:
		return true
	default:
		return false
	}
}

// Returns BranchInfo for commitIndicator and indicatorType.
func GetBranchInfo(commitIndicator string, indicatorType IndicatorType) GitLog {
	util.RequireNotEmptyString(commitIndicator)
	if !indicatorType.IsValid() {
		panic("Invalid IndicatorType " + string(indicatorType))
	}
	if indicatorType == IndicatorTypeGuess {
		indicatorType = guessIndicatorType(commitIndicator)
	}
	var info GitLog
	switch indicatorType {
	case IndicatorTypePr:
		slog.Debug("Using commitIndicator as a pull request number " + commitIndicator)

		branchName := util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: gitutil.GhRetries}, "gh", "pr", "view", commitIndicator, "--json", "headRefName", "-q", ".headRefName")
		util.RequireGitRef(branchName)
		// Fetch the branch in case the lastest commit is only on GitHub.
		util.ExecuteOrDie(util.ExecuteOptions{}, "git", "fetch", "origin", branchName)
		// Get the first commit of the branch on Github.
		prCommit := util.ExecuteOrDieTrimmed(util.ExecuteOptions{Retries: gitutil.GhRetries}, "gh", "pr", "view", commitIndicator, "--json", "commits", "-q", "[.commits[].oid] | first")
		util.RequireHexString(prCommit)
		gitLogs := newGitLogs(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", "--no-patch", newGitLogsFormat, "--abbrev-commit", prCommit))
		if len(gitLogs) == 0 {
			panic(fmt.Sprint("Could not find first commit (", prCommit, ") of PR ", commitIndicator))
		}
		info = gitLogs[0]
		// Set the branch name in case it differs because the PR was created manually.
		info.Branch = branchName
		slog.Info("Using pull request " + commitIndicator + ", commit " + info.Commit + ", branch " + info.Branch)
	case IndicatorTypeCommit:
		slog.Debug("Using commitIndicator as a commit hash " + commitIndicator)
		gitLogs := newGitLogs(util.ExecuteOrDie(util.ExecuteOptions{}, "git", "show", "--no-patch", newGitLogsFormat, "--abbrev-commit", commitIndicator))
		if len(gitLogs) == 0 {
			panic(fmt.Sprint("Could not find commit ", commitIndicator))
		}
		info = gitLogs[0]
		slog.Info("Using commit " + info.Commit + ", branch " + info.Branch)
	case IndicatorTypeList:
		slog.Debug("Using commitIndicator as a list index " + commitIndicator)
		newCommits := GetNewCommits(util.GetCurrentBranchName(), "")
		listIndex, err := strconv.Atoi(commitIndicator)
		if err != nil {
			panic("When indicator type is " + string(IndicatorTypeList) + " commit indicator must be a number, given " + commitIndicator)
		}
		// list indicators are 1 based, convert to 0 based.
		listIndex--
		if listIndex >= len(newCommits) || listIndex < 0 {
			panic("list index " + fmt.Sprint(listIndex) +
				" (parsed from " + commitIndicator + ") " +
				"out of bounds for list of new commits with size " +
				fmt.Sprint(len(newCommits)))
		}
		slog.Info("Using list index " + commitIndicator + ", commit " + newCommits[listIndex].Commit + " " + newCommits[listIndex].Subject)
		info = newCommits[listIndex]
	default:
		panic("Impossible: guessIndicatorType only returns known values, " + fmt.Sprint(indicatorType))
	}
	return info
}

func guessIndicatorType(commitIndicator string) IndicatorType {
	if _, err := strconv.Atoi(commitIndicator); err == nil {
		if len(commitIndicator) < 3 {
			return IndicatorTypeList
		}
		if len(commitIndicator) < 7 {
			return IndicatorTypePr
		}
	}
	if strings.ContainsFunc(strings.ToUpper(commitIndicator), func(r rune) bool {
		return r < '0' || (r > '9' && r < 'A') || r > 'F'
	}) {
		panic("Invalid commit indicator: " + commitIndicator)
	}
	return IndicatorTypeCommit
}

func getBranchForSantizedSubject(sanitizedSubject string) string {
	name := RunTemplate("branch-name.template", branchNameTemplateText, getBranchTemplateData(sanitizedSubject))
	// Branch names that are too long cause problems with Github.
	name = truncateString(name, 120)
	return name
}

func truncateString(str string, maxBytes int) string {
	for i := range str {
		if i >= maxBytes {
			return str[:i]
		}
	}
	return str
}

func GetPullRequestText(commitHash string, featureFlag string, ticketUrlPattern string) PullRequestText {
	data := getPullRequestTemplateData(commitHash, featureFlag, ticketUrlPattern)
	title := RunTemplate("pr-title.template", prTitleTemplateText, data)
	description := RunTemplate("pr-description.template", prDescriptionTemplateText, data)
	return PullRequestText{Description: description, Title: title}
}

func RunTemplate(configFilename string, defaultTemplateText string, data any) string {
	configFile := util.GetConfigFile(configFilename)
	var parsed *template.Template
	var err error
	if configFile != "" {
		parsed, err = template.ParseFiles(configFile)
		if err != nil {
			panic(fmt.Sprint("Could not parse ", configFile, ": ", err))
		}
	} else {
		parsed, err = template.New("").Parse(defaultTemplateText)
		if err != nil {
			panic(fmt.Sprint("Could not parse ", defaultTemplateText, ": ", err))
		}
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		panic(err)
	}
	return output.String()
}

func getPullRequestTemplateData(commitHash string, featureFlag string, ticketUrlPattern string) templateData {
	commitSummary := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "--no-pager", "show", "--no-patch", "--format=%s", commitHash)
	commitBody := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "--no-pager", "show", "--no-patch", "--format=%b", commitHash)
	commentLineRegex := regexp.MustCompile("(?m)^#.*$")
	commitBody = commentLineRegex.ReplaceAllString(commitBody, "")
	commitSummaryCleaned := util.ExecuteOrDieTrimmed(util.ExecuteOptions{}, "git", "show", "--no-patch", "--format=%f", commitHash)
	summaryMatches := ticketNumberRegex.FindStringSubmatch(commitSummary)
	ticketNumber := strings.TrimSpace(summaryMatches[1])
	resolvedTicketUrl := strings.ReplaceAll(ticketUrlPattern, "{TicketNumber}", ticketNumber)
	return templateData{
		Username:                   gitutil.GetUsername(),
		TicketNumber:               ticketNumber,
		TicketUrlPattern:           resolvedTicketUrl,
		CommitBody:                 commitBody,
		CommitSummary:              commitSummary,
		CommitSummaryWithoutTicket: summaryMatches[2],
		CommitSummaryCleaned:       commitSummaryCleaned,
		FeatureFlag:                featureFlag,
	}
}

var ticketNumberRegex = regexp.MustCompile(`^(\S+-[[:digit:]]+ )?(.*)`)

// HasTicketNumber returns true if the commit summary starts with a
// Jira-style ticket number (e.g. "CONV-9999 Add feature").
func HasTicketNumber(commitSummary string) bool {
	matches := ticketNumberRegex.FindStringSubmatch(commitSummary)
	return len(matches) > 1 && strings.TrimSpace(matches[1]) != ""
}

// TemplateUsesTicketUrlPattern returns true if the PR description or title
// template references the TicketUrlPattern variable.
func TemplateUsesTicketUrlPattern() bool {
	return templateTextContains("pr-description.template", prDescriptionTemplateText, "TicketUrlPattern") ||
		templateTextContains("pr-title.template", prTitleTemplateText, "TicketUrlPattern")
}

func templateTextContains(configFilename string, defaultText string, search string) bool {
	configFile := util.GetConfigFile(configFilename)
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			panic(fmt.Sprint("Could not read ", configFile, ": ", err))
		}
		return strings.Contains(string(data), search)
	}
	return strings.Contains(defaultText, search)
}

func getBranchTemplateData(sanitizedSummary string) branchTemplateData {
	// Dots are not allowed in branch names of some Github configurations.
	username := strings.ReplaceAll(gitutil.GetUsername(), ".", "-")
	return branchTemplateData{
		UsernameCleaned:      username,
		CommitSummaryCleaned: sanitizedSummary,
	}
}
