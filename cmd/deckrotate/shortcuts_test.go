package main

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

// Every global shortcut must run a command the CLI accepts.
func TestShortcutFileActionsAreValidCommands(t *testing.T) {
	f, err := os.Open("../../packaging/kglobalaccel/io.github.rustyraptor.deckrotate.desktop")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var declared []string
	actions := map[string]string{} // action id -> Exec
	section := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case strings.HasPrefix(line, "[Desktop Action "):
			section = strings.TrimSuffix(strings.TrimPrefix(line, "[Desktop Action "), "]")
		case strings.HasPrefix(line, "["):
			section = ""
		case section == "" && strings.HasPrefix(line, "Actions="):
			declared = strings.Split(strings.TrimSuffix(strings.TrimPrefix(line, "Actions="), ";"), ";")
		case section != "" && strings.HasPrefix(line, "Exec="):
			actions[section] = strings.TrimPrefix(line, "Exec=")
		}
	}
	if len(declared) < 5 {
		t.Fatalf("only %d actions declared: %v", len(declared), declared)
	}
	for _, id := range declared {
		exec, ok := actions[id]
		if !ok {
			t.Errorf("action %s has no Exec", id)
			continue
		}
		fields := strings.Fields(exec)
		if fields[0] != "@BINDIR@/deckrotate" {
			t.Errorf("%s: Exec should start with @BINDIR@/deckrotate, got %q", id, fields[0])
		}
		if _, _, err := parseCommand(fields[1:]); err != nil {
			t.Errorf("%s: %q is not a valid command: %v", id, exec, err)
		}
	}
	for _, must := range []string{"ToggleLock", "RotateCW", "RotateCCW", "ToggleMode", "Accept"} {
		if _, ok := actions[must]; !ok {
			t.Errorf("missing action %s", must)
		}
	}
}
