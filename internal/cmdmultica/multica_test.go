package cmdmultica

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAgentCommandsExposeCRUD(t *testing.T) {
	root := New()
	var agent *cobra.Command
	for _, command := range root.Commands() {
		if command.Name() == "agent" {
			agent = command
			break
		}
	}
	if agent == nil {
		t.Fatal("agent command is not registered")
	}
	want := map[string]bool{"list": false, "get": false, "create": false, "update": false, "delete": false, "restore": false}
	for _, command := range agent.Commands() {
		if _, ok := want[command.Name()]; ok {
			want[command.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("agent %s command is not registered", name)
		}
	}
}
