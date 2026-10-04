import { Icon } from "../../components/Icon";
import { EvidenceChips, useCaseFrame } from "./shared";

export function SignalsPage() {
  const { id, detail, known } = useCaseFrame();
  return (
    <section className="case-section">
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
            <EvidenceChips ids={sig.evidence_ids} caseId={id} known={known} />
          </article>
        ))
      )}
    </section>
  );
}
