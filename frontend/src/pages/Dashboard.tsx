import { useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { DeskOps } from "../components/DeskOps";
import { qk } from "@/lib/api/keys";
import { useCaseQueue, useDashboard, useMeta } from "@/lib/api/hooks";
import { useUiStore } from "@/store/ui";

const statuses = [
  "all",
  "open",
  "in_review",
  "escalated",
  "resolved",
  "closed",
];
const labels: Record<string, string> = {
  all: "All cases",
  open: "Open",
  in_review: "In review",
  escalated: "Escalated",
  resolved: "Resolved",
  closed: "Closed",
};

export function Dashboard() {
  const summary = useDashboard();
  const meta = useMeta();
  const queue = useCaseQueue({ limit: 25 });
  const client = useQueryClient();
  const { status, search, priority } = useUiStore((s) => s.queue);
  const setStatus = useUiStore((s) => s.setQueueStatus);
  const setSearch = useUiStore((s) => s.setQueueSearch);
  const setPriority = useUiStore((s) => s.setQueuePriority);
  const clearFilters = useUiStore((s) => s.clearQueueFilters);

  const data = summary.data;
  const loaded = queue.data?.pages.flatMap((page) => page.data) ?? [];
  const error =
    summary.error?.message ||
    queue.error?.message ||
    (summary.isError || queue.isError ? "Unable to load cases." : "");
  const refreshing = summary.isFetching || queue.isFetching;
  const loadingMore = queue.isFetchingNextPage;
  const cursor = queue.hasNextPage;

  const cases = loaded.filter(
    (item) =>
      (status === "all" || item.status === status) &&
      (priority === "all" || item.priority === priority) &&
      `${item.title} ${item.id} ${item.account_external_id || ""}`
        .toLowerCase()
        .includes(search.toLowerCase()),
  );
  const counts = data?.counts || {};
  const metrics = [
    { key: "open", icon: "folder", note: "Awaiting investigation" },
    { key: "in_review", icon: "eye", note: "Under active review" },
    { key: "escalated", icon: "alert", note: "Requires senior attention" },
    { key: "resolved", icon: "check", note: "Review completed" },
  ] as const;

  async function refresh() {
    await Promise.all([
      client.invalidateQueries({ queryKey: qk.dashboard }),
      client.invalidateQueries({ queryKey: ["cases"] }),
    ]);
  }

  return (
    <>
      <div className="page-heading">
        <div>
          <h1>Case queue</h1>
          <p className="muted">
            Open work for this tenant. Triage counts signals. It is not a fraud
            score.
          </p>
        </div>
        <button
          className="secondary"
          disabled={refreshing || loadingMore}
          onClick={() => void refresh()}
        >
          <Icon name="refresh" size={16} />
          {refreshing && loaded.length > 0 ? "Refreshing…" : "Refresh queue"}
        </button>
      </div>
      {error && (
        <p className="error" role="alert">
          {error}{" "}
          <button className="secondary" onClick={() => void refresh()}>
            Try again
          </button>
        </p>
      )}
      <div className="desk-metrics">
        <div className="queue-overview">
          <div>
            <h2>Counts</h2>
            <p>Click a count to filter the list below.</p>
          </div>
          <div className="overview-facts">
            <span>
              <strong>
                {data
                  ? Object.values(counts).reduce((sum, n) => sum + n, 0)
                  : "—"}
              </strong>
              Total cases
            </span>
            <span>
              <strong>
                {data
                  ? (counts.open || 0) +
                    (counts.in_review || 0) +
                    (counts.escalated || 0)
                  : "—"}
              </strong>
              Awaiting a decision
            </span>
          </div>
        </div>
        <section className="grid stats" aria-label="Queue counts">
          {metrics.map((m) => (
            <article
              className={`stat ${m.key} ${status === m.key ? "is-selected" : ""}`}
              key={m.key}
            >
              <b>{data ? counts[m.key] || 0 : "—"}</b>
              <span className="stat-name">{labels[m.key]}</span>
              <small>{m.note}</small>
              <button
                className="stat-filter"
                type="button"
                aria-label={`Filter ${labels[m.key].toLowerCase()} cases`}
                aria-pressed={status === m.key}
                onClick={() => setStatus(status === m.key ? "all" : m.key)}
              />
            </article>
          ))}
        </section>
      </div>

      <div className="context-strip">
        <Icon name="shield" size={18} />
        <p>
          <strong>Synthetic data workspace</strong>
          <span>
            Signals guide review; triage bands are not a fraud probability.
          </span>
        </p>
        <span className="context-mode">
          {(meta.data || data?.meta)
            ? (meta.data || data!.meta).ai_live
              ? "Live Gemini"
              : `${(meta.data || data!.meta).ai_provider} · Not live`
            : "Loading model status…"}
        </span>
      </div>
      <section className="card queue-card" aria-labelledby="queue-title">
        <div className="section-heading">
          <div>
            <h2 id="queue-title">
              Investigation queue{" "}
              <span className="count">{loaded.length || "—"}</span>
            </h2>
            <p className="muted">
              Search loaded cases or filter by status and triage priority.
            </p>
          </div>
          <span className="small-muted">
            <Icon name="clock" size={15} /> {data?.queue_depth ?? "—"} report
            jobs waiting
          </span>
        </div>
        <div className="queue-controls">
          <div className="tabs" aria-label="Filter by case status">
            {statuses.map((s) => (
              <button
                key={s}
                className={status === s ? "active" : ""}
                aria-pressed={status === s}
                onClick={() => setStatus(s)}
              >
                {labels[s]}
              </button>
            ))}
          </div>
          <div className="filters">
            <div className="search-field">
              <Icon name="search" size={18} />
              <input
                aria-label="Search cases"
                placeholder="Search displayed cases…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            <select
              aria-label="Filter by priority"
              value={priority}
              onChange={(e) => setPriority(e.target.value)}
            >
              <option value="all">All priorities</option>
              {Array.from(new Set(loaded.map((c) => c.priority)))
                .sort()
                .map((p) => (
                  <option key={p} value={p}>
                    {p.charAt(0).toUpperCase() + p.slice(1)} priority
                  </option>
                ))}
            </select>
          </div>
        </div>
        {!data && !error && queue.isPending ? (
          <div className="empty-state" role="status">
            <Icon name="clock" size={28} />
            <h3>Loading your workspace</h3>
            <p>Gathering the latest cases and signals…</p>
          </div>
        ) : !data && error ? (
          <div className="empty-state">
            <h3>Queue unavailable</h3>
            <p>Try again to reload your cases.</p>
          </div>
        ) : cases.length === 0 ? (
          <div className="empty-state">
            <Icon name="folder" size={32} />
            <h3>
              {loaded.length ? "No matching cases" : "Your queue is clear"}
            </h3>
            <p>
              {loaded.length
                ? "Try another search or adjust your filters."
                : "Cases will appear here when synthetic activity triggers a signal."}
            </p>
            {(search || status !== "all" || priority !== "all") && (
              <button className="secondary" onClick={clearFilters}>
                Clear filters
              </button>
            )}
          </div>
        ) : (
          <div className="table-scroll">
            <table className="case-table">
              <thead>
                <tr>
                  <th>Case / account</th>
                  <th>Triage</th>
                  <th>Status</th>
                  <th>Signals</th>
                  <th>Opened</th>
                  <th>
                    <span className="sr-only">Open case</span>
                  </th>
                </tr>
              </thead>
              <tbody>
                {cases.map((item) => (
                  <tr key={item.id}>
                    <td>
                      <Link className="case-title" to={`/cases/${item.id}`}>
                        <span className={`case-icon ${item.priority}`}>
                          <Icon name="folder" size={18} />
                        </span>
                        <span>
                          <strong>{item.title}</strong>
                          <small>{item.account_external_id || item.id}</small>
                        </span>
                      </Link>
                    </td>
                    <td>
                      <span className={`priority ${item.priority}`}>
                        {item.signal_count}{" "}
                        {item.signal_count === 1 ? "signal" : "signals"} ·{" "}
                        {item.priority}
                      </span>
                    </td>
                    <td>
                      <span className={`tag ${item.status}`}>
                        <i />
                        {item.status.replaceAll("_", " ")}
                      </span>
                    </td>
                    <td>
                      <span className="signal-count">{item.signal_count}</span>
                    </td>
                    <td className="date-cell">
                      {Number.isNaN(Date.parse(item.opened_at))
                        ? "—"
                        : new Date(item.opened_at).toLocaleDateString("en-US", {
                            month: "short",
                            day: "numeric",
                            year: "numeric",
                          })}
                    </td>
                    <td>
                      <Link
                        className="open-case"
                        to={`/cases/${item.id}`}
                        aria-label={`Open ${item.title}`}
                      >
                        <Icon name="arrow" size={18} />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        {cursor && (
          <div className="load-more">
            <button
              className="secondary"
              disabled={loadingMore || refreshing}
              onClick={() => void queue.fetchNextPage()}
            >
              {loadingMore ? "Loading…" : "Load more cases"}
            </button>
            <span>Search and filters apply to loaded cases.</span>
          </div>
        )}
        <div className="table-footer">
          <span>
            Showing {cases.length} of {loaded.length} loaded cases
          </span>
          <span>Synthetic records only</span>
        </div>
      </section>
      <DeskOps />
      <div className="desk-note">
        <Icon name="spark" size={18} />
        <p>
          {meta.data?.limitations[0] ||
            data?.meta.limitations[0] ||
            "Synthetic demonstration data only."}
        </p>
      </div>
    </>
  );
}
