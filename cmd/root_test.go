package cmd

import (
	"io"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		args    []string
		wantErr string
		check   func(*testing.T, *commandTree)
	}{
		{name: "too many arguments", args: []string{"https://example.com", "extra"}, wantErr: "unexpected argument"},
		{name: "invalid flag", args: []string{"--invalid-flag"}, wantErr: "unknown flag"},
		{name: "exclusive output modes", args: []string{"--quiet", "--verbose", "https://example.com"}, wantErr: "can't be used together"},
		{name: "short flags", args: []string{"-v", "-m", "-p", "20", "https://example.com"}, check: func(t *testing.T, tree *commandTree) {
			if !tree.Verbose || !tree.Mobile || tree.MaxPages != 20 {
				t.Fatalf("flags not parsed: %+v", tree)
			}
		}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			tree, err := parseCommand(test.args, io.Discard)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCommand() error = %v", err)
			}
			test.check(t, tree)
		})
	}
}

func TestParseServeCommand(t *testing.T) {
	t.Parallel()
	cmd, err := parseServeCommand([]string{"./saved", "-p", "3000"}, io.Discard)
	if err != nil {
		t.Fatalf("parseServeCommand() error = %v", err)
	}
	if cmd.Directory != "./saved" || cmd.Port != 3000 {
		t.Fatalf("serve flags not parsed: %+v", cmd)
	}
}

func TestRunRequiresURL(t *testing.T) {
	t.Parallel()
	err := run(t.Context(), &commandTree{NoRobots: true}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "provide at least one URL") {
		t.Fatalf("run() error = %v", err)
	}
}
