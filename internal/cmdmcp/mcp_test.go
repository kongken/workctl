package cmdmcp

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/kongken/workctl/internal/delegation"
	"github.com/kongken/workctl/internal/multicacli"
	"github.com/spf13/cobra"
)

func TestMCPCommandGroup(t *testing.T) {
	root := New()
	var serve *cobra.Command
	for _, command := range root.Commands() {
		if command.Name() == "serve" {
			serve = command
		}
	}
	if serve == nil {
		t.Fatal("serve command is not registered")
	}
	if serve.Flags().Lookup("multica-command") == nil {
		t.Fatal("serve command is missing --multica-command")
	}
	if serve.Flags().Lookup("read-only") == nil {
		t.Fatal("serve command is missing --read-only")
	}
}

func TestServerExposesBridgeTools(t *testing.T) {
	server := newServer("", false)
	want := map[string]bool{
		"get_linear_issue":            false,
		"list_multica_agents":         false,
		"get_multica_agent":           false,
		"create_multica_agent":        false,
		"update_multica_agent":        false,
		"archive_multica_agent":       false,
		"restore_multica_agent":       false,
		"sync_agent_prompt":           false,
		"get_multica_work_status":     false,
		"list_multica_issues":         false,
		"create_multica_issue":        false,
		"update_multica_issue":        false,
		"add_multica_issue_comment":   false,
		"list_multica_issue_comments": false,
		"delegate_to_multica":         false,
	}
	for _, tool := range server.Tools {
		if _, ok := want[tool.Name]; !ok {
			t.Errorf("unexpected tool %q", tool.Name)
			continue
		}
		want[tool.Name] = true
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %q has no input schema", tool.Name)
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("tool %q is not registered", name)
		}
	}
}

func TestServerReadOnlyHidesWriteTools(t *testing.T) {
	server := newServer("", true)
	for _, tool := range server.Tools {
		if tool.Write {
			t.Errorf("read-only server exposes write tool %q", tool.Name)
		}
	}
	if len(server.Tools) != 6 {
		t.Fatalf("read-only tool count = %d, want 6", len(server.Tools))
	}
}

func fakeClient(t *testing.T, stdout string) func() multicacli.Client {
	t.Helper()
	return func() multicacli.Client {
		return multicacli.Client{Command: "multica-test", Exec: func(_ context.Context, _ string, _ []string, _ io.Reader) ([]byte, []byte, error) {
			return []byte(stdout), nil, nil
		}}
	}
}

