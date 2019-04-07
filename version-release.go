package main

import (
	"bytes"
	"fmt"
	jira "github.com/andygrunwald/go-jira"
	"github.com/spf13/cobra"
	"strings"
)

func newVersionReleaseCommand() *cobra.Command {
	m := &cobra.Command{
		Use:   "release",
		Short: "Create Version Release",
	}

	m.AddCommand(newVersionReleaseReportCommand())
	m.AddCommand(newVersionReleaseLinkCommand())
	return m
}

func newVersionReleaseReportCommand() *cobra.Command {
	m := &cobra.Command{
		Use:   "report",
		Short: "Create Release Version Report",
		Run:   runVersionReleaseReportCommandFunc,
	}
	return m
}

func newVersionReleaseLinkCommand() *cobra.Command {
	m := &cobra.Command{
		Use:   "link",
		Short: "Create Release Version Report",
		Run:   runVersionReleaseLinkCommandFunc,
	}
	return m
}

func runVersionReleaseReportCommandFunc(cmd *cobra.Command, args []string) {
	// create version release report
	var pageBody bytes.Buffer

	formatPageBeginForHtmlOutput(&pageBody)
	genVersionReleaseReportHtml(&pageBody)
	formatPageEndForHtmlOutput(&pageBody)

	fmt.Println(pageBody.String())
}

func runVersionReleaseLinkCommandFunc(cmd *cobra.Command, args []string) {
	// Link
	linkIssues := config.IssueLinks
	for _, linkIssue := range linkIssues {
		err := linkRelatedJiraIssues(linkIssue.LinkTo, linkIssue.Labels, linkIssue.ReleaseVer)
		perror(err)
	}
}

func linkRelatedJiraIssues(linkToIssue string, labels []string, releaseVer string) error {
	labelsStr := strings.Join(labels, ",")
	jiraIssues := queryJiraIssues(fmt.Sprintf("labels in (%s) and fixVersion = %s and type = Epic", labelsStr, releaseVer))

	for _, issue := range jiraIssues {
		issueLink := &jira.IssueLink{
			InwardIssue:  &jira.Issue{Key: linkToIssue},
			OutwardIssue: &jira.Issue{Key: issue.Key},
			Type:         jira.IssueLinkType{Name: "Relates"},
		}

		_, err := jiraClient.Issue.AddLink(issueLink)
		if err != nil {
			return err
		}
	}

	return nil
}

func genVersionReleaseReportHtml(buf *bytes.Buffer) {
	formatSectionBeginForHtmlOutput(buf)
	buf.WriteString("<p>")
	releaseItemIssue := queryJiraIssues("key = TIDB-3476")
	formatJiraIssueForHtmlOutput(buf, &releaseItemIssue[0])
	formatJiraIssueToExpandForHtmlOutput(buf, &releaseItemIssue[0], nil)
	buf.WriteString("</p>")
	formatSectionEndForHtmlOutput(buf)
}
