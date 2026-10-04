import { describe, expect, it } from "vitest";
import { endpoints } from "./endpoints";

/**
 * Mirrors internal/api/server.go routeTable. If a Go route is added or
 * removed, this list should fail until the client grows the same entry.
 */
const backendRoutes = [
  "GET /healthz",
  "GET /readyz",
  "GET /api/v1/openapi.yaml",
  "POST /api/v1/auth/login",
  "POST /api/v1/auth/logout",
  "GET /api/v1/auth/me",
  "GET /api/v1/meta",
  "GET /api/v1/dashboard",
  "POST /api/v1/events",
  "POST /api/v1/events/batch",
  "GET /api/v1/transactions",
  "GET /api/v1/cases",
  "GET /api/v1/cases/{id}",
  "POST /api/v1/cases/{id}/notes",
  "POST /api/v1/cases/{id}/actions",
  "GET /api/v1/cases/{id}/evidence",
  "GET /api/v1/cases/{id}/reports",
  "POST /api/v1/cases/{id}/reports",
  "GET /api/v1/cases/{id}/graph",
  "GET /api/v1/cases/{id}/knockouts",
  "POST /api/v1/cases/{id}/dissent",
  "GET /api/v1/cases/{id}/audit",
  "GET /api/v1/cases/{id}/assistant",
  "POST /api/v1/cases/{id}/assistant",
  "POST /api/v1/jobs/{id}/replay",
  "POST /api/v1/admin/retention",
  "GET /api/v1/metrics",
] as const;

const clientKeys = [
  "health",
  "ready",
  "openapi",
  "login",
  "logout",
  "me",
  "meta",
  "dashboard",
  "ingestEvent",
  "ingestBatch",
  "transactions",
  "cases",
  "caseDetail",
  "addNote",
  "act",
  "evidence",
  "reports",
  "createReport",
  "graph",
  "knockouts",
  "dissent",
  "audit",
  "assistant",
  "ask",
  "replayJob",
  "runRetention",
  "metrics",
] as const;

describe("API client coverage", () => {
  it("has one helper per backend route", () => {
    expect(clientKeys).toHaveLength(backendRoutes.length);
    expect(Object.keys(endpoints).sort()).toEqual([...clientKeys].sort());
  });
});
