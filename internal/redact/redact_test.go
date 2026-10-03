package redact

import "testing"

func TestText(t *testing.T) {
	in := "card 4111111111111111 mail ada@example.test token sk-live-abcdef1234"
	out := Text(in)
	if out == in {
		t.Fatal(out)
	}
	for _, banned := range []string{"4111111111111111", "ada@example.test", "sk-live-abcdef1234"} {
		if contains(out, banned) {
			t.Fatalf("leaked %s in %s", banned, out)
		}
	}
	if Text("latte for the office") != "latte for the office" {
		t.Fatal("plain memo changed")
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
