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
		AssigneeID:  "agent-1",
		ProjectID:   "project-1",
		Status:      "todo",
	}); err != nil {
		t.Fatal(err)
	}
	want := []string{"issue", "create", "--title", "Fix login", "--description-stdin", "--output", "json", "--assignee-id", "agent-1", "--project", "project-1", "--status", "todo"}
	if !reflect.DeepEqual(gotArgs, want) {
		t.Fatalf("args = %#v, want %#v", gotArgs, want)
	}
	if string(body) != "source body" {
		t.Fatalf("body = %q", body)
	}
}

func TestAgentLifecycleCommands(t *testing.T) {
	tests := []struct {
		name string
		call func(Client) error
		want []string
	}{
		{
			name: "list archived",
			call: func(c Client) error {
				_, err := c.ListAgents(context.Background(), true)
				return err
			},
			want: []string{"agent", "list", "--include-archived", "--output", "json"},
		},
		{
			name: "get",
			call: func(c Client) error {
				_, err := c.GetAgent(context.Background(), "agent-1")
				return err
			},
			want: []string{"agent", "get", "agent-1", "--output", "json"},
		},
		{
			name: "get issue",
			call: func(c Client) error {
				_, err := c.GetIssue(context.Background(), "MUL-1")
				return err
			},
			want: []string{"issue", "get", "MUL-1", "--output", "json"},
		},
		{
			name: "archive",
			call: func(c Client) error {
				_, err := c.ArchiveAgent(context.Background(), "agent-1")
				return err
			},
			want: []string{"agent", "archive", "agent-1", "--output", "json"},
		},
		{
			name: "restore",
			call: func(c Client) error {
				_, err := c.RestoreAgent(context.Background(), "agent-1")
				return err
			},
			want: []string{"agent", "restore", "agent-1", "--output", "json"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got []string
			client := Client{Command: "multica-test", Exec: func(_ context.Context, _ string, args []string, _ io.Reader) ([]byte, []byte, error) {
				got = append([]string(nil), args...)
				return []byte(`{"id":"agent-1"}`), nil, nil
			}}
			if err := test.call(client); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("args = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestCreateAndUpdateAgentArguments(t *testing.T) {
	var calls [][]string
	client := Client{Command: "multica-test", Exec: func(_ context.Context, _ string, args []string, _ io.Reader) ([]byte, []byte, error) {
		calls = append(calls, append([]string(nil), args...))
		return []byte(`{"id":"agent-1"}`), nil, nil
	}}
	max := int32(3)
	if _, err := client.CreateAgent(context.Background(), AgentCreateInput{
		Name: "Builder", RuntimeID: "runtime-1", Description: "builds", Instructions: "be concise", Model: "gpt-4o", MaxConcurrentTasks: &max,
	}); err != nil {
		t.Fatal(err)
	}
	name := "Builder 2"
	status := "paused"
	if _, err := client.UpdateAgent(context.Background(), "agent-1", AgentUpdateInput{Name: &name, Status: &status}); err != nil {
		t.Fatal(err)
	}
	wantCreate := []string{"agent", "create", "--name", "Builder", "--runtime-id", "runtime-1", "--description", "builds", "--instructions", "be concise", "--model", "gpt-4o", "--max-concurrent-tasks", "3", "--output", "json"}
	wantUpdate := []string{"agent", "update", "agent-1", "--name", "Builder 2", "--status", "paused", "--output", "json"}
	if !reflect.DeepEqual(calls, [][]string{wantCreate, wantUpdate}) {
		t.Fatalf("args = %#v, want %#v", calls, [][]string{wantCreate, wantUpdate})
	}
}

func TestIssueListUpdateAndComments(t *testing.T) {
	var calls [][]string
	var bodies [][]byte
	client := Client{Command: "multica-test", Exec: func(_ context.Context, _ string, args []string, stdin io.Reader) ([]byte, []byte, error) {
		calls = append(calls, append([]string(nil), args...))
		if stdin == nil {
			bodies = append(bodies, nil)
			return []byte(`{}`), nil, nil
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, data)
		return []byte(`{}`), nil, nil
	}}

	if _, err := client.ListIssues(context.Background(), IssueListInput{AssigneeID: "a1", Status: "todo", Limit: 10, Offset: 5}); err != nil {
		t.Fatal(err)
	}
	description := "new body"
	status := "done"
	if _, err := client.UpdateIssue(context.Background(), "MUL-1", IssueUpdateInput{Description: &description, Status: &status}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.AddComment(context.Background(), "MUL-1", "hello", "c1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListComments(context.Background(), "MUL-1", true, 3); err != nil {
		t.Fatal(err)
	}

	wantCalls := [][]string{
		{"issue", "list", "--output", "json", "--assignee-id", "a1", "--status", "todo", "--limit", "10", "--offset", "5"},
		{"issue", "update", "MUL-1", "--output", "json", "--description-stdin", "--status", "done"},
		{"issue", "comment", "add", "MUL-1", "--content-stdin", "--output", "json", "--parent", "c1"},
		{"issue", "comment", "list", "MUL-1", "--output", "json", "--roots-only", "--recent", "3"},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("args = %#v, want %#v", calls, wantCalls)
	}
	wantBodies := [][]byte{nil, []byte("new body"), []byte("hello"), nil}
	if !reflect.DeepEqual(bodies, wantBodies) {
		t.Fatalf("bodies = %#v, want %#v", bodies, wantBodies)
	}
}

func TestAddCommentRejectsEmptyContent(t *testing.T) {
	callCount := 0
	client := Client{Command: "multica-test", Exec: func(_ context.Context, _ string, _ []string, _ io.Reader) ([]byte, []byte, error) {
		callCount++
		return []byte(`{}`), nil, nil
	}}
	if _, err := client.AddComment(context.Background(), "MUL-1", "  ", ""); err == nil {
		t.Fatal("expected an error for empty content")
	}
	if callCount != 0 {
		t.Fatalf("multica was invoked %d times", callCount)
	}
}
