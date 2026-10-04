import { FormEvent, useMemo, useState } from "react";
import {
  useHealth,
  useIngestBatch,
  useIngestEvent,
  useMetrics,
  useReadiness,
  useReplayJob,
  useRetention,
  useTransactions,
} from "@/lib/api/hooks";
import type { IngestEvent } from "@/lib/api/types";
import { useSessionStore } from "@/store/session";
import { money } from "@/lib/claims";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";

function sampleEvent(): IngestEvent {
  const stamp = Date.now();
  return {
    idempotency_key: `desk_${stamp}`,
    source_system: "northwind_core",
    source_id: `txn_desk_${stamp}`,
    schema_version: "1",
    occurred_at: new Date().toISOString(),
    account_external_id: "acct_demo",
    account_label: "Northwind demo account",
    merchant: {
      external_id: "merch_desk",
      name: "Harbor Supply",
      category: "retail",
    },
    amount_minor: 4200,
    currency: "USD",
    status: "approved",
    memo: "Desk-entered synthetic event. No money moves.",
  };
}

/**
 * Surfaces the remaining backend routes on the queue for roles that
 * can call them. The case file stays the investigation desk; this is
 * the operations strip, not a second product.
 */
export function DeskOps() {
  const can = useSessionStore((s) => s.can);
  const canRead = can("read");
  const canIngest = can("ingest");
  const canReplay = can("replay_job");
  const canAdmin = can("metrics") || can("retention");
  const health = useHealth(canRead);
  const ready = useReadiness(canRead);
  const metrics = useMetrics(can("metrics"));
  const txns = useTransactions(8, canRead);
  const ingestOne = useIngestEvent();
  const ingestMany = useIngestBatch();
  const replay = useReplayJob();
  const retention = useRetention();
  const [payload, setPayload] = useState(() =>
    JSON.stringify(sampleEvent(), null, 2),
  );
  const [jobId, setJobId] = useState("");
  const [note, setNote] = useState("");
  const [error, setError] = useState("");

  const process = useMemo(() => {
    const snap = metrics.data?.metrics;
    if (!snap) return null;
    return [
      ["requests", snap.requests],
      ["accepted", snap.ingest_accepted],
      ["replayed", snap.ingest_replay],
      ["quarantined", snap.ingest_quarantine],
      ["reports ready", snap.reports_ready],
      ["reports failed", snap.reports_failed],
      ["queue depth", snap.queue_depth],
    ] as const;
  }, [metrics.data]);

  async function run(fn: () => Promise<unknown>, ok: string) {
    setError("");
    setNote("");
    try {
      await fn();
      setNote(ok);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }

  function parseEvent(): IngestEvent {
    const raw = JSON.parse(payload) as IngestEvent;
    if (!raw || typeof raw !== "object") throw new Error("Event must be JSON");
    return raw;
  }

  async function submitOne(e: FormEvent) {
    e.preventDefault();
    const event = parseEvent();
    await run(
      () => ingestOne.mutateAsync(event),
      `Event ${event.idempotency_key} sent. The queue will refresh if a signal fired.`,
    );
  }

  async function submitBatch() {
    const event = parseEvent();
    await run(
      () => ingestMany.mutateAsync([event]),
      `Batch of 1 sent for ${event.idempotency_key}.`,
    );
  }

  if (!canRead) return null;

  return (
    <section className="card queue-card" aria-labelledby="ops-title">
      <div className="section-heading">
        <div>
          <h2 id="ops-title">Desk operations</h2>
          <p className="muted">
            Routes that sit beside the case file: health, recent ledger rows,
            ingest, job replay, and admin counters. Synthetic data only.
          </p>
        </div>
        <span className="small-muted">
          API {health.data?.status || "…"} · store {ready.data?.status || "…"}
        </span>
      </div>

      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {note && (
        <p className="success" role="status">
          {note}
        </p>
      )}

      {txns.data && txns.data.length > 0 && (
        <div className="table-scroll">
          <table className="transaction-table">
            <caption className="sr-only">Recent tenant transactions</caption>
            <thead>
              <tr>
                <th>Merchant / date</th>
                <th>Amount</th>
                <th>Account</th>
              </tr>
            </thead>
            <tbody>
              {txns.data.map((tx) => (
                <tr key={tx.id}>
                  <td>
                    <strong>{tx.merchant_name || "No merchant"}</strong>
                    <small>{tx.occurred_at.slice(0, 10)}</small>
                  </td>
                  <td className="amount">
                    {money(tx.amount_minor, tx.currency)}
                  </td>
                  <td>
                    <span className="mono">
                      {tx.account_external_id || tx.id}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {canIngest && (
        <form className="grid" onSubmit={submitOne}>
          <Label htmlFor="ingest-json">Synthetic event JSON</Label>
          <Textarea
            id="ingest-json"
            rows={10}
            value={payload}
            onChange={(e) => setPayload(e.target.value)}
            spellCheck={false}
          />
          <div className="actions">
            <Button type="submit" disabled={ingestOne.isPending}>
              {ingestOne.isPending ? "Sending…" : "Ingest event"}
            </Button>
            <Button
              type="button"
              variant="secondary"
              className="secondary"
              disabled={ingestMany.isPending}
              onClick={() => void submitBatch()}
            >
              {ingestMany.isPending ? "Sending…" : "Ingest as batch"}
            </Button>
          </div>
        </form>
      )}

      {canReplay && (
        <form
          className="filters"
          onSubmit={(e) => {
            e.preventDefault();
            if (!jobId.trim()) {
              setError("A job id is required to replay.");
              return;
            }
            void run(
              () => replay.mutateAsync(jobId.trim()),
              `Job ${jobId.trim()} queued as pending.`,
            );
          }}
        >
          <Label htmlFor="job-id">Failed job id</Label>
          <Input
            id="job-id"
            value={jobId}
            onChange={(e) => setJobId(e.target.value)}
            placeholder="job_…"
          />
          <Button type="submit" variant="secondary" className="secondary">
            {replay.isPending ? "Replaying…" : "Replay job"}
          </Button>
        </form>
      )}

      {canAdmin && (
        <div className="overview-facts">
          {(process || []).map(([label, value]) => (
            <span key={label}>
              <strong>{value ?? "—"}</strong>
              {label}
            </span>
          ))}
          {metrics.data?.note && <p className="muted">{metrics.data.note}</p>}
          {can("retention") && (
            <Button
              type="button"
              variant="secondary"
              className="secondary"
              disabled={retention.isPending}
              onClick={() =>
                void run(
                  () => retention.mutateAsync(),
                  "Expired assistant messages were purged for this tenant.",
                )
              }
            >
              {retention.isPending ? "Purging…" : "Run assistant retention"}
            </Button>
          )}
        </div>
      )}
      <p className="table-footer">
        <span>Machine-readable contract</span>
        <a href="/api/v1/openapi.yaml">OpenAPI spec</a>
      </p>
    </section>
  );
}
