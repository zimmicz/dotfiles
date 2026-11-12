package main

import "testing"

func TestNumberLists(t *testing.T) {
	in := "Otázky:\n# *první*\npokračování\n➡️ doporučení\n# druhá\n\n# třetí\n\nText\n# znovu"
	want := "Otázky:\n1. *první*\npokračování\n➡️ doporučení\n2. druhá\n\n3. třetí\n\nText\n1. znovu"
	if got := numberLists(in); got != want {
		t.Fatalf("got %q", got)
	}
}
