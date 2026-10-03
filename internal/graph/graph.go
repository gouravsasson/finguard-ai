package graph

import (
	"context"
	"fmt"
	"sort"

	"finguard.ai/internal/domain"
	"finguard.ai/internal/store"
)

type Service struct {
	Store *store.Store
}

func (s *Service) ForCase(ctx context.Context, tenantID, caseID string) (domain.Graph, error) {
	sc, err := s.Store.Begin(ctx, tenantID)
	if err != nil {
		return domain.Graph{}, err
	}
	defer sc.Rollback()
	if _, err := sc.GetCase(ctx, caseID); err != nil {
		return domain.Graph{}, err
	}
	txns, err := sc.CaseTransactions(ctx, caseID)
	if err != nil {
		return domain.Graph{}, err
	}
	ev, err := sc.ListEvidence(ctx, caseID)
	if err != nil {
		return domain.Graph{}, err
	}
	if err := sc.Commit(ctx); err != nil {
		return domain.Graph{}, err
	}
	return Build(txns, ev), nil
}

// Build draws only observed edges. A shared device is labeled as an observation
// and is never an identity edge.
func Build(txns []domain.Transaction, evidence []domain.Evidence) domain.Graph {
	g := domain.Graph{
		Limitations: []string{
			"Edges are observed relationships from ingested synthetic events.",
			"A shared device is not proof that accounts belong to the same person.",
			"The graph does not infer identity.",
		},
	}
	nodes := map[string]domain.GraphNode{}
	var edges []domain.GraphEdge
	addNode := func(n domain.GraphNode) {
		if n.ID == "" {
			return
		}
		if _, ok := nodes[n.ID]; ok {
			return
		}
		n.Kind = "observed"
		nodes[n.ID] = n
	}
	evidenceBySource := map[string]string{}
	deviceCounts := map[string]int{}
	for _, item := range evidence {
		if item.Kind == "transaction_record" {
			if src, ok := item.Payload["source_id"].(string); ok {
				evidenceBySource[src] = item.ID
			}
		}
		if item.Kind == "device_observation" {
			dev, _ := item.Payload["device_external_id"].(string)
			switch n := item.Payload["distinct_accounts"].(type) {
			case float64:
				deviceCounts[dev] = int(n)
			case int:
				deviceCounts[dev] = n
			}
		}
	}
	for _, tx := range txns {
		addNode(domain.GraphNode{ID: tx.AccountID, Type: "account", Label: tx.AccountExternal})
		addNode(domain.GraphNode{ID: tx.ID, Type: "transaction", Label: tx.SourceID, Properties: map[string]any{
			"amount_minor": tx.AmountMinor, "currency": tx.Currency, "status": tx.Status,
		}})
		evIDs := []string{}
		if id := evidenceBySource[tx.SourceID]; id != "" {
			evIDs = append(evIDs, id)
		}
		edges = append(edges, domain.GraphEdge{
			ID: "e:" + tx.AccountID + ":" + tx.ID, From: tx.AccountID, To: tx.ID, Type: "initiated",
			Kind: "observed", Provenance: "transaction.account_id", EvidenceIDs: evIDs,
		})
		if tx.MerchantID != "" {
			addNode(domain.GraphNode{ID: tx.MerchantID, Type: "merchant", Label: tx.MerchantName})
			edges = append(edges, domain.GraphEdge{
				ID: "e:" + tx.ID + ":" + tx.MerchantID, From: tx.ID, To: tx.MerchantID, Type: "at_merchant",
				Kind: "observed", Provenance: "transaction.merchant_id", EvidenceIDs: evIDs,
			})
		}
		if tx.DeviceID != "" {
			label := tx.DeviceExternal
			props := map[string]any{"identity_inference": false}
			if n, ok := deviceCounts[tx.DeviceExternal]; ok {
				props["distinct_accounts"] = n
				label = fmt.Sprintf("%s (%d accounts observed)", tx.DeviceExternal, n)
			}
			addNode(domain.GraphNode{ID: tx.DeviceID, Type: "device", Label: label, Properties: props})
			edges = append(edges, domain.GraphEdge{
				ID: "e:" + tx.ID + ":" + tx.DeviceID, From: tx.ID, To: tx.DeviceID, Type: "used_device",
				Kind: "observed", Provenance: "transaction.device_id", EvidenceIDs: evIDs,
			})
		}
		if tx.LocationID != "" {
			addNode(domain.GraphNode{ID: tx.LocationID, Type: "location", Label: tx.LocationLabel})
			edges = append(edges, domain.GraphEdge{
				ID: "e:" + tx.ID + ":" + tx.LocationID, From: tx.ID, To: tx.LocationID, Type: "at_location",
				Kind: "observed", Provenance: "transaction.location_id", EvidenceIDs: evIDs,
			})
		}
	}
	for _, n := range nodes {
		g.Nodes = append(g.Nodes, n)
	}
	sort.Slice(g.Nodes, func(i, j int) bool {
		if g.Nodes[i].Type == g.Nodes[j].Type {
			return g.Nodes[i].ID < g.Nodes[j].ID
		}
		return g.Nodes[i].Type < g.Nodes[j].Type
	})
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	g.Edges = edges
	return g
}
