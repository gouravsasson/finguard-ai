package jobs

import (
	"testing"
	"time"
)

func TestBackoffBounded(t *testing.T) {
	if Backoff(1) != time.Second {
		t.Fatal(Backoff(1))
	}
	if Backoff(10) != 30*time.Second {
		t.Fatal(Backoff(10))
	}
	if Backoff(3) != 4*time.Second {
		t.Fatal(Backoff(3))
	}
}
