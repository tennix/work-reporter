package main

import (
	"bytes"
	"fmt"
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

func runDailyCommandFunc(cmd *cobra.Command, args []string) {
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour).Format(githubUTCDateFormat)

	var buf bytes.Buffer
	buf.WriteString("*Daily Report*\n\n")

	issues := getCreatedIssues(start, nil)
	formatSectionForSlackOutput(&buf, "New Issues", "New issues in last 24 hours")
	formatGitHubIssuesForSlackOutput(&buf, issues)
	buf.WriteString("\n")

	for _, member := range allMembers {
		issues = getPullReuestsMentioned(start, nil, member)
		formatSectionForSlackOutput(&buf, fmt.Sprintf("Pull Requests that mentioned you @%v", member), "PR that mentioned you in last 24 hours")
		formatGitHubIssuesForSlackOutput(&buf, issues)
		buf.WriteString("\n")
	}


	members := strings.Join(allMemberEmals, ",")
	dailyIssues := queryJiraIssues(fmt.Sprintf(`assignee in (%v)  AND updated >= -1d ORDER BY updated`, members))
	formatSectionForSlackOutput(&buf, "Team JIRA Issue", "Updated in last 24 hours")
	formatJiraIssuesForSlackOutput(&buf, dailyIssues)
	buf.WriteString("\n")

	processingStatus := `"Job Closed", 完成, TODO, "To Do", DUPLICATED, Blocked, Closed, "WON'T FIX", Paused, Resolved`
	dueDateIssues := queryJiraIssues(fmt.Sprintf(`status not in (%v) AND assignee in (%v) and duedate <= 2d  ORDER BY updated`, processingStatus, members))
	formatSectionForSlackOutput(&buf, "Going To Due Date JIRA Issue", "The due date will be less than 2 day")
	formatJiraIssuesForSlackOutput(&buf, dueDateIssues)
	buf.WriteString("\n")

	processingIssues := queryJiraIssues(fmt.Sprintf(`status not in (%v) AND assignee in (%v) ORDER BY updated`, processingStatus, members))
	formatSectionForSlackOutput(&buf, "JIRA Issue Without Due Date", "Please add due date to processing JIRA issues")
	formatJiraIssuesForSlackOutput(&buf, findOutIssuesWithoutDueDate(processingIssues))
	buf.WriteString("\n")

	sendToSlack(buf.String())
	//fmt.Println(buf.String())
}
