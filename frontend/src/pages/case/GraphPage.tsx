import { useCaseGraph } from "@/lib/api/hooks";
import { GraphView, useCaseFrame } from "./shared";

export function GraphPage() {
  const { id } = useCaseFrame();
  const graphQuery = useCaseGraph(id);
  return (
    <section className="case-section">
      <h2>Observed graph</h2>
      {graphQuery.data ? <GraphView graph={graphQuery.data} /> : <p>Loading graph…</p>}
    </section>
  );
}
