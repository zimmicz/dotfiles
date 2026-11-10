package main

import "testing"

func TestNumberLists(t *testing.T) {
	in := "Otázky:\n# první\n# druhá\n\n# třetí\nText\n# znovu"
	want := "Otázky:\n1. první\n2. druhá\n\n3. třetí\nText\n1. znovu"
	if got := numberLists(in); got != want {
		t.Fatalf("got %q", got)
	}
}
