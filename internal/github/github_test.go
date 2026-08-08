package github

import "testing"

func TestParseIssues(t *testing.T) {
	data := []byte(`[
		{"author":{"id":"MDQ6VXNlcjIwNDM4Mjg=","is_bot":false,"login":"zzzeid","name":"Zeid"},"labels":[],"number":14093,"title":"gh stack submit fails","updatedAt":"2026-08-07T23:20:27Z","url":"https://github.com/cli/cli/issues/14093"},
		{"author":{"id":"MDQ6VXNlcjg0NDMxNzI0","is_bot":false,"login":"beingminimal","name":"Bhaumin"},"labels":[{"id":"LA_kwDODKw3uc7QD3p7","name":"needs-triage","description":"needs to be reviewed","color":"D6393F"}],"number":14092,"title":"feat: add support","updatedAt":"2026-08-06T20:56:21Z","url":"https://github.com/cli/cli/issues/14092"}
	]`)

	issues, err := parseIssues(data)
	if err != nil {
		t.Fatalf("parseIssues: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Number != 14093 || issues[0].Author.Login != "zzzeid" || len(issues[0].Labels) != 0 {
		t.Errorf("unexpected first issue: %+v", issues[0])
	}
	if issues[1].Labels[0].Name != "needs-triage" {
		t.Errorf("unexpected labels on second issue: %+v", issues[1].Labels)
	}
}

func TestParsePullRequests(t *testing.T) {
	data := []byte(`[
		{"author":{"id":"MDQ6VXNlcjE2MTE1MTA=","is_bot":false,"login":"williammartin","name":"William Martin"},"headRefName":"williammartin-api-host-commit-split","isDraft":true,"number":14104,"title":"Honour api_host","updatedAt":"2026-08-07T18:46:38Z","url":"https://github.com/cli/cli/pull/14104"},
		{"author":{"is_bot":true,"login":"app/dependabot"},"headRefName":"dependabot/github_actions/azure/login-3.0.1","isDraft":false,"number":14101,"title":"chore(deps): bump azure/login","updatedAt":"2026-08-07T15:06:14Z","url":"https://github.com/cli/cli/pull/14101"}
	]`)

	prs, err := parsePullRequests(data)
	if err != nil {
		t.Fatalf("parsePullRequests: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 pull requests, got %d", len(prs))
	}
	if prs[0].HeadRefName != "williammartin-api-host-commit-split" || !prs[0].IsDraft {
		t.Errorf("unexpected first PR: %+v", prs[0])
	}
	if prs[1].Author.Login != "app/dependabot" || prs[1].IsDraft {
		t.Errorf("unexpected second PR: %+v", prs[1])
	}
}

func TestParseIssuesEmpty(t *testing.T) {
	issues, err := parseIssues([]byte(`[]`))
	if err != nil {
		t.Fatalf("parseIssues: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %+v", issues)
	}
}
