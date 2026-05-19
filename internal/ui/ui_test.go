package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestPlainCheckOutput(t *testing.T) {
	var out bytes.Buffer
	view := New(&out, Options{Plain: true})

	view.Check(CheckOK, "Git installed", "")
	view.Check(CheckWarn, "GitHub auth", "not logged in")
	view.Check(CheckFail, "SSH key", "missing")

	got := out.String()
	want := "[OK] Git installed\n[WARN] GitHub auth: not logged in\n[FAIL] SSH key: missing\n"
	if got != want {
		t.Fatalf("plain checks mismatch\nwant:\n%q\ngot:\n%q", want, got)
	}
}

func TestPlainProfilesTable(t *testing.T) {
	var out bytes.Buffer
	view := New(&out, Options{Plain: true})

	got := view.ProfilesTable([][]string{{"*", "work", "Work User <work@example.com>", "workhub", "yes"}})

	for _, part := range []string{"Active", "Profile", "work", "work@example.com", "workhub"} {
		if !strings.Contains(got, part) {
			t.Fatalf("ProfilesTable() missing %q in:\n%s", part, got)
		}
	}
}

func TestForceColorOverridesPlainTerminalDetection(t *testing.T) {
	var out bytes.Buffer
	view := New(&out, Options{Color: "always"})

	view.Success("Initialized gitid")

	if !strings.Contains(out.String(), "\x1b[") {
		t.Fatalf("forced color output did not contain ANSI escape codes: %q", out.String())
	}
}
