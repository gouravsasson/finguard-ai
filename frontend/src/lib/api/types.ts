/**
 * Mirrors the Go types in internal/domain and the service packages.
 * Kept hand-written so the field names stay readable at the call site.
 */

export type Role = "viewer" | "investigator" | "senior_investigator" | "admin";

export type CaseStatus =
  | "open"
  | "in_review"
  | "escalated"
  | "resolved"
  | "closed";

export type TriageBand = "low" | "medium" | "high";

export type User = {
  id: string;
  email: string;
  name: string;
  role: string;
  tenant_id: string;
};

export type CaseRow = {
  id: string;
  status: string;
  priority: string;
  title: string;
  version: number;
  signal_count: number;
  account_external_id?: string;
  assignee_id?: string;
  opened_at: string;
  updated_at?: string;
  resolved_at?: string;
  resolution?: string;
  rationale?: string;
};

export type Signal = {
  id: string;
  rule_code: string;
  rule_version: string;
  explanation: string;
  threshold: string;
  observed_value: string;
  evidence_ids: string[];
  transaction_id: string;
  inputs?: Record<string, unknown>;
  created_at?: string;
};

export type Txn = {
  id: string;
  source_id: string;
  source_system?: string;
  amount_minor: number;
  currency: string;
  status: string;
  occurred_at: string;
  account_external_id?: string;
  merchant_name?: string;
  device_external_id?: string;
  location_label?: string;
  location_country?: string;
  memo?: string;
  schema_version?: string;
};

export type Evidence = {
  id: string;
  kind: string;
  source: string;
  classification?: string;
  content_hash: string;
  payload: {
    summary?: string;
    identity_inference?: boolean;
    distinct_accounts?: number;
  };
  provenance: Record<string, string>;
  created_at: string;
  retention_until?: string;
};

export type Fact = { claim: string; evidence_ids?: string[] };
export type Hypothesis = {
  claim: string;
  evidence_ids?: string[];
  uncertainty?: string;
};

export type ReportBody = {
  summary?: string;
  observed_facts?: Fact[];
  hypotheses?: Hypothesis[];
  alternative_explanations?: Hypothesis[];
  missing_evidence?: string[];
  suggested_review_steps?: string[];
  limitations?: string[];
  unverified_claims?: { claim: string; reason: string }[];
  validation_status?: string;
  model_metadata?: {
    provider?: string;
    live?: boolean;
    model?: string;
    prompt_version?: string;
  };
};

export type Report = {
  id: string;
  status: string;
  validation_status: string;
  provider: string;
  model: string;
  prompt_version: string;
  live: boolean;
  report: ReportBody;
  latency_ms?: number;
  input_tokens?: number;
  output_tokens?: number;
  created_at?: string;
};

export type GraphNode = {
  id: string;
  type: string;
  label: string;
  kind: string;
  properties?: Record<string, unknown>;
};

export type GraphEdge = {
  id: string;
  from: string;
  to: string;
  type: string;
  kind: string;
  provenance: string;
  evidence_ids: string[];
};

export type Graph = {
  nodes: GraphNode[];
  edges: GraphEdge[];
  limitations: string[];
};

export type Audit = {
  id: string;
  actor_label: string;
  action: string;
  rationale?: string;
  created_at: string;
  entity_type?: string;
  entity_id?: string;
  metadata?: Record<string, unknown>;
};

export type KnockoutAttempt = {
  kind: string;
  title: string;
  detail: string;
  clears: boolean;
  transaction_ids: string[];
  evidence_ids?: string[];
};

export type KnockoutSignal = {
  signal_id: string;
  rule_code: string;
  explanation: string;
  reproduced: boolean;
  note?: string;
  attempts: KnockoutAttempt[];
};

export type KnockoutBoard = {
  method: string;
  rule_version: string;
  limitation: string;
  signals: KnockoutSignal[];
  reading: {
    concern: string;
    concern_evidence: string[];
    benign: string;
    benign_evidence: string[];
    would_change: string[];
    change_evidence: string[];
  };
};

export type Dissent = {
  provider: string;
  model: string;
  live: boolean;
  validation_status: string;
  concern: Fact;
  benign: Fact;
  what_would_change: Fact[];
  limitations: string[];
};

export type Note = {
  id: string;
  author_id: string;
  body: string;
  created_at: string;
};

export type Message = {
  id: string;
  role: string;
  content: string;
  citations: string[];
  refused: boolean;
  created_at?: string;
};

/** GET /api/v1/meta */
export type Meta = {
  product: string;
  synthetic_only: boolean;
  adverse_actions: boolean;
  ai_provider: string;
  ai_live: boolean;
  model: string;
  prompt_version: string;
  limitations: string[];
};

/** GET /api/v1/dashboard */
export type DashboardResponse = {
  counts: Record<string, number>;
  cases: CaseRow[];
  queue_depth: number;
  meta: Meta;
};

/** GET /api/v1/cases */
export type CasePage = {
  data: CaseRow[];
  page: { next_cursor: string; limit: number };
};

/** GET /api/v1/cases/{id} */
export type CaseDetail = {
  case: CaseRow;
  signals: Signal[];
  transactions: Txn[];
  notes: Note[];
};

/** POST /api/v1/cases/{id}/actions */
export type CaseAction = {
  action: "assign" | "start_review" | "escalate" | "resolve" | "close";
  rationale?: string;
  version: number;
};

/** POST /api/v1/events — mirrors ingestion.Event in internal/ingestion/validate.go */
export type IngestEvent = {
  idempotency_key: string;
  source_system: string;
  source_id: string;
  schema_version: string;
  occurred_at: string;
  account_external_id: string;
  account_label?: string;
  merchant?: {
    external_id: string;
    name?: string;
    category?: string;
    country?: string;
  } | null;
  device?: { external_id: string } | null;
  location?: { country: string; region?: string; city?: string } | null;
  amount_minor: number;
  currency: string;
  status: "approved" | "declined" | "pending";
  memo?: string;
};

export type IngestResult = {
  idempotency_key: string;
  status: "accepted" | "replayed" | "quarantined" | "conflict" | "error";
  transaction_id?: string;
  case_id?: string;
  signal_codes?: string[];
  quarantine_reason?: string;
  raw_event_id?: string;
};

/** GET /api/v1/metrics */
export type MetricsResponse = {
  metrics: {
    requests: number;
    ingest_accepted: number;
    ingest_replay: number;
    ingest_quarantine: number;
    reports_ready: number;
    reports_failed: number;
    ai_calls: number;
    ai_latency_avg_ms: number;
    queue_depth: number;
  };
  note: string;
};

export type HealthStatus = { status: string };
export type RetentionResult = { deleted: number };
export type JobReplayResult = { status: string };

/** Envelope the backend uses for every list route. */
export type ListEnvelope<T> = { data: T };
