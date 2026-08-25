package main

import "testing"

func TestBusinessChain46(t *testing.T) {
	first, second := ExecuteBusinessChain("first-ok", "second-ok")
	if first.Value != "first-ok" || second.Value != "second-ok" {
		t.Fatalf("each operation must retain its own result: %#v %#v", first, second)
	}
}
