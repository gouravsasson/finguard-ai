import { useState } from "react";
import type { CaseAction } from "@/lib/api/types";
import { useCaseAction } from "@/lib/api/hooks";
import { actionLabels, availableActions } from "../../lib/case-actions";
import { useSessionStore } from "@/store/session";
import { Feedback, useCaseFrame, useFeedback } from "./shared";

export function DecisionPage() {
  const { detail, canWrite } = useCaseFrame();
  const role = useSessionStore((s) => s.user?.role || "viewer");
  const act = useCaseAction(detail.case.id);
  const { error, notice, setError, setNotice, run } = useFeedback();
  const [rationale, setRationale] = useState("");
  const roleActions = availableActions(role, detail.case.status);

  return (
    <section className="case-section">
      <h2>Human decision</h2>
      <Feedback error={error} notice={notice} />
      {roleActions.length === 0 ? (
        <p className="muted">
          {canWrite
            ? "No further decisions are available for this case status."
            : "This role is read-only. A senior investigator records the decision."}
        </p>
      ) : (
        <>
          <p className="muted">
            A rationale of at least 20 characters is required to escalate, resolve, or
            close.
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
                disabled={act.isPending}
                onClick={() => {
                  if (
                    (action === "escalate" || action === "resolve" || action === "close") &&
                    Array.from(rationale.trim()).length < 20
                  ) {
                    setError("A rationale of at least 20 characters is required.");
                    return;
                  }
                  run(async () => {
                    await act.mutateAsync({
                      action: action as CaseAction["action"],
                      rationale,
                      version: detail.case.version,
                    });
                    setRationale("");
                    setNotice(
                      `${actionLabels[action]} completed. The audit trail has been updated.`,
                    );
                  });
                }}
              >
                {act.isPending && act.variables?.action === action
                  ? "Saving…"
                  : actionLabels[action]}
              </button>
            ))}
          </div>
        </>
      )}
      {detail.case.resolution && <p>Resolution recorded: {detail.case.resolution}.</p>}
    </section>
  );
}
