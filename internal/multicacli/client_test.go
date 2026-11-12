package multicacli

import (
	"context"
	"io"
	"reflect"
	"testing"
)

func TestUpdateAgentInstructionsPassesPromptAsOneArgument(t *testing.T) {
	var gotCommand string
	var gotArgs []string
	var gotStdin io.Reader
	client := Client{
		Command: "multica-test",
		Exec: func(_ context.Context, command string, args []string, stdin io.Reader) ([]byte, []byte, error) {
			gotCommand = command
			gotArgs = append([]string(nil), args...)
			gotStdin = stdin
			return []byte(`{"id":"agent-1"}`), nil, nil
		},
	}

	if _, err := client.UpdateAgentInstructions(context.Background(), "agent-1", "Use $HOME and `code` safely"); err != nil {
		t.Fatal(err)
	}
	if gotCommand != "multica-test" {
		t.Fatalf("command = %q", gotCommand)
	}
	want := []string{"agent", "update", "agent-1", "--instructions", "Use $HOME and `code` safely", "--output", "json"}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
	if gotStdin != nil {
		t.Fatal("prompt update should not use stdin")
	}
}

func TestCreateIssueStreamsDescriptionAndOptionalProject(t *testing.T) {
	var gotArgs []string
	var body []byte
	client := Client{
		Command: "multica-test",
		Exec: func(_ context.Context, _ string, args []string, stdin io.Reader) ([]byte, []byte, error) {
			gotArgs = append([]string(nil), args...)
			var err error
			body, err = io.ReadAll(stdin)
			if err != nil {
				t.Fatal(err)
			}
			return []byte(`{"id":"issue-1"}`), nil, nil
		},
	}

	if _, err := client.CreateIssue(context.Background(), CreateIssueInput{
		Title:       "Fix login",
		Description: "source body",
		AgentID:     "agent-1",
		ProjectID:   "project-1",
		Status:      "todo",
	}); err != nil {
		t.Fatal(err)
	}
	want := []string{"issue", "create", "--title", "Fix login", "--description-stdin", "--assignee-id", "agent-1", "--status", "todo", "--output", "json", "--project", "project-1"}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
	if string(body) != "source body" {
		t.Fatalf("body = %q", body)
	}
}
