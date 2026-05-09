package common

import (
	"os"
	"testing"
)

func TestConfirmAction_AcceptsYes(t *testing.T) {
	orig := os.Stdin
	defer func() { os.Stdin = orig }()

	f, err := os.CreateTemp("", "confirm-action")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString("yes\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	os.Stdin = f

	if !ConfirmAction("proceed?") {
		t.Fatal("expected true for yes")
	}
}
