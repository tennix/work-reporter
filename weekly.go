package main

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/andygrunwald/go-jira"
	"github.com/spf13/cobra"
)

const jiraLabelColorGrey = "Grey"
const jiraLabelColorRed = "Red"
const jiraLabelColorYellow = "Yellow"
const jiraLabelColorGreen = "Green"
const jiraLabelColorBlue = "Blue"

func newWeeklyDeadlineReportCommand() *cobra.Command {
	m := &cobra.Command{
		Use:   "dead-line-report",
		Short: "Create Weekly Dead-Line-Report",
		Run:   runWeelyDeadLineReportCommandFunc,
	}
	return m
}

func newWeeklyReportCommand() *cobra.Command {
	m := &cobra.Command{
		Use:   "report",
		Short: "Create Weekly Report",
		Run:   runWeeklyReportCommandFunc,
	}
	return m
}

func newRotateSprintCommand() *cobra.Command {
	m := &cobra.Command{
		Use:   "rotate-sprint",
		Short: "Rotate Current Week Sprint",
		Run:   runRotateSprintCommandFunc,
	}
	return m
}

func newWeeklyCommand() *cobra.Command {
	m := &cobra.Command{
		Use:   "weekly",
		Short: "Weely Tasks",
	}
	m.AddCommand(newWeeklyDeadlineReportCommand())
	m.AddCommand(newWeeklyReportCommand())
	return m
}

type JiraIssueArray []jira.Issue

// collectEpicJiraIssue finds out all the epic issues.
// first return value: epic issues
// second return value: others issues
func collectEpicJiraIssue(jiraIssues []jira.Issue) map[string]*JiraIssueArray {
	collectIssueMap := make(map[string]*JiraIssueArray)
	for _, issue := range jiraIssues {
		issueType := strings.ToLower(issue.Fields.Type.Name)
		issuesArr, ok := collectIssueMap[issueType]
		if !ok {
			issuesArr = &JiraIssueArray{}
			collectIssueMap[issueType] = issuesArr
		}

		*issuesArr = append(*issuesArr, issue)
	}

	return collectIssueMap
}

type IssueRepeatChecker map[string]struct{}

//
func (ir IssueRepeatChecker) Check(key string) bool {
	_, ok := ir[key]
	if !ok {
		ir[key] = struct{}{}
	}

	return ok
}

var allFieldsOpts jira.SearchOptions = jira.SearchOptions{
	MaxResults: 1000,
	Fields:     []string{"*all"},
}

// duedate2EstimateSeconds convert duedate to seconds with the same unit as jira time tracking.
// As default, jira time tracking estimated 1d is equal to 8h.
func duedate2EstimateSeconds(duedate time.Time, now time.Time) int {
	days := duedate.Sub(now).Hours() / 24.0
	seconds := days * float64(config.Jira.TimeTrackingDayHours) * 3600
	return int(seconds)
}

func findNextWeekIssues(member Member, now time.Time) []jira.Issue {
	// Find all duedate less than 7 days.
	jql := fmt.Sprintf(`assignee = "%s" AND duedate <= 7d AND statusCategory = indeterminate`, member.Email)
	nextWeekIssues := queryJiraIssuesWithOptions(jql, &allFieldsOpts)
	remainingMorethan7dIssues := queryJiraIssuesWithOptions(fmt.Sprintf(`assignee = "%s" AND remainingEstimate > 7d AND duedate > 7d AND priority >= High AND statusCategory = indeterminate`, member.Email), &allFieldsOpts)

	for _, issue := range remainingMorethan7dIssues {
		if issue.Fields.TimeTracking == nil {
			continue
		}
		dueDate := time.Time(issue.Fields.Duedate)
		if issue.Fields.TimeTracking.RemainingEstimateSeconds >= duedate2EstimateSeconds(dueDate, now) {
			// we need to handle this job in the next week.
			nextWeekIssues = append(nextWeekIssues, issue)
		}
	}

	return nextWeekIssues
}

