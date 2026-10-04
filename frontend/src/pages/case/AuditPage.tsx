import { useCaseAudit } from "@/lib/api/hooks";
import { stamp, useCaseFrame } from "./shared";

export function AuditPage() {
  const { id } = useCaseFrame();
  const auditQuery = useCaseAudit(id);
  const audit = auditQuery.data || [];

  return (
    <section className="case-section">
      <h2>Audit</h2>
      {auditQuery.isPending ? (
        <p>Loading audit…</p>
      ) : audit.length === 0 ? (
        <p>No case audit events yet.</p>
      ) : (
        <ol className="audit-list">
          {audit.map((row) => (
            <li key={row.id}>
              <span className="mono">{stamp(row.created_at)}</span> · {row.actor_label} ·{" "}
              {row.action}
              {row.rationale ? ` · ${row.rationale}` : ""}
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
