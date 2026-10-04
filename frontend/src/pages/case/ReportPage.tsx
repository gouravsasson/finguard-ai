import { useCaseReports, useCreateReport } from "@/lib/api/hooks";
import { Feedback, ReportView, useCaseFrame, useFeedback } from "./shared";

export function ReportPage() {
  const { id, known, canWrite } = useCaseFrame();
  const reportsQuery = useCaseReports(id);
  const createReport = useCreateReport(id);
  const { error, notice, setError, setNotice, run } = useFeedback();
  const report = reportsQuery.data?.[0];

  return (
    <section className="case-section">
      <div className="row">
        <h2>Draft report</h2>
        {canWrite && (
          <button
            type="button"
            disabled={createReport.isPending}
            onClick={() =>
              run(async () => {
                const result = await createReport.mutateAsync();
                if (result.status === "ready") {
                  setNotice("Report draft updated. Read each claim beside its evidence.");
                } else {
                  setError(
                    `Report ${result.status}. Evidence and manual review remain available.`,
                  );
                }
              })
            }
          >
            {createReport.isPending ? "Drafting…" : "Draft from evidence"}
          </button>
        )}
      </div>
      <Feedback error={error} notice={notice} />
      {!report && (
        <p>No report yet. The case stays usable if the model is unavailable.</p>
      )}
      {report && <ReportView report={report} caseId={id} known={known} />}
    </section>
  );
}
