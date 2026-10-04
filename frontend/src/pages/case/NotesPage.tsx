import { FormEvent, useState } from "react";
import { useAddNote } from "@/lib/api/hooks";
import { Feedback, useCaseFrame, useFeedback } from "./shared";

export function NotesPage() {
  const { detail, canWrite } = useCaseFrame();
  const addNote = useAddNote(detail.case.id);
  const { error, notice, run } = useFeedback();
  const [note, setNote] = useState("");

  return (
    <section className="case-section">
      <h2>Notes</h2>
      <Feedback error={error} notice={notice} />
      {canWrite ? (
        <form
          onSubmit={(e: FormEvent) => {
            e.preventDefault();
            run(async () => {
              await addNote.mutateAsync(note);
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
          <button type="submit" disabled={addNote.isPending || !note.trim()}>
            {addNote.isPending ? "Saving…" : "Save note"}
          </button>
        </form>
      ) : (
        <p className="muted">This role can read notes. It cannot add them.</p>
      )}
      {detail.notes.length === 0 ? (
        <p className="muted">No notes.</p>
      ) : (
        detail.notes.map((n) => <p key={n.id}>{n.body}</p>)
      )}
    </section>
  );
}
