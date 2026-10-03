package config

import "testing"

func TestRequestRateConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("ENV", "test")
	t.Setenv("RATE_LIMIT_PER_MIN", "")
	c, err := Load()
	if err != nil || c.RateLimitPerMin != 120 {
		t.Fatalf("default: %d %v", c.RateLimitPerMin, err)
	}
	t.Setenv("RATE_LIMIT_PER_MIN", "1200")
	c, err = Load()
	if err != nil || c.RateLimitPerMin != 1200 {
		t.Fatalf("override: %d %v", c.RateLimitPerMin, err)
	}
	for _, v := range []string{"0", "-1", "bad"} {
		t.Setenv("RATE_LIMIT_PER_MIN", v)
		if _, err := Load(); err == nil {
			t.Fatalf("accepted %q", v)
		}
	}
}