func createPersonalWeeklyReport(pageBody *bytes.Buffer, member Member, now time.Time) {
	user := getUser(member.Email)
	pageBody.WriteString("<h1><ac:link>")
	pageBody.WriteString(fmt.Sprintf("<ri:user ri:userkey=\"%s\"/>", user.UserKey))
	pageBody.WriteString("</ac:link></h1>")

	formatHeadLineHtmlOutput(pageBody, "h2", "This Week:")

	jiraIssues := queryJiraIssuesWithOptions(fmt.Sprintf(`assignee = "%s" AND %s`, member.Email, config.Jira.WeeklyPersonalIssues), &allFieldsOpts)

	repeatChecker := make(IssueRepeatChecker)
	collectIssueMap := collectEpicJiraIssue(jiraIssues)
	// first format epic, to make up repeatChecker.
	epicIssues := collectIssueMap["epic"]

	formatHeadLineHtmlOutput(pageBody, "h3", "Jira Issue Updates")
	if epicIssues == nil && len(collectIssueMap) == 0 {
		pageBody.WriteString("<p><i>None</i></p>")
	}

	if epicIssues != nil {
		formatHeadLineHtmlOutput(pageBody, "h4", "Epic")
		formatUnorderedListIssuesForHtmlOutput(pageBody, nil, *epicIssues, repeatChecker)
		pageBody.WriteString("<br/>")
	}

	for issueType, issues := range collectIssueMap {
		if issueType == "epic" || (issues != nil && len(*issues) == 0) {
			continue
		}
		issueArr := *issues
		formatHeadLineHtmlOutput(pageBody, "h4", issueArr[0].Fields.Type.Name)
		formatUnorderedListIssuesForHtmlOutput(pageBody, nil, issueArr, repeatChecker)
		pageBody.WriteString("<br/>")
	}

	genPullRequestsReport(pageBody, member.Github, now.AddDate(0, 0, -6).Format("2006-01-02"), now.Format("2006-01-02"))

	// Next week work plans
	formatHeadLineHtmlOutput(pageBody, "h2", "Next Week:")

	nextWeekIssues := findNextWeekIssues(member, now)
	repeatChecker = make(IssueRepeatChecker)
	formatNextWeekPlansForHtmlOutput(pageBody, nextWeekIssues, repeatChecker)

	// create a new confluence page.
	//fmt.Println(pageBody.String())
	// date := fmt.Sprintf("%s ~ %s", now.AddDate(0, 0, -6).Format("2006/01/02"), now.Format("2006/01/02"))

	// createPersonalWeeklyReportToConfluence(date, member.Name, pageBody.String())

}

func runWeeklyReportCommandFunc(cmd *cobra.Command, args []string) {
	now := time.Now()
	var pageBody = new(bytes.Buffer)
	for _, team := range config.Teams {
		for _, member := range team.Members {
			createPersonalWeeklyReport(pageBody, member, now)
			pageBody.WriteString("<hr/>")
		}
	}

	c := createWeeklyReportToConfluence(now, pageBody)
	sendToSlack("Weekly report has been generated %s%s, please manually edit it to add/fix missing/wrong issues.", config.Confluence.Endpoint, c.Links.WebUI)
}

func runWeelyDeadLineReportCommandFunc(cmd *cobra.Command, args []string) {
	//boardID := getBoardID(config.Jira.Project, "scrum")
	//lastSprint := getLatestPassedSprint(boardID)
	//nextSprint := getNearestFutureSprint(boardID)

	var body bytes.Buffer

	//startDate := lastSprint.StartDate.Format(dayFormat)
	//endDate := lastSprint.EndDate.Format(dayFormat)
	//
	//githubStartDate := lastSprint.StartDate.UTC().Format(githubUTCDateFormat)
	//githubEndDate := lastSprint.EndDate.UTC().Format(githubUTCDateFormat)

	formatPageBeginForHtmlOutput(&body)
	genWeeklyReportToc(&body)
	genWeeklyReportDuedate(&body)
	//genWeeklyReportIssuesPRs(&body, githubStartDate, githubEndDate)

	//for _, team := range config.Teams {
	//	fmt.Println(team.Name)
	//	formatSectionBeginForHtmlOutput(&body)
	//	body.WriteString(fmt.Sprintf("<h1>%s Team</h1>", team.Name))
	//	for _, m := range team.Members {
	//		genWeeklyUserPage(&body, m, *lastSprint, *nextSprint)
	//	}
	//	formatSectionEndForHtmlOutput(&body)
	//}

	formatPageEndForHtmlOutput(&body)

	now := time.Now()
	title := fmt.Sprintf("%s SQL-Infra Due Dates", now.Format("2006-01-02"))
	createWeeklyDueDateReport(title, body.String())
}

