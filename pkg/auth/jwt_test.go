package auth

import (
	"testing"
	"time"
)

func TestIssueAndParse(t *testing.T) {
	tok, err := Issue("user-1", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	sub, err := Parse(tok)
	if err != nil {
		t.Fatal(err)
	}
	if sub != "user-1" {
		t.Fatalf("got %s", sub)
	}
}
