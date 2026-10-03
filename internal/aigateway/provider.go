package aigateway

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"
	"time"
)

// Generator is the provider contract. Implementations must not expose tools.
type Generator interface {
	Generate(ctx context.Context, req Request) (Response, error)
}

// Retry wraps transient provider failures. Invalid output is not retried here;
// the caller decides whether the payload is poison.
type Retry struct {
	Next       Generator
	MaxRetries int
}

func (r Retry) Generate(ctx context.Context, req Request) (Response, error) {
	attempts := r.MaxRetries
	if attempts < 0 {
		attempts = 0
	}
	var last error
	for i := 0; i <= attempts; i++ {
		if err := ctx.Err(); err != nil {
			return Response{}, err
		}
		resp, err := r.Next.Generate(ctx, req)
		if err == nil {
			return resp, nil
		}
		last = err
		if !transient(err) || i == attempts {
			break
		}
		delay := time.Duration(200*(1<<i)) * time.Millisecond
		delay += time.Duration(rand.IntN(100)) * time.Millisecond
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return Response{}, ctx.Err()
		case <-timer.C:
		}
	}
	if last == nil {
		last = errors.New("provider failed")
	}
	return Response{}, last
}

func transient(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, hint := range []string{"429", "503", "unavailable", "timeout", "temporarily", "rate limit"} {
		if strings.Contains(msg, hint) {
			return true
		}
	}
	return false
}
