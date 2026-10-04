/**
 * Query keys. Case-scoped keys nest under ["case", id] so a single
 * invalidation after an action refreshes every panel for that case.
 */
export const qk = {
  session: ["session"] as const,
  meta: ["meta"] as const,
  dashboard: ["dashboard"] as const,
  metrics: ["metrics"] as const,
  health: ["health"] as const,
  ready: ["ready"] as const,
  transactions: (limit: number) => ["transactions", limit] as const,

  cases: (filters: { status?: string; limit?: number } = {}) =>
    ["cases", filters.status ?? "all", filters.limit ?? 25] as const,

  case: (id: string) => ["case", id] as const,
  caseDetail: (id: string) => ["case", id, "detail"] as const,
  caseEvidence: (id: string) => ["case", id, "evidence"] as const,
  caseReports: (id: string) => ["case", id, "reports"] as const,
  caseGraph: (id: string) => ["case", id, "graph"] as const,
  caseKnockouts: (id: string) => ["case", id, "knockouts"] as const,
  caseAudit: (id: string) => ["case", id, "audit"] as const,
  caseAssistant: (id: string) => ["case", id, "assistant"] as const,
};