func TestListAgentsToolReturnsFormattedJSON(t *testing.T) {
	ts := toolset{client: fakeClient(t, `{"agents":[{"id":"a1"}]}`)}
	got, err := ts.listMulticaAgents(context.Background(), json.RawMessage(`{"include_archived":true}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, `"agents"`) || !strings.Contains(got, "a1") {
		t.Fatalf("output = %s", got)
	}
}

func TestGetMulticaWorkStatusRequiresIssue(t *testing.T) {
	ts := toolset{client: fakeClient(t, `{}`)}
	if _, err := ts.getMulticaWorkStatus(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Fatal("expected an error for a missing issue_id")
	}
}

func TestSyncAgentPromptRequiresFile(t *testing.T) {
	ts := toolset{client: fakeClient(t, `{}`)}
	if _, err := ts.syncAgentPrompt(context.Background(), json.RawMessage(`{"agent_id":"a1"}`)); err == nil {
		t.Fatal("expected an error for a missing prompt_file")
	}
}

func TestDelegateToMulticaDefaultsStatus(t *testing.T) {
	fake := &fakeMultica{}
	ts := toolset{service: func() delegation.Service {
		return delegation.Service{
			ResolveIssue: func(string) (string, error) { return "ENG-1", nil },
			FetchIssue: func(context.Context, string) (json.RawMessage, error) {
				return json.RawMessage(`{"identifier":"ENG-1","title":"Fix login"}`), nil
			},
			Multica: fake,
		}
	}}
	got, err := ts.delegateToMultica(context.Background(), json.RawMessage(`{"issue":"ENG-1","agent_id":"a1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if fake.createInput.Status != "todo" {
		t.Fatalf("status = %q", fake.createInput.Status)
	}
	if !strings.Contains(got, `"linear_issue": "ENG-1"`) {
		t.Fatalf("output = %s", got)
	}
}

type fakeMultica struct {
	createInput multicacli.CreateIssueInput
}

func (f *fakeMultica) UpdateAgentInstructions(context.Context, string, string) (json.RawMessage, error) {
	return json.RawMessage(`{"id":"a1"}`), nil
}

func (f *fakeMultica) CreateIssue(_ context.Context, input multicacli.CreateIssueInput) (json.RawMessage, error) {
	f.createInput = input
	return json.RawMessage(`{"identifier":"MUL-1"}`), nil
}

func capturingClient(stdout string, gotArgs *[]string) func() multicacli.Client {
	return func() multicacli.Client {
		return multicacli.Client{Command: "multica-test", Exec: func(_ context.Context, _ string, args []string, _ io.Reader) ([]byte, []byte, error) {
			*gotArgs = append([]string(nil), args...)
			return []byte(stdout), nil, nil
		}}
	}
}

func TestCreateMulticaAgentBuildsCommand(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`{"id":"a1"}`, &args)}
	if _, err := ts.createMulticaAgent(context.Background(), json.RawMessage(`{"name":"Bot","runtime_id":"r1"}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"agent", "create", "--name", "Bot", "--runtime-id", "r1", "--output", "json"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestCreateMulticaAgentRequiresRuntime(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`{}`, &args)}
	if _, err := ts.createMulticaAgent(context.Background(), json.RawMessage(`{"name":"Bot"}`)); err == nil {
		t.Fatal("expected an error for a missing runtime_id")
	}
	if len(args) != 0 {
		t.Fatalf("command ran despite a validation error: %#v", args)
	}
}

func TestUpdateMulticaAgentOmitsAbsentFields(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`{"id":"a1"}`, &args)}
	if _, err := ts.updateMulticaAgent(context.Background(), json.RawMessage(`{"agent_id":"a1","status":"archived"}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"agent", "update", "a1", "--status", "archived", "--output", "json"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestArchiveMulticaAgent(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`{"id":"a1"}`, &args)}
	if _, err := ts.archiveMulticaAgent(context.Background(), json.RawMessage(`{"agent_id":"a1"}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"agent", "archive", "a1", "--output", "json"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestCreateMulticaIssuePassesFields(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`{"identifier":"MUL-1"}`, &args)}
	if _, err := ts.createMulticaIssue(context.Background(), json.RawMessage(`{"title":"T","description":"D","assignee_id":"a1","project_id":"p1","status":"todo","priority":"high","parent_id":"MUL-2"}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"issue", "create", "--title", "T", "--description-stdin", "--output", "json", "--assignee-id", "a1", "--project", "p1", "--status", "todo", "--priority", "high", "--parent", "MUL-2"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestUpdateMulticaIssueOmitsAbsentFields(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`{"identifier":"MUL-1"}`, &args)}
	if _, err := ts.updateMulticaIssue(context.Background(), json.RawMessage(`{"issue_id":"MUL-1","status":"done"}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"issue", "update", "MUL-1", "--output", "json", "--status", "done"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestListMulticaIssuesPassesFilters(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`{"issues":[]}`, &args)}
	if _, err := ts.listMulticaIssues(context.Background(), json.RawMessage(`{"assignee_id":"a1","status":"in_progress","limit":10,"offset":20}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"issue", "list", "--output", "json", "--assignee-id", "a1", "--status", "in_progress", "--limit", "10", "--offset", "20"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestListMulticaIssueCommentsPassesFlags(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`[]`, &args)}
	if _, err := ts.listMulticaIssueComments(context.Background(), json.RawMessage(`{"issue_id":"MUL-1","roots_only":true,"recent":5}`)); err != nil {
		t.Fatal(err)
	}
	want := []string{"issue", "comment", "list", "MUL-1", "--output", "json", "--roots-only", "--recent", "5"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestAddMulticaIssueCommentRequiresContent(t *testing.T) {
	var args []string
	ts := toolset{client: capturingClient(`{}`, &args)}
	if _, err := ts.addMulticaIssueComment(context.Background(), json.RawMessage(`{"issue_id":"MUL-1"}`)); err == nil {
		t.Fatal("expected an error for empty content")
	}
	if len(args) != 0 {
		t.Fatalf("command ran despite a validation error: %#v", args)
	}
}
