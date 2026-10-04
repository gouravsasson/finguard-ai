import { Link, NavLink, Outlet, useParams } from "react-router-dom";
import { useCaseDetail, useCaseEvidence } from "@/lib/api/hooks";
import { useSessionStore } from "@/store/session";
import { Icon } from "../../components/Icon";
import type { CaseFrame } from "./shared";

const sections = [
  ["", "Signals"],
  ["evidence", "Evidence"],
  ["knockouts", "Knockout desk"],
  ["transactions", "Transactions"],
  ["report", "Draft report"],
  ["graph", "Observed graph"],
  ["ask", "Ask"],
  ["decision", "Decision"],
  ["notes", "Notes"],
  ["audit", "Audit"],
] as const;

export function CaseLayout() {
  const { id = "" } = useParams();
  const role = useSessionStore((s) => s.user?.role || "viewer");
  const detailQuery = useCaseDetail(id);
  const evidenceQuery = useCaseEvidence(id);
  const detail = detailQuery.data ?? null;
  const evidence = evidenceQuery.data || [];

  if (detailQuery.isError && !detail) {
    return (
      <p className="error" role="alert">
        {detailQuery.error?.message || "Unable to load this case."}{" "}
        <Link to="/workspace">Back to queue</Link>
      </p>
    );
  }
  if (!detail) return <p role="status">Loading case…</p>;

  const cse = detail.case;
  const frame: CaseFrame = {
    id,
    detail,
    evidence,
    known: new Set(evidence.map((item) => item.id)),
    canWrite:
      role === "investigator" ||
      role === "senior_investigator" ||
      role === "admin",
  };

  return (
    <div className="case-page">
      <Link className="back-link" to="/workspace">
        <Icon name="back" size={16} /> Back to case queue
      </Link>
      <p className="case-banner">
        Decision support only. Resolving a case records “no adverse action”. It
        does not accuse anyone, freeze an account, or move money.
      </p>
      <div className="case-heading">
        <h1>{cse.title}</h1>
        <span className={`tag ${cse.status}`}>{cse.status.replaceAll("_", " ")}</span>
        <span className={`tag ${cse.priority}`}>{cse.priority} triage</span>
      </div>
      <p className="muted mono">
        {cse.id} · account {cse.account_external_id} · version {cse.version}
      </p>
      <section className="case-stats" aria-label="Case overview">
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
      <nav className="case-nav" aria-label="Case sections">
        {sections.map(([path, label]) => (
          <NavLink key={label} to={path} end={path === ""}>
            {label}
          </NavLink>
        ))}
      </nav>
      <Outlet context={frame} />
    </div>
  );
}
