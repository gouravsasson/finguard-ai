import { FormEvent, useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import {
  api,
  Audit,
  CaseRow,
  Dissent,
  Evidence,
  Graph,
  KnockoutBoard,
  Message,
  Note,
  Report,
  Signal,
  Txn,
} from "../api";
import { Icon } from "../components/Icon";
import { availableActions, actionLabels } from "../lib/case-actions";
import { money } from "../lib/claims";

type Detail = {
  case: CaseRow;
  signals: Signal[];
  transactions: Txn[];
  notes: Note[];
};

export function CaseWorkspace({ role }: { role: string }) {
  const { id = "" } = useParams();
  const [detail, setDetail] = useState<Detail | null>(null);
  const [evidence, setEvidence] = useState<Evidence[]>([]);
  const [reports, setReports] = useState<Report[]>([]);
  const [graph, setGraph] = useState<Graph | null>(null);
  const [audit, setAudit] = useState<Audit[]>([]);
  const [messages, setMessages] = useState<Message[]>([]);
  const [knockouts, setKnockouts] = useState<KnockoutBoard | null>(null);
  const [dissent, setDissent] = useState<Dissent | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState("");
  const [rationale, setRationale] = useState("");
  const [note, setNote] = useState("");
  const [question, setQuestion] = useState(
    "What is supported by the evidence, and what is missing?",
  );
  const [highlight, setHighlight] = useState("");

  async function load() {
    const [d, ev, rep, g, au, msgs, board] = await Promise.all([
      api<Detail>(`/api/v1/cases/${id}`),
      api<{ data: Evidence[] }>(`/api/v1/cases/${id}/evidence`),
      api<{ data: Report[] }>(`/api/v1/cases/${id}/reports`),
      api<Graph>(`/api/v1/cases/${id}/graph`),
      api<{ data: Audit[] }>(`/api/v1/cases/${id}/audit`),
      api<{ data: Message[] }>(`/api/v1/cases/${id}/assistant`),
      api<KnockoutBoard>(`/api/v1/cases/${id}/knockouts`),
    ]);
    setDetail(d);
    setEvidence(ev.data || []);
    setReports(rep.data || []);
    setGraph(g);
    setAudit(au.data || []);
    setMessages(msgs.data || []);
    setKnockouts(board);
  }

  useEffect(() => {
    setError("");
    setDetail(null);
    setHighlight("");
    setRationale("");
    setNote("");
    setDissent(null);
    setKnockouts(null);
    load().catch((err) => setError(err.message));
  }, [id]);

  async function run(label: string, fn: () => Promise<void>) {
    setBusy(label);
    setError("");
    setNotice("");
    try {
      await fn();
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    } finally {
      setBusy("");
    }
  }

  if (error && !detail)
    return (
      <p className="error" role="alert">
        {error} <Link to="/workspace">Back to queue</Link>
      </p>
    );
  if (!detail) return <p role="status">Loading case…</p>;
  const cse = detail.case;
  const report = reports[0];
  const known = new Set(evidence.map((item) => item.id));
  const roleActions = availableActions(role, cse.status);
  const canWrite =
    role === "investigator" ||
    role === "senior_investigator" ||
    role === "admin";

  return (
    <>
      <Link className="back-link" to="/workspace">
        <Icon name="back" size={16} /> Back to case queue
      </Link>
      <div className="banner">
        Decision support only. Resolving a case records “no adverse action”. It
        does not accuse anyone, freeze an account, or move money.
      </div>
      <div className="row case-heading">
        <h1>{cse.title}</h1>
        <span className={`tag ${cse.status}`}>
          {cse.status.replaceAll("_", " ")}
        </span>
        <span className={`tag ${cse.priority}`}>{cse.priority} triage</span>
      </div>
      <p className="muted mono">
        {cse.id} · account {cse.account_external_id} · version {cse.version}
      </p>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="success" role="status">
          {notice}
        </p>
      )}

      <section className="case-summary" aria-label="Case overview">
        <div>
          <span>Account</span>
          <strong>{cse.account_external_id || "Not recorded"}</strong>
        </div>
        <div>
          <span>Detected signals</span>
          <strong>
            {detail.signals.length}
            <small> to examine</small>
          </strong>
        </div>
        <div>
          <span>Transactions</span>
          <strong>
            {detail.transactions.length}
            <small> linked records</small>
          </strong>
        </div>
        <div>
          <span>Evidence</span>
          <strong>
            {evidence.length}
            <small> source records</small>
          </strong>
        </div>
      </section>
      <nav className="case-subnav" aria-label="Case sections">
        {[
          ["signals", "Signals"],
          ["knockouts", "Knockout desk"],
          ["transactions", "Transactions"],
          ["report", "Draft report"],
          ["relationships", "Observed graph"],
          ["assistant", "Ask"],
          ["decision", "Decision"],
          ["notes", "Notes"],
          ["audit", "Audit"],
        ].map(([section, label]) => (
          <a key={section} href={`#${section}`}>
            {label}
          </a>
        ))}
      </nav>
      <div className="split">
        <div className="grid">
          <section className="card" id="signals">
            <div className="section-title">
              <Icon name="alert" />
              <h2>Detected signals</h2>
              <span className="count">{detail.signals.length}</span>
            </div>
            {detail.signals.length === 0 ? (
              <p>No deterministic signals are attached.</p>
            ) : (
              detail.signals.map((sig) => (
                <article key={sig.id} className="claim">
                  <div className="signal-title">
                    <span className="signal-indicator" />
                    <strong>{sig.rule_code.replaceAll("_", " ")}</strong>
                    <span className="mono">Rule v{sig.rule_version}</span>
                  </div>
                  <p>{sig.explanation}</p>
                  <div className="signal-values">
                    <div>
                      <span>Configured threshold</span>
                      <strong>{sig.threshold}</strong>
                    </div>
                    <div>
                      <span>Observed value</span>
                      <strong>{sig.observed_value}</strong>
                    </div>
                  </div>
                  <EvidenceChips
                    ids={sig.evidence_ids}
                    known={known}
                    onPick={setHighlight}
                  />
                </article>
              ))
            )}
          </section>

          <section className="card" id="knockouts">
            <div className="section-title">
              <Icon name="spark" />
              <h2>Knockout desk</h2>
            </div>
            <p>
              Replay each signal with a hypothetical edit. A cleared replay shows
              what would have to be different. It does not change the ledger,
              and it does not decide that anyone did something wrong.
            </p>
            {!knockouts ? <p>Loading replays…</p> : (
              <>
                <p className="muted">{knockouts.limitation}</p>
                {knockouts.signals.length === 0 ? <p>No signals to replay.</p> : knockouts.signals.map((sig) => (
                  <article key={sig.signal_id} className="claim">
                    <strong>{sig.rule_code.replaceAll("_", " ")}</strong>
                    {!sig.reproduced && <p>{sig.note || "This signal did not reproduce."}</p>}
                    {sig.attempts.map((attempt) => (
                      <div key={attempt.kind + attempt.title} className="knockout-attempt">
                        <span className={attempt.clears ? "tag resolved" : "tag escalated"}>{attempt.clears ? "Clears" : "Stays"}</span>
                        <div>
                          <strong>{attempt.title}</strong>
                          <p>{attempt.detail}</p>
                          <EvidenceChips ids={attempt.evidence_ids || []} known={known} onPick={setHighlight} />
                        </div>
                      </div>
                    ))}
                  </article>
                ))}
                <div className="claim">
                  <h3>Dissent from the replays</h3>
                  <p>{(dissent?.concern.claim) || knockouts.reading.concern}</p>
                  <EvidenceChips ids={dissent?.concern.evidence_ids || knockouts.reading.concern_evidence || []} known={known} onPick={setHighlight} />
                  <p>{(dissent?.benign.claim) || knockouts.reading.benign}</p>
                  <EvidenceChips ids={dissent?.benign.evidence_ids || knockouts.reading.benign_evidence || []} known={known} onPick={setHighlight} />
                  {dissent && (
                    <p className={`tag ${dissent.live ? "high" : ""}`}>
                      {dissent.live ? "Live model" : "Not a live model call"} · {dissent.provider} · {dissent.validation_status}
                    </p>
                  )}
                  {canWrite && (
                    <button type="button" disabled={!!busy} onClick={() => run("dissent", async () => {
                      const drafted = await api<Dissent>(`/api/v1/cases/${id}/dissent`, { method: "POST", body: "{}" });
                      setDissent(drafted);
                      setNotice(drafted.live ? "Dissent drafted by the configured model from the replays." : "Dissent drafted without a live model call. The replays above remain the source.");
                    })}>{busy === "dissent" ? "Drafting dissent…" : "Draft dissent from replays"}</button>
                  )}
                </div>
              </>
            )}
          </section>

          <section className="card" id="transactions">
            <h2>Transactions</h2>
            {detail.transactions.length === 0 ? (
              <p>No transactions are linked to this case.</p>
            ) : (
              <div className="table-scroll">
                <table className="transaction-table">
                  <thead>
                    <tr>
                      <th>Merchant / date</th>
                      <th>Amount</th>
                      <th>Status / context</th>
                    </tr>
                  </thead>
                  <tbody>
                    {detail.transactions.map((tx) => (
                      <tr key={tx.id}>
                        <td>
                          <strong>{tx.merchant_name || "No merchant"}</strong>
                          <small>{stamp(tx.occurred_at)}</small>
                        </td>
                        <td className="amount">
                          {money(tx.amount_minor, tx.currency)}
                        </td>
                        <td>
                          <span className="tag">{tx.status}</span>
                          <small>
                            {tx.device_external_id || "No device"} ·{" "}
                            {tx.location_label || "No location"}
                          </small>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="card" id="report">
            <div className="row">
              <h2>Draft report</h2>
              {canWrite && (
                <button
                  type="button"
                  disabled={!!busy}
                  onClick={() =>
                    run("report", async () => {
                      const result = await api<Report>(
                        `/api/v1/cases/${id}/reports`,
                        {
                          method: "POST",
                        },
                      );
                      if (result.status === "ready") {
                        setNotice(
                          "Report draft updated. Read each claim beside its evidence.",
                        );
                      } else {
                        setError(
                          `Report ${result.status}. Evidence and manual review remain available.`,
                        );
                      }
                    })
                  }
                >
                  {busy === "report" ? "Drafting…" : "Draft from evidence"}
                </button>
              )}
            </div>
            {!report && (
              <p>
                No report yet. The case stays usable if the model is
                unavailable.
              </p>
            )}
            {report && (
              <ReportView report={report} known={known} onPick={setHighlight} />
            )}
          </section>

          <section className="card" id="relationships">
            <h2>Observed graph</h2>
            {graph ? <GraphView graph={graph} /> : <p>Loading graph…</p>}
          </section>

          <section className="card" id="assistant">
            <h2>Ask about this case</h2>
            {canWrite ? (
              <form
                onSubmit={(e: FormEvent) => {
                  e.preventDefault();
                  run("ask", async () => {
                    await api(`/api/v1/cases/${id}/assistant`, {
                      method: "POST",
                      body: JSON.stringify({ question }),
                    });
                  });
                }}
              >
                <label htmlFor="question">Question</label>
                <textarea
                  required
                  id="question"
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                />
                <button type="submit" disabled={!!busy || !question.trim()}>
                  {busy === "ask" ? "Asking…" : "Ask assistant"}
                </button>
              </form>
            ) : (
              <p className="muted">
                This role can read questions. It cannot send new ones.
              </p>
            )}
            {messages.length === 0 ? (
              <p className="muted">No questions yet.</p>
            ) : (
              messages.map((m) => (
                <article key={m.id} className="claim">
                  <strong>
                    {m.role}
                    {m.refused ? " · refused" : ""}
                  </strong>
                  <p>{m.content}</p>
                  <EvidenceChips
                    ids={m.citations || []}
                    known={known}
                    onPick={setHighlight}
                  />
                </article>
              ))
            )}
          </section>

          <section className="card" id="decision">
            <h2>Human decision</h2>
            {roleActions.length === 0 ? (
              <p className="muted">
                {canWrite
                  ? "No further decisions are available for this case status."
                  : "This role is read-only. A senior investigator records the decision."}
              </p>
            ) : (
              <>
                <p className="muted">
                  A rationale of at least 20 characters is required to escalate,
                  resolve, or close.
                </p>
                <label htmlFor="rationale">Rationale</label>
                <textarea
                  placeholder="Explain the evidence and reasoning behind your decision…"
                  id="rationale"
                  maxLength={2000}
                  value={rationale}
                  onChange={(e) => setRationale(e.target.value)}
                />
                <div className="actions">
                  {roleActions.map((action) => (
                    <button
                      key={action}
                      type="button"
                      className={action === "resolve" ? "" : "secondary"}
                      disabled={!!busy}
                      onClick={() => {
                        if (
                          (action === "escalate" ||
                            action === "resolve" ||
                            action === "close") &&
                          Array.from(rationale.trim()).length < 20
                        ) {
                          setError(
                            "A rationale of at least 20 characters is required.",
                          );
                          return;
                        }
                        run(action, async () => {
                          await api(`/api/v1/cases/${id}/actions`, {
                            method: "POST",
                            body: JSON.stringify({
                              action,
                              rationale,
                              version: detail.case.version,
                            }),
                          });
                          setRationale("");
                          setNotice(
                            `${actionLabels[action]} completed. The audit trail has been updated.`,
                          );
                        });
                      }}
                    >
                      {busy === action ? "Saving…" : actionLabels[action]}
                    </button>
                  ))}
                </div>
              </>
            )}
            {cse.resolution && <p>Resolution recorded: {cse.resolution}.</p>}
          </section>

          <section className="card" id="notes">
            <h2>Notes</h2>
            {canWrite ? (
              <form
                onSubmit={(e: FormEvent) => {
                  e.preventDefault();
                  run("note", async () => {
                    await api(`/api/v1/cases/${id}/notes`, {
                      method: "POST",
                      body: JSON.stringify({ body: note }),
                    });
                    setNote("");
                  });
                }}
              >
                <label htmlFor="note">Add a note</label>
                <textarea
                  placeholder="Document an observation or next step…"
                  required
                  id="note"
                  maxLength={4000}
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                />
                <button type="submit" disabled={!!busy || !note.trim()}>
                  {busy === "note" ? "Saving…" : "Save note"}
                </button>
              </form>
            ) : (
              <p className="muted">
                This role can read notes. It cannot add them.
              </p>
            )}
            {detail.notes.length === 0 ? (
              <p className="muted">No notes.</p>
            ) : (
              detail.notes.map((n) => <p key={n.id}>{n.body}</p>)
            )}
          </section>

          <section className="card" id="audit">
            <h2>Audit</h2>
            {audit.length === 0 ? (
              <p>No case audit events yet.</p>
            ) : (
              <ol className="audit-list">
                {audit.map((row) => (
                  <li key={row.id}>
                    <span className="mono">{stamp(row.created_at)}</span> ·{" "}
                    {row.actor_label} · {row.action}{" "}
                    {row.rationale ? `· ${row.rationale}` : ""}
                  </li>
                ))}
              </ol>
            )}
          </section>
        </div>

        <aside className="card evidence" aria-label="Evidence" tabIndex={0}>
          <div className="section-title">
            <Icon name="shield" />
            <h2>Evidence library</h2>
            <span className="count">{evidence.length}</span>
          </div>
          <p className="muted">Source records behind this investigation.</p>
          {evidence.length === 0 ? (
            <p>No evidence records.</p>
          ) : (
            evidence.map((item) => (
              <article
                key={item.id}
                id={item.id}
                className={highlight === item.id ? "target" : ""}
              >
                <div className="evidence-kind">
                  <Icon name="folder" size={14} />
                  <span>{item.kind.replaceAll("_", " ")}</span>
                </div>
                <div className="mono">{item.id}</div>
                <p>{item.payload.summary || item.kind}</p>
                <p className="muted">
                  {item.kind} · {item.source} ·{" "}
                  {item.provenance.method || "recorded"}
                </p>
              </article>
            ))
          )}
        </aside>
      </div>
    </>
  );
}

function stamp(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toISOString().slice(0, 16).replace("T", " ") + " UTC";
}

function EvidenceChips({
  ids,
  known,
  onPick,
}: {
  ids: string[];
  known: Set<string>;
  onPick: (id: string) => void;
}) {
  if (!ids?.length)
    return <p className="muted">No evidence ids on this statement.</p>;
  return (
    <div className="chips">
      {ids.map((id) => (
        <button
          key={id}
          type="button"
          className="chip"
          onClick={() => {
            onPick(id);
            document
              .getElementById(id)
              ?.scrollIntoView({ behavior: "smooth", block: "center" });
          }}
        >
          {known.has(id) ? id : `${id} (not in case)`}
        </button>
      ))}
    </div>
  );
}

function ReportView({
  report,
  known,
  onPick,
}: {
  report: Report;
  known: Set<string>;
  onPick: (id: string) => void;
}) {
  const body = report.report || {};
  return (
    <div>
      <p className={`tag ${report.live ? "high" : ""}`}>
        {report.live ? "Live model" : "Not a live model call"} ·{" "}
        {report.provider} · {report.validation_status}
      </p>
      {report.status !== "ready" && (
        <p className="error" role="status">
          Report status: {report.status}. The case is still open for human
          review.
        </p>
      )}
      <p>{body.summary}</p>
      <h3>Observed facts</h3>
      {(body.observed_facts || []).length === 0 ? (
        <p>No grounded observed facts.</p>
      ) : (
        body.observed_facts!.map((fact, i) => (
          <article key={i} className="claim">
            <p>{fact.claim}</p>
            <EvidenceChips
              ids={fact.evidence_ids || []}
              known={known}
              onPick={onPick}
            />
          </article>
        ))
      )}
      <h3>Hypotheses</h3>
      {(body.hypotheses || []).map((item, i) => (
        <p key={i}>
          {item.claim}{" "}
          <span className="muted">Uncertainty {item.uncertainty}.</span>
        </p>
      ))}
      <h3>Benign alternatives</h3>
      {(body.alternative_explanations || []).map((item, i) => (
        <p key={i}>{item.claim}</p>
      ))}
      <h3>Missing evidence</h3>
      <ul>
        {(body.missing_evidence || []).map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
      <h3>Review steps</h3>
      <ul>
        {(body.suggested_review_steps || []).map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
      {(body.unverified_claims || []).length > 0 && (
        <>
          <h3>Unverified claims</h3>
          <ul>
            {body.unverified_claims!.map((item) => (
              <li key={item.claim}>
                {item.claim} — {item.reason}
              </li>
            ))}
          </ul>
        </>
      )}
      <h3>Limitations</h3>
      <ul>
        {(body.limitations || []).map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

function GraphView({ graph }: { graph: Graph }) {
  const columns = ["account", "transaction", "merchant", "device", "location"];
  const grouped = columns.map((type) =>
    graph.nodes.filter((node) => node.type === type),
  );
  const graphHeight = Math.max(
    340,
    ...grouped.map((nodes) => nodes.length * 60 + 70),
  );
  const pos = new Map<string, { x: number; y: number }>();
  grouped.forEach((nodes, col) =>
    nodes.forEach((node, row) =>
      pos.set(node.id, { x: 70 + col * 205, y: 55 + row * 60 }),
    ),
  );
  return (
    <>
      <div
        className="graph"
        tabIndex={0}
        role="img"
        aria-label="Observed relationship graph. Shared devices are not identity conclusions."
      >
        <svg
          viewBox={`0 0 1100 ${graphHeight}`}
          style={{ minWidth: 900, height: graphHeight }}
        >
          <title>Observed case relationships</title>
          {columns.map((column, i) => (
            <text
              key={column}
              x={50 + i * 205}
              y={20}
              fontSize="10"
              fill="#63776d"
            >
              {column.toUpperCase()}
            </text>
          ))}
          {graph.edges.map((edge) => {
            const a = pos.get(edge.from);
            const b = pos.get(edge.to);
            if (!a || !b) return null;
            return (
              <line
                key={edge.id}
                x1={a.x}
                y1={a.y}
                x2={b.x}
                y2={b.y}
                stroke="#1e5c45"
                strokeWidth="1.2"
              />
            );
          })}
          {graph.nodes.map((node) => {
            const p = pos.get(node.id);
            if (!p) return null;
            return (
              <g key={node.id}>
                <circle
                  cx={p.x}
                  cy={p.y}
                  r="6"
                  fill={node.type === "device" ? "#8a4b08" : "#1e5c45"}
                />
                <text x={p.x + 10} y={p.y + 4} fontSize="11" fill="#1c1712">
                  {node.label || node.type}
                </text>
              </g>
            );
          })}
        </svg>
      </div>
      <ul>
        {graph.limitations.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
      <p className="muted">
        Every edge is labeled observed. Provenance is the source field on the
        transaction, not an inferred identity.
      </p>
    </>
  );
}
