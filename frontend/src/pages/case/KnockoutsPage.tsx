import { useState } from "react";
import type { Dissent } from "@/lib/api/types";
import { useCaseKnockouts, useDissent } from "@/lib/api/hooks";
import { EvidenceChips, Feedback, useCaseFrame, useFeedback } from "./shared";

export function KnockoutsPage() {
  const { id, known, canWrite } = useCaseFrame();
  const knockoutsQuery = useCaseKnockouts(id);
  const dissentMut = useDissent(id);
  const [dissent, setDissent] = useState<Dissent | null>(null);
  const { error, notice, setNotice, run } = useFeedback();
  const knockouts = knockoutsQuery.data ?? null;

  return (
    <section className="case-section">
      <h2>Knockout desk</h2>
      <Feedback error={error} notice={notice} />
      <p>
        Replay each signal with a hypothetical edit. A cleared replay shows what
        would have to be different. It does not change the ledger, and it does
        not decide that anyone did something wrong.
      </p>
      {!knockouts ? (
        <p>Loading replays…</p>
      ) : (
        <>
          <p className="muted">{knockouts.limitation}</p>
          {knockouts.signals.length === 0 ? (
            <p>No signals to replay.</p>
          ) : (
            knockouts.signals.map((sig) => (
              <article key={sig.signal_id} className="claim">
                <strong>{sig.rule_code.replaceAll("_", " ")}</strong>
                {!sig.reproduced && <p>{sig.note || "This signal did not reproduce."}</p>}
                {sig.attempts.map((attempt) => (
                  <div key={attempt.kind + attempt.title} className="knockout-attempt">
                    <span className={attempt.clears ? "tag resolved" : "tag escalated"}>
                      {attempt.clears ? "Clears" : "Stays"}
                    </span>
                    <div>
                      <strong>{attempt.title}</strong>
                      <p>{attempt.detail}</p>
                      <EvidenceChips
                        ids={attempt.evidence_ids || []}
                        caseId={id}
                        known={known}
                      />
                    </div>
                  </div>
                ))}
              </article>
            ))
          )}
          <div className="claim">
            <h3>Dissent from the replays</h3>
            <p>{dissent?.concern.claim || knockouts.reading.concern}</p>
            <EvidenceChips
              ids={dissent?.concern.evidence_ids || knockouts.reading.concern_evidence || []}
              caseId={id}
              known={known}
            />
            <p>{dissent?.benign.claim || knockouts.reading.benign}</p>
            <EvidenceChips
              ids={dissent?.benign.evidence_ids || knockouts.reading.benign_evidence || []}
              caseId={id}
              known={known}
            />
            {dissent && (
              <p className={`tag ${dissent.live ? "high" : ""}`}>
                {dissent.live ? "Live model" : "Not a live model call"} · {dissent.provider} ·{" "}
                {dissent.validation_status}
              </p>
            )}
            {canWrite && (
              <button
                type="button"
                disabled={dissentMut.isPending}
                onClick={() =>
                  run(async () => {
                    const drafted = await dissentMut.mutateAsync();
                    setDissent(drafted);
                    setNotice(
                      drafted.live
                        ? "Dissent drafted by the configured model from the replays."
                        : "Dissent drafted without a live model call. The replays above remain the source.",
                    );
                  })
                }
              >
                {dissentMut.isPending ? "Drafting dissent…" : "Draft dissent from replays"}
              </button>
            )}
          </div>
        </>
      )}
    </section>
  );
}