func runRotateSprintCommandFunc(cmd *cobra.Command, args []string) {
	boardID := getBoardID(config.Jira.Project, "scrum")
	activeSprint := getActiveSprint(boardID)
	nextSprint := createNextSprint(boardID, *activeSprint.EndDate)

	pendingIssues := queryJiraIssues(
		fmt.Sprintf("project = %s and Sprint = %d and statusCategory != Done",
			config.Jira.Project, activeSprint.ID,
		))
	// Close the old sprint.
	updateSprintState(activeSprint.ID, "closed")
	// Move issues to the next sprint.
	moveIssuesToSprint(nextSprint.ID, pendingIssues)
	// Active the next sprint.
	updateSprintState(nextSprint.ID, "active")
	sendToSlack("Current active Sprint %s is closed", activeSprint.Name)
}

func genWeeklyUserPage(buf *bytes.Buffer, m Member, curSprint jira.Sprint, nextSprint jira.Sprint) {
	sprintID := curSprint.ID
	nextSprintID := nextSprint.ID

	html := `
<ac:structured-macro ac:name="jira">
  <ac:parameter ac:name="columns">key,summary,created,updated,priority,status</ac:parameter>
  <ac:parameter ac:name="server">%s</ac:parameter>
  <ac:parameter ac:name="serverId">%s</ac:parameter>
  <ac:parameter ac:name="jqlQuery">project = TIKV AND sprint = %d AND assignee = "%s"</ac:parameter>
</ac:structured-macro>
`

	buf.WriteString(fmt.Sprintf("\n<h2>%s</h2>\n", m.Name))
	buf.WriteString("\n<h3>Work</h3>\n")
	buf.WriteString(fmt.Sprintf(html, config.Jira.Server, config.Jira.ServerID, sprintID, m.Email))
	genPullRequestsReport(buf, m.Github, curSprint.StartDate.Format(dayFormat), curSprint.EndDate.Format(dayFormat))
	if nextSprintID > 0 {
		buf.WriteString("\n<h3>Next Week</h3>\n")
		buf.WriteString(fmt.Sprintf(html, config.Jira.Server, config.Jira.ServerID, nextSprintID, m.Email))
	}
}

func genPullRequestsReport(buf *bytes.Buffer, user, start, end string) {
	buf.WriteString("<h3>GitHub PR updates</h3>")
	prs := getCreatedPullRequests(user, start, &end)
	reviewPRs := getReviewPullRequests(user, start, &end)

	if len(prs) == 0 && len(reviewPRs) == 0 {
		buf.WriteString("<p><i>None</i></p>")
		return
	}

	buf.WriteString("<h4>PR</h4>")
	formatGitHubIssuesForHtmlOutput(buf, prs)
	buf.WriteString("<h4>Review PR</h4>")
	formatGitHubIssuesForHtmlOutput(buf, reviewPRs)
}

