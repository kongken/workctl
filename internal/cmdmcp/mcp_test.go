package cmdmcp

import (
	"context"
	"encoding/json"
	"io"
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
}

func TestServerExposesBridgeTools(t *testing.T) {
	server := newServer("")
	want := map[string]bool{
		"get_linear_issue":        false,
		"list_multica_agents":     false,
		"sync_agent_prompt":       false,
		"delegate_to_multica":     false,
		"get_multica_work_status": false,
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
