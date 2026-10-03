package graph

import (
	"testing"

	"finguard.ai/internal/domain"
)

func TestSharedDeviceIsObservedNotIdentity(t *testing.T) {
	g := Build([]domain.Transaction{{
		ID: "txn_1", AccountID: "acct_1", AccountExternal: "acct_a", SourceID: "s1",
		DeviceID: "dev_1", DeviceExternal: "phone_shared", AmountMinor: 100, Currency: "USD", Status: "approved",
	}}, []domain.Evidence{{
		ID: "ev_1", Kind: "transaction_record", Payload: map[string]any{"source_id": "s1"},
	}, {
		ID: "ev_2", Kind: "device_observation",
		Payload: map[string]any{"device_external_id": "phone_shared", "distinct_accounts": float64(2), "identity_inference": false},
	}})
	for _, e := range g.Edges {
		if e.Kind != "observed" {
			t.Fatalf("edge kind %s", e.Kind)
		}
		if e.Type == "same_person" || e.Type == "identity" {
			t.Fatal(e.Type)
		}
	}
	found := false
	for _, n := range g.Nodes {
		if n.Type == "device" {
			found = true
			if n.Properties["identity_inference"] != false {
				t.Fatal(n.Properties)
			}
		}
	}
	if !found {
		t.Fatal("missing device")
	}
	if len(g.Limitations) == 0 {
		t.Fatal("expected limitations")
	}
}