func genWeeklyReportDuedate(buf *bytes.Buffer) {
	formatSectionBeginForHtmlOutput(buf)

	buf.WriteString("\n<h1>Issues Exceed Due Date</h1>\n")
	stopStatus := `"Job Closed",Closed,"CAN'T REPRODUCE",Paused,Blocked,完成,TODO,"To Do"`

	html := `
<ac:structured-macro ac:name="jira">
  <ac:parameter ac:name="columns">key,summary,created,updated,assignee,status,due</ac:parameter>
  <ac:parameter ac:name="server">%s</ac:parameter>
  <ac:parameter ac:name="serverId">%s</ac:parameter>
  <ac:parameter ac:name="jqlQuery">%s</ac:parameter>
</ac:structured-macro>
`

	for _, member := range allSQLInfraMemberEmals {
		jqlQuery := fmt.Sprintf(`assignee = %v AND duedate &lt; now() AND status not in (%s)`, member, stopStatus)
		buf.WriteString(fmt.Sprintf(html, config.Jira.Server, config.Jira.ServerID, jqlQuery))
	}

	//	buf.WriteString("\n<h1>Highest Priority</h1>\n")
	//	buf.WriteString("\n<blockquote>Unresolved highest priority OnCalls (priority = Highest AND resolution = Unresolved)</blockquote>\n")
	//	html = `
	//<ac:structured-macro ac:name="jira">
	//  <ac:parameter ac:name="columns">key,summary,created,updated,assignee,status</ac:parameter>
	//  <ac:parameter ac:name="server">%s</ac:parameter>
	//  <ac:parameter ac:name="serverId">%s</ac:parameter>
	//  <ac:parameter ac:name="jqlQuery">project = %s AND priority = Highest AND resolution = Unresolved</ac:parameter>
	//</ac:structured-macro>
	//`
	//	buf.WriteString(fmt.Sprintf(html, config.Jira.Server, config.Jira.ServerID, config.Jira.OnCall))

	formatSectionEndForHtmlOutput(buf)
}

func genWeeklyReportIssuesPRs(buf *bytes.Buffer, start, end string) {
	formatSectionBeginForHtmlOutput(buf)
	issues := getCreatedIssues(start, &end)
	buf.WriteString("\n<h1>New Issues</h1>\n")
	buf.WriteString(fmt.Sprintf("\n<blockquote>New GitHub issues (created: %s..%s)</blockquote>\n", start, end))
	formatGitHubIssuesForHtmlOutput(buf, issues)
	formatSectionEndForHtmlOutput(buf)
}

func createWeeklyDueDateReport(title string, value string) {
	space := config.Confluence.Space
	c := getContentByTitle(space, title)

	if c.Id != "" {
		c = updateContent(c, value)
	} else {
		parent := getContentByTitle(space, config.Confluence.WeeklyDueDatePath)
		c = createContent(space, parent.Id, title, value)
	}

	//sendToSlack("Weekly report for sprint %s is generated: %s%s", title, config.Confluence.Endpoint, c.Links.WebUI)
}

func createConfluencePath(space string, title string) {
	c := getContentByTitle(space, title)
	if c.Id != "" {
		// path is exists.
		return
	}

	parent := getContentByTitle(space, config.Confluence.WeeklyPath)
	c = createContent(space, parent.Id, title, "")
}

func createPersonalWeeklyReportToConfluence(date string, name string, body string) {
	space := config.Confluence.Space
	personalReportTitle := name + " " + date
	c := getContentByTitle(space, personalReportTitle)
	if c.Id != "" {
		// c = updateContent(c, body)
	} else {
		// create this week's path.
		createConfluencePath(space, date)
		parent := getContentByTitle(space, date)
		c = createContent(space, parent.Id, personalReportTitle, body)
	}
}

func createWeeklyReportToConfluence(now time.Time, pageBody *bytes.Buffer) Content {
	weeklyTitle := fmt.Sprintf("%s ~~ %s", now.AddDate(0, 0, -6).Format("2006-01-02"), now.Format("2006-01-02"))
	space := config.Confluence.Space
	c := getContentByTitle(space, weeklyTitle)

	monthTitle := now.Format("2006-01")
	createConfluencePath(space, monthTitle)
	if c.Id != "" {
		c = updateContent(c, pageBody.String())
	} else {
		createConfluencePath(space, monthTitle)
		parent := getContentByTitle(space, monthTitle)
		c = createContent(space, parent.Id, weeklyTitle, pageBody.String())
	}
	return c
}
