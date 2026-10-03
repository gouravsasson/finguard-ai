package evidence

import (
	"context"
	"fmt"
	"sort"
	"time"

	"finguard.ai/internal/aigateway"
	"finguard.ai/internal/idgen"
	"finguard.ai/internal/store"
)

const MaxItems = 40

func ID(tenant, stableKey string) string {
	return idgen.Stable("ev_", tenant+"|"+stableKey)
}

func Build(ctx context.Context, sc *store.Scope, caseID string) (aigateway.Package, error) {
	items, err := sc.ListEvidence(ctx, caseID)
	if err != nil {
		return aigateway.Package{}, err
	}
	signals, err := sc.ListSignals(ctx, caseID)
	if err != nil {
		return aigateway.Package{}, err
	}
	pkg := aigateway.Package{CaseID: caseID}
	for _, sig := range signals {
		pkg.Signals = append(pkg.Signals, aigateway.SignalView{
			RuleCode: sig.RuleCode, RuleVersion: sig.RuleVersion, Explanation: sig.Explanation,
			Threshold: sig.Threshold, ObservedValue: sig.ObservedValue, EvidenceIDs: sig.EvidenceIDs,
		})
	}
	for _, item := range items {
		summary, _ := item.Payload["summary"].(string)
		if summary == "" {
			summary = item.Kind
		}
		var when time.Time
		if raw, ok := item.Payload["occurred_at"].(string); ok {
			when, _ = time.Parse(time.RFC3339, raw)
		}
		pkg.Items = append(pkg.Items, aigateway.EvidenceItem{ID: item.ID, Kind: item.Kind, Summary: summary, OccurredAt: when})
		if item.Kind == "device_observation" {
			if dev, ok := item.Payload["device_external_id"].(string); ok && dev != "" {
				pkg.Devices = append(pkg.Devices, dev)
			}
		}
	}
	if len(pkg.Items) > MaxItems {
		pkg.Items = pkg.Items[:MaxItems]
		pkg.Truncated = true
		pkg.Limitations = append(pkg.Limitations, fmt.Sprintf("Evidence package limited to %d records.", MaxItems))
	}
	sort.Strings(pkg.Devices)
	return pkg, nil
}
