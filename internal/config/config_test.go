package config

import "testing"

func TestParseTokenPrefixes(t *testing.T) {
	got, err := ParseTokenPrefixes(" tokA:alice- ; tokB:bob-;")
	if err != nil || len(got) != 2 || got["tokA"] != "alice-" || got["tokB"] != "bob-" {
		t.Fatalf("got %v, %v", got, err)
	}
	if got, err := ParseTokenPrefixes("  "); got != nil || err != nil {
		t.Fatalf("empty: %v, %v", got, err)
	}
	for _, bad := range []string{
		"tok",               // no prefix
		"tok:",              // empty prefix
		":prefix",           // empty token
		"a:x-;a:y-",         // duplicate token
		"a:bob;b:bob-extra", // overlapping namespaces
	} {
		if _, err := ParseTokenPrefixes(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}
