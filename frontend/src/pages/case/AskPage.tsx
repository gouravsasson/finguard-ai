import { FormEvent, useState } from "react";
import { useAsk, useCaseAssistant } from "@/lib/api/hooks";
import { EvidenceChips, Feedback, useCaseFrame, useFeedback } from "./shared";

export function AskPage() {
  const { id, known, canWrite } = useCaseFrame();
  const assistantQuery = useCaseAssistant(id);
  const ask = useAsk(id);
  const { error, notice, run } = useFeedback();
  const [question, setQuestion] = useState(
    "What is supported by the evidence, and what is missing?",
  );
  const messages = assistantQuery.data || [];

  return (
    <section className="case-section">
      <h2>Ask about this case</h2>
      <Feedback error={error} notice={notice} />
      {canWrite ? (
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            run(async () => {
              await ask.mutateAsync(question);
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
          <button type="submit" disabled={ask.isPending || !question.trim()}>
            {ask.isPending ? "Asking…" : "Ask assistant"}
          </button>
        </form>
      ) : (
        <p className="muted">This role can read questions. It cannot send new ones.</p>
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
            <EvidenceChips ids={m.citations || []} caseId={id} known={known} />
          </article>
        ))
      )}
    </section>
  );
}
