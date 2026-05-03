package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/IvanBern/ivai-os/internal/tools"
)

func executeGitHubPR(argsJSON string) (string, error) {
	var args struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Base  string `json:"base"`
		Repo  string `json:"repo"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	base := args.Base
	if base == "" {
		base = "main"
	}
	cmd := fmt.Sprintf("gh pr create --title %q --body %q --base %s", args.Title, args.Body, base)
	if args.Repo != "" {
		cmd = fmt.Sprintf("cd %s && %s", args.Repo, cmd)
	}
	return tools.ExecuteCommand(cmd)
}

func executeCodeHealth(repoPath string) (string, error) {
	body, _ := json.Marshal(map[string]string{"repo": repoPath})
	result, err := tools.HttpRequest("POST", "http://host.orb.internal:9876/", string(body), map[string]string{"Content-Type": "application/json"})
	if err != nil {
		return "", err
	}
	var resp struct {
		OK     bool   `json:"ok"`
		Output string `json:"output"`
	}
	json.Unmarshal([]byte(result), &resp)
	return resp.Output, nil
}

func executeCodeHealthTool(argsJSON string) (string, error) {
	var args struct {
		Repo string `json:"repo"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	return executeCodeHealth(args.Repo)
}

func executeCreateIssue(argsJSON string) (string, error) {
	var args struct {
		Title    string `json:"title"`
		Body     string `json:"body"`
		Labels   string `json:"labels"`
		Assignee string `json:"assignee"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	cmd := fmt.Sprintf("gh issue create --repo IvanBern/ivai-os --title %q --body %q", args.Title, args.Body)
	if args.Labels != "" {
		cmd += fmt.Sprintf(" --label %q", args.Labels)
	}
	if args.Assignee != "" {
		cmd += fmt.Sprintf(" --assignee %q", args.Assignee)
	}
	return tools.ExecuteCommand(cmd)
}

func executeListIssues(argsJSON string) (string, error) {
	var args struct {
		State  string `json:"state"`
		Labels string `json:"labels"`
		Limit  string `json:"limit"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	if args.State == "" {
		args.State = "open"
	}
	if args.Limit == "" {
		args.Limit = "10"
	}
	cmd := fmt.Sprintf("gh issue list --repo IvanBern/ivai-os --state %s --limit %s --json title,state,labels,assignees", args.State, args.Limit)
	if args.Labels != "" {
		cmd += fmt.Sprintf(" --label %q", args.Labels)
	}
	return tools.ExecuteCommand(cmd)
}

func executeUpdateWiki(argsJSON string) (string, error) {
	var args struct {
		Page    string `json:"page"`
		Content string `json:"content"`
	}
	json.Unmarshal([]byte(argsJSON), &args)
	filename := args.Page + ".md"
	cmd := fmt.Sprintf("cd /tmp && rm -rf ivai-wiki && git clone https://github.com/IvanBern/ivai-os.wiki.git ivai-wiki && cd ivai-wiki && cat > %s << 'WIKIEOF'\n%s\nWIKIEOF\n && git add %s && git commit -m 'update %s' && git push", filename, args.Content, filename, args.Page)
	return tools.ExecuteCommand(cmd)
}

func featureEnabled(name string) bool {
	return os.Getenv("IVAI_FEATURE_"+strings.ToUpper(name)) != "false"
}
