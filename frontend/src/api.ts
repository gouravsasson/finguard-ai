export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("X-FinGuard-Request", "1");
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const res = await fetch(path, { ...init, headers, credentials: "include" });
  const text = await res.text();
  const data = text ? JSON.parse(text) : {};
  if (!res.ok) {
    const message = data?.error?.message || res.statusText;
    throw new ApiError(res.status, message);
  }
  return data as T;
}

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
};

export type Txn = {
  id: string;
  source_id: string;
  amount_minor: number;
  currency: string;
  status: string;
  occurred_at: string;
  merchant_name?: string;
  device_external_id?: string;
  location_label?: string;
  memo?: string;
};

export type Evidence = {
  id: string;
  kind: string;
  source: string;
  content_hash: string;
  payload: { summary?: string; identity_inference?: boolean; distinct_accounts?: number };
  provenance: Record<string, string>;
  created_at: string;
};

export type Fact = { claim: string; evidence_ids?: string[] };
export type Hypothesis = { claim: string; evidence_ids?: string[]; uncertainty?: string };

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
  model_metadata?: { provider?: string; live?: boolean; model?: string; prompt_version?: string };
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
};

export type Graph = {
  nodes: { id: string; type: string; label: string; kind: string; properties?: Record<string, unknown> }[];
  edges: { id: string; from: string; to: string; type: string; kind: string; provenance: string; evidence_ids: string[] }[];
  limitations: string[];
};

export type Audit = {
  id: string;
  actor_label: string;
  action: string;
  rationale?: string;
  created_at: string;
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

export type Note = { id: string; author_id: string; body: string; created_at: string };
export type Message = { id: string; role: string; content: string; citations: string[]; refused: boolean };
