package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"finguard.ai/internal/domain"
	"finguard.ai/internal/store"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Store *store.Store
	TTL   time.Duration
}

var dummyHash []byte

func init() {
	h, err := bcrypt.GenerateFromPassword([]byte("not-a-real-password"), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	dummyHash = h
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func (s *Service) Login(ctx context.Context, email, password string) (string, domain.Principal, error) {
	user, err := s.Store.UserByEmail(ctx, email)
	hash := dummyHash
	if err == nil {
		hash = []byte(user.PasswordHash)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return "", domain.Principal{}, err
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil || err != nil {
		return "", domain.Principal{}, domain.ErrUnauthenticated
	}
	raw, hashHex, err := newToken()
	if err != nil {
		return "", domain.Principal{}, err
	}
	ttl := s.TTL
	if ttl == 0 {
		ttl = 12 * time.Hour
	}
	if err := s.Store.CreateSession(ctx, "ses_"+hashHex[:16], user.ID, hashHex, time.Now().UTC().Add(ttl)); err != nil {
		return "", domain.Principal{}, err
	}
	return raw, domain.Principal{
		UserID: user.ID, TenantID: user.TenantID, Email: user.Email, Name: user.Name, Role: user.Role,
	}, nil
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	return s.Store.DeleteSession(ctx, tokenHash(raw))
}

func (s *Service) FromToken(ctx context.Context, raw string) (domain.Principal, error) {
	if raw == "" {
		return domain.Principal{}, domain.ErrUnauthenticated
	}
	return s.Store.PrincipalByToken(ctx, tokenHash(raw))
}

func newToken() (raw, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(buf)
	return raw, tokenHash(raw), nil
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// Can reports whether a role may perform an action. Viewers are read-only.
func Can(role, action string) bool {
	switch action {
	case "read":
		return role == domain.RoleViewer || role == domain.RoleInvestigator || role == domain.RoleSenior || role == domain.RoleAdmin
	case "ingest", "note", "review", "report", "ask":
		return role == domain.RoleInvestigator || role == domain.RoleSenior || role == domain.RoleAdmin
	case "escalate", "resolve", "close", "replay_job":
		return role == domain.RoleSenior || role == domain.RoleAdmin
	case "retention", "metrics":
		return role == domain.RoleAdmin
	default:
		return false
	}
}
