# Work Reporter

We have been heavily using Github, JIRA and Confluence in our work, the tool needs to support:

## Weekly

+ Grabs new JIRA issues from the TiDB board, adds to weekly report
+ Grabs new Github issues, adds to weekly report
+ For each team member, grabs his/her current Sprint / next Sprint work from JIRA, reviewed pull requests from Github, adds to weekly report
+ Grabs the JIRA issues that are exceeded the due date.

## Daily

+ Grabs new issues, pull requests during last 24 hours, adds to daily report
+ sends messages to slack channel

## TODO

### Daily

+ Grabs the JIRA issues that are reaching the due date.
+ Grabs the processing JIRA issues without setting a due date, and @ the issue owner.
