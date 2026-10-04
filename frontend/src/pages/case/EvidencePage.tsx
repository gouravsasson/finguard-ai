import { useEffect } from "react";
import { Icon } from "../../components/Icon";
import { useCaseFrame } from "./shared";

export function EvidencePage() {
  const { evidence } = useCaseFrame();
  const focus = window.location.hash.replace("#", "");

  useEffect(() => {
    if (!focus) return;
    document.getElementById(focus)?.scrollIntoView({ block: "center" });
  }, [focus, evidence]);

  return (
    <section className="case-section">
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
            className={focus === item.id ? "target" : ""}
          >
            <div className="evidence-kind">
              <Icon name="folder" size={14} />
              <span>{item.kind.replaceAll("_", " ")}</span>
            </div>
            <div className="mono">{item.id}</div>
            <p>{item.payload.summary || item.kind}</p>
            <p className="muted">
              {item.kind} · {item.source} · {item.provenance.method || "recorded"}
            </p>
          </article>
        ))
      )}
    </section>
  );
}
