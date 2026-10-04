import { request } from "./client";
import type {
  Audit,
  CaseAction,
  CaseDetail,
  CasePage,
  CaseRow,
  DashboardResponse,
  Dissent,
  Evidence,
  Graph,
  HealthStatus,
  IngestEvent,
  IngestResult,
  JobReplayResult,
  KnockoutBoard,
  ListEnvelope,
  Message,
  Meta,
  MetricsResponse,
  Note,
  Report,
  RetentionResult,
  Txn,
  User,
} from "./types";

const V1 = "/api/v1";

/**
 * One function per backend route. The route table lives in
 * internal/api/server.go; this file covers all 26 of its entries.
 */
export const endpoints = {
  // --- system -------------------------------------------------------------
  health: () => request<HealthStatus>("/healthz"),
  ready: () => request<HealthStatus>("/readyz"),
  openapi: () => request<string>(`${V1}/openapi.yaml`),

  // --- auth ---------------------------------------------------------------
  login: (email: string, password: string) =>
    request<{ user: User }>(`${V1}/auth/login`, {
      method: "POST",
      body: { email, password },
    }),
  logout: () =>
    request<{ status: string }>(`${V1}/auth/logout`, {
      method: "POST",
      body: {},
    }),
  me: () => request<{ user: User }>(`${V1}/auth/me`),

  // --- product context ----------------------------------------------------
  meta: () => request<Meta>(`${V1}/meta`),
  dashboard: () => request<DashboardResponse>(`${V1}/dashboard`),
  metrics: () => request<MetricsResponse>(`${V1}/metrics`),

  // --- ingestion ----------------------------------------------------------
  ingestEvent: (event: IngestEvent) =>
    request<IngestResult>(`${V1}/events`, { method: "POST", body: event }),
  ingestBatch: (events: IngestEvent[]) =>
    request<{ results: IngestResult[] }>(`${V1}/events/batch`, {
      method: "POST",
      body: { events },
    }),
  transactions: (limit = 50) =>
    request<ListEnvelope<Txn[]>>(`${V1}/transactions`, { query: { limit } }),

  // --- case queue ---------------------------------------------------------
  cases: (params: { status?: string; cursor?: string; limit?: number } = {}) =>
    request<CasePage>(`${V1}/cases`, {
      query: {
        status: params.status,
        cursor: params.cursor,
        limit: params.limit ?? 25,
      },
    }),

  // --- one case -----------------------------------------------------------
  caseDetail: (id: string) => request<CaseDetail>(`${V1}/cases/${id}`),
  addNote: (id: string, body: string) =>
    request<Note>(`${V1}/cases/${id}/notes`, { method: "POST", body: { body } }),
  act: (id: string, action: CaseAction) =>
    request<CaseRow>(`${V1}/cases/${id}/actions`, {
      method: "POST",
      body: action,
    }),
  evidence: (id: string) =>
    request<ListEnvelope<Evidence[]>>(`${V1}/cases/${id}/evidence`),
  reports: (id: string) =>
    request<ListEnvelope<Report[]>>(`${V1}/cases/${id}/reports`),
  createReport: (id: string) =>
    request<Report>(`${V1}/cases/${id}/reports`, { method: "POST", body: {} }),
  graph: (id: string) => request<Graph>(`${V1}/cases/${id}/graph`),
  knockouts: (id: string) => request<KnockoutBoard>(`${V1}/cases/${id}/knockouts`),
  dissent: (id: string) =>
    request<Dissent>(`${V1}/cases/${id}/dissent`, { method: "POST", body: {} }),
  audit: (id: string) => request<ListEnvelope<Audit[]>>(`${V1}/cases/${id}/audit`),
  assistant: (id: string) =>
    request<ListEnvelope<Message[]>>(`${V1}/cases/${id}/assistant`),
  ask: (id: string, question: string) =>
    request<Message>(`${V1}/cases/${id}/assistant`, {
      method: "POST",
      body: { question },
    }),

  // --- operations ---------------------------------------------------------
  replayJob: (id: string) =>
    request<JobReplayResult>(`${V1}/jobs/${id}/replay`, {
      method: "POST",
      body: {},
    }),
  runRetention: () =>
    request<RetentionResult>(`${V1}/admin/retention`, {
      method: "POST",
      body: {},
    }),
};
