package main

import (
	"bytes"
	"fmt"
	"github.com/andygrunwald/go-jira"
	"github.com/google/go-github/github"
	"html"
	"strings"
)

const (
	processIssueField = "customfield_11100"
)

func formatPageBeginForHtmlOutput(buf *bytes.Buffer) {
	buf.WriteString(`<ac:layout>`)
}

func formatPageEndForHtmlOutput(buf *bytes.Buffer) {
	buf.WriteString(`</ac:layout>`)
}

func formatHeadLineHtmlOutput(buf *bytes.Buffer, headlineTag string, headlineText string) {
	buf.WriteString(fmt.Sprintf(`<%s>%s</%s>`, headlineTag, headlineText, headlineTag))
}

func formatSectionBeginForHtmlOutput(buf *bytes.Buffer) {
	buf.WriteString(`<ac:layout-section ac:type="single"><ac:layout-cell><hr/>`)
	buf.WriteString("\n")
}

func formatSectionEndForHtmlOutput(buf *bytes.Buffer) {
	buf.WriteString(`</ac:layout-cell></ac:layout-section>`)
	buf.WriteString("\n")
}

func formatLabelForHtmlOutput(name string, color string) string {
	s := fmt.Sprintf(`
	<ac:structured-macro ac:macro-id="9f29312a-2730-48f0-ab6d-91d6bef3f016" ac:name="status" ac:schema-version="1">
		<ac:parameter ac:name="colour">%s</ac:parameter>
		<ac:parameter ac:name="title">%s</ac:parameter>
	</ac:structured-macro>`, color, html.EscapeString(name))
	return s
}

func formatJiraIssueForHtmlOutput(buf *bytes.Buffer, issue *jira.Issue) {
	html := `
	<p><ac:structured-macro ac:name="jira" ac:schema-version="1">
			<ac:parameter ac:name="server">%s</ac:parameter>
			<ac:parameter ac:name="columns">key,summary,type,created,updated,due,assignee,reporter,priority,status,resolution</ac:parameter>
			<ac:parameter ac:name="serverId">%s</ac:parameter>
			<ac:parameter ac:name="key">%s</ac:parameter>
	</ac:structured-macro></p>
	`
	buf.WriteString(fmt.Sprintf(html, config.Jira.Server, config.Jira.ServerID, issue.Key))
}

func getIssueProgressField(issue *jira.Issue) string {
	processField, ok := issue.Fields.Unknowns[processIssueField]
	if !ok {
		return ""
	}

	if processFieldStr, ok := processField.(string); ok {
		return processFieldStr
	}

	return ""
}

func formatJiraIssueWithProgressForHtmlOutput(buf *bytes.Buffer, issue *jira.Issue, repeatChecker IssueRepeatChecker) {
	issueType := strings.ToLower(issue.Fields.Type.Name)
	switch issueType {
	case "epic":
		formatEpicIssueWithProgressForHtmlOutput(buf, issue, repeatChecker)
	default:
		formatNormalIssueWithProgressForHtmlOutput(buf, issue)
	}
}

func formatNormalIssueWithProgressForHtmlOutput(buf *bytes.Buffer, issue *jira.Issue) {
	html := `
    <ac:structured-macro ac:name="jira" ac:schema-version="1">
      <ac:parameter ac:name="server">%s</ac:parameter>
      <ac:parameter ac:name="serverId">%s</ac:parameter>
      <ac:parameter ac:name="key">%s</ac:parameter>
    </ac:structured-macro> %s`

	progress := getIssueProgressField(issue)
	if progress != "" {
		progress = ": " + progress
	}

	buf.WriteString(fmt.Sprintf(html, config.Jira.Server, config.Jira.ServerID, issue.Key, progress))
}

func formatUnorderedListIssuesForHtmlOutput(buf *bytes.Buffer, issues []jira.Issue, repeatChecker IssueRepeatChecker) {
	if len(issues) == 0 {
		return
	}

	// start unorderd list
	buf.WriteString(`<ul>`)

	for _, issue := range issues {
		exists := repeatChecker.Check(issue.Key)
		if exists {
			continue
		}
		buf.WriteString(`<li>`)
		formatJiraIssueWithProgressForHtmlOutput(buf, &issue, repeatChecker)
		buf.WriteString(`</li>`)
	}
	buf.WriteString(`</ul>`)
}

func formatEpicIssueWithProgressForHtmlOutput(buf *bytes.Buffer, issue *jira.Issue, repeatChecker IssueRepeatChecker) {
	// format epic issue self.
	formatNormalIssueWithProgressForHtmlOutput(buf, issue)

	// format issues belongs to this epic.
	// TODO: make jql this configurable.
	issuesInEpic := queryJiraIssues(fmt.Sprintf(`"Epic Link" = %s AND %s`, issue.Key, config.Jira.WeeklyPersonalIssues))

	formatUnorderedListIssuesForHtmlOutput(buf, issuesInEpic, repeatChecker)
}

