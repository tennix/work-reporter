package main

import (
	"bytes"
	"fmt"
	"regexp"
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

func runDailyCommandFunc(cmd *cobra.Command, args []string) {
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour).Format(githubUTCDateFormat)

	var buf bytes.Buffer
	buf.WriteString("*Daily Report*\n\n")

	issues := getCreatedIssues(start, nil)
	formatSectionForSlackOutput(&buf, "New Issues", "New issues in last 24 hours")
	formatGitHubIssuesForSlackOutput(&buf, issues)
	buf.WriteString("\n")

	//issues = getCreatedPullRequests(start, nil)
	//formatSectionForSlackOutput(&buf, "New Pull Requests", "New PRs in last 24 hours")
	//formatGitHubIssuesForSlackOutput(&buf, issues)
	//buf.WriteString("\n")

	for _, member := range allMembers {
		issues = getPullReuestsMentioned(start, nil, member)
		formatSectionForSlackOutput(&buf, fmt.Sprintf("Pull Requests that mentioned you @%v", member), "PR that mentioned you in last 24 hours")
		formatGitHubIssuesForSlackOutput(&buf, issues)
		buf.WriteString("\n")
	}


	dailyIssues := queryJiraIssues(`status not in ("Job Closed", 完成, DUPLICATED, Blocked, Closed, "WON'T FIX", Paused, Resolved) AND assignee in ("wink@pingcap.com", "xiaoliangliang@pingcap.com", chenshuang, "lixia@pingcap.com")  AND updated >= -1d ORDER BY updated`)
	formatSectionForSlackOutput(&buf, "Team JIRA Issue", "Updated in last 24 hours")
	formatJiraIssuesForSlackOutput(&buf, dailyIssues)
	buf.WriteString("\n")

	dailyIssues = queryJiraIssues(`assignee in ("longheng@pingcap.com","wink@pingcap.com","xiaoliangliang@pingcap.com","lixia@pingcap.com","chenshuang@pingcap.com") AND created  >= -1d`)
	formatSectionForSlackOutput(&buf, "New JIRA Issues", "JIRA issues created in last 24 hours")
	formatJiraIssuesForSlackOutput(&buf, dailyIssues)
	//buf.WriteString("\n")

	sendToSlack(buf.String())
	//fmt.Println(buf.String())
}
