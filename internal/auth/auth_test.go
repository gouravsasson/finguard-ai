package auth

import (
	"testing"

	"finguard.ai/internal/domain"
)

func TestRBAC(t *testing.T) {
	if Can(domain.RoleViewer, "note") || Can(domain.RoleInvestigator, "escalate") || Can(domain.RoleInvestigator, "resolve") {
		t.Fatal("role allowed too much")
	}
	if !Can(domain.RoleInvestigator, "report") || !Can(domain.RoleSenior, "escalate") || !Can(domain.RoleAdmin, "retention") {
		t.Fatal("expected allow")
	}
	if Can("stranger", "read") {
		t.Fatal("unknown role")
	}
}