func formatJiraIssueToExpandForHtmlOutput(buf *bytes.Buffer, issue *jira.Issue, parentIssue *jira.Issue) {
	// start of the expand
	buf.WriteString(fmt.Sprintf(`
	<ac:structured-macro ac:name="expand" ac:schema-version="1">
	<ac:parameter ac:name="title">%s linked issues</ac:parameter>
	<ac:rich-text-body>
	`, issue.Key))

	// list of non-epic issues
	buf.WriteString(fmt.Sprintf(`<p>
	<ac:structured-macro ac:name="jira" ac:schema-version="1">
		<ac:parameter ac:name="server">%s</ac:parameter>
		<ac:parameter ac:name="columns">key,summary,type,created,updated,due,assignee,priority,status,resolution</ac:parameter>
		<ac:parameter ac:name="maximumIssues">50</ac:parameter>
		<ac:parameter ac:name="jqlQuery">(issue in linkedIssues(%s) or "Epic Link" = %s) AND type != "Version Release" and type != Epic </ac:parameter>
		<ac:parameter ac:name="serverId">%s</ac:parameter>
	</ac:structured-macro></p>
	`, config.Jira.Server, issue.Key, issue.Key, config.Jira.ServerID))

	var epicIssues []jira.Issue
	if parentIssue != nil {
		epicIssues = queryJiraIssues(fmt.Sprintf(`issue in linkedIssues(%s) AND type != "Version Release" and type = Epic and key != %s`, issue.Key, parentIssue.Key))
	} else {
		epicIssues = queryJiraIssues(fmt.Sprintf(`issue in linkedIssues(%s) AND type != "Version Release" and type = Epic`, issue.Key))
	}

	for _, epicIssue := range epicIssues {
		// make expands for epic issue.
		formatJiraIssueForHtmlOutput(buf, &epicIssue)
		formatJiraIssueToExpandForHtmlOutput(buf, &epicIssue, issue)
	}

	// end of the expand
	buf.WriteString(`
	</ac:rich-text-body>
	</ac:structured-macro>
	`)
}

func formatGitHubIssueForHtmlOutput(issue github.Issue) string {
	isFromTeam := false
	login := issue.GetUser().GetLogin()

	for _, id := range allMembers {
		if strings.EqualFold(id, login) {
			isFromTeam = true
			break
		}
	}

	var labelColor = jiraLabelColorGrey
	if issue.GetState() == "closed" {
		labelColor = jiraLabelColorGreen
	}

	s := fmt.Sprintf(
		`%s <a href="%s">%s</a> by @%s`,
		formatLabelForHtmlOutput(regexRepo.FindStringSubmatch(issue.GetHTMLURL())[1], labelColor),
		issue.GetHTMLURL(),
		html.EscapeString(issue.GetTitle()),
		html.EscapeString(issue.GetUser().GetLogin()),
	)

	if issue.Assignees != nil && len(issue.Assignees) > 0 {
		s += fmt.Sprintf(", assigned to")
		for _, assigne := range issue.Assignees {
			s += fmt.Sprintf(" @%s", assigne.GetLogin())
		}
	}

	if !isFromTeam {
		s += " " + formatLabelForHtmlOutput("Community", jiraLabelColorBlue)
	}

	return s
}

func formatGitHubIssuesForHtmlOutput(buf *bytes.Buffer, issues []github.Issue) {
	if len(issues) == 0 {
		buf.WriteString("<p><i>None</i></p>\n")
		return
	}
	buf.WriteString("<ul>")
	for _, issue := range issues {
		buf.WriteString(fmt.Sprintf("<li>%s</li>\n", formatGitHubIssueForHtmlOutput(issue)))
	}
	buf.WriteString("</ul>")
}

func genWeeklyReportToc(buf *bytes.Buffer) {
	formatSectionBeginForHtmlOutput(buf)

	toc := `
<ac:structured-macro ac:name="toc">
  <ac:parameter ac:name="printable">true</ac:parameter>
  <ac:parameter ac:name="style">square</ac:parameter>
  <ac:parameter ac:name="maxLevel">2</ac:parameter>
  <ac:parameter ac:name="class">bigpink</ac:parameter>
  <ac:parameter ac:name="type">list</ac:parameter>
</ac:structured-macro>
	`
	buf.WriteString(toc)

	formatSectionEndForHtmlOutput(buf)
}
