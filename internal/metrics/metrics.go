package metrics

import (
	"sync/atomic"
	"time"
)

type Metrics struct {
	Requests         atomic.Int64
	IngestAccepted   atomic.Int64
	IngestReplay     atomic.Int64
	IngestQuarantine atomic.Int64
	ReportsReady     atomic.Int64
	ReportsFailed    atomic.Int64
	AILatencyMS      atomic.Int64
	AICalls          atomic.Int64
}

func (m *Metrics) ObserveAI(d time.Duration, ok bool) {
	m.AICalls.Add(1)
	m.AILatencyMS.Add(d.Milliseconds())
	if ok {
		m.ReportsReady.Add(1)
	} else {
		m.ReportsFailed.Add(1)
	}
}

func (m *Metrics) Snapshot() map[string]int64 {
	calls := m.AICalls.Load()
	var avg int64
	if calls > 0 {
		avg = m.AILatencyMS.Load() / calls
	}
	return map[string]int64{
		"requests":          m.Requests.Load(),
		"ingest_accepted":   m.IngestAccepted.Load(),
		"ingest_replay":     m.IngestReplay.Load(),
		"ingest_quarantine": m.IngestQuarantine.Load(),
		"reports_ready":     m.ReportsReady.Load(),
		"reports_failed":    m.ReportsFailed.Load(),
		"ai_calls":          calls,
		"ai_latency_avg_ms": avg,
	}
}
