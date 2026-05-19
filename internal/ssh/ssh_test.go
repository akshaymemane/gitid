package ssh

import (
	"strings"
	"testing"
)

func TestReplaceManagedBlock(t *testing.T) {
	existing := "Host old\n  HostName old.example\n\n" + managedStart + "\nold body\n" + managedEnd + "\n"
	got := replaceManagedBlock(existing, "Host github-work\n  HostName github.com")

	if strings.Contains(got, "old body") {
		t.Fatalf("old managed body was not replaced:\n%s", got)
	}
	if !strings.Contains(got, "Host old") {
		t.Fatalf("unmanaged SSH config was removed:\n%s", got)
	}
	if !strings.Contains(got, "Host github-work") {
		t.Fatalf("new managed body missing:\n%s", got)
	}
}
