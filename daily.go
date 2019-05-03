package main

import (
	"bytes"
	"fmt"
	"github.com/google/go-github/github"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var regexRepo = regexp.MustCompile("github\\.com\\/([^\\/]+\\/[^\\/]+)\\/")

func newDailyCommand() *cobra.Command {
	m := &cobra.Command{
		Use:   "daily",
		Short: "Daily Report",
		Args:  cobra.MinimumNArgs(0),
		Run:   runDailyCommandFunc,
	}

	return m
}

type GithubItem struct {
	issue    github.Issue
	memtions []string
}

// collectMentionsPR merge the memtion members.
func collectMentionsPR(collector map[string]*GithubItem, member string, issues []github.Issue) {
	for _, issue := range issues {
		link := issue.GetHTMLURL()
		githubItem, ok := collector[link]
		if !ok {
			item := &GithubItem{
				issue:    issue,
				memtions: []string{member},
			}
			collector[link] = item
			continue
		}

		githubItem.memtions = append(githubItem.memtions, member)
	}
}

func runDailyCommandFunc(cmd *cobra.Command, args []string) {
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour).Format(githubUTCDateFormat)

	var buf bytes.Buffer
	buf.WriteString("*Daily Report*\n\n")

	//issues := getCreatedIssues(start, nil)
	//formatSectionForSlackOutput(&buf, "New Issues", "New issues in last 24 hours")
	//formatGitHubIssuesForSlackOutput(&buf, issues)
	//buf.WriteString("\n")

	collector := make(map[string]*GithubItem)
	for _, member := range allSQLInfraMembers {
		mentionedPRs := getPullReuestsMentioned(start, nil, member)
		collectMentionsPR(collector, member, mentionedPRs)
	}

	formatSectionForSlackOutput(&buf, fmt.Sprintf("Pull Requests that mentioned you"), "PR that mentioned you in last 24 hours")
	formatCollectMentionsPRForSlackOutput(&buf, collector)
	buf.WriteString("\n")

	members := strings.Join(allSQLInfraMemberEmals, ",")
	dailyIssues := queryJiraIssues(fmt.Sprintf(`assignee in (%v)  AND updated >= -1d ORDER BY assignee`, members))
	formatSectionForSlackOutput(&buf, "Team JIRA Issue", "Updated in last 24 hours")
	formatJiraIssuesForSlackOutput(&buf, dailyIssues)
	buf.WriteString("\n")

	// TODO: make the filter syntax configurable in the config file.
	// nonProcessStatus := `"Job Closed", 完成, TODO, "To Do", DUPLICATED, Blocked, Closed, Paused, Resolved, "CAN'T REPRODUCE", Cancelled, "WON'T FIX"`
	nonProcessStatus := config.Jira.NonProcessStatus
	dueDateIssues := queryJiraIssues(fmt.Sprintf(`status not in (%v) AND assignee in (%v) and duedate <= 2d  ORDER BY assignee`, nonProcessStatus, members))
	formatSectionForSlackOutput(&buf, "Getting To Due Date JIRA Issue", "The due date will be less than 2 day")
	formatJiraIssuesForSlackOutput(&buf, dueDateIssues)
	buf.WriteString("\n")

	//processingIssues := queryJiraIssues(fmt.Sprintf(`status not in (%v) AND assignee in (%v) ORDER BY assignee`, nonProcessStatus, members))
	//formatSectionForSlackOutput(&buf, "JIRA Issue Without Due Date", "Please add due date to processing JIRA issues")
	//formatJiraIssuesForSlackOutput(&buf, findOutIssuesWithoutDueDate(processingIssues))
	//buf.WriteString("\n")

	if printToConsole {
		fmt.Println(buf.String())
	} else {
		sendToSlack(buf.String())
	}
}
