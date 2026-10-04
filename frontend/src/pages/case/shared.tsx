import { useEffect, useRef, useState } from "react";
import { Link, useOutletContext } from "react-router-dom";
import type { CaseDetail, Evidence, Graph, Report } from "@/lib/api/types";

export type CaseFrame = {
  id: string;
  detail: CaseDetail;
  evidence: Evidence[];
  known: Set<string>;
  canWrite: boolean;
};

export function useCaseFrame() {
  return useOutletContext<CaseFrame>();
}

export function useFeedback() {
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function run(fn: () => Promise<void>) {
    setError("");
    setNotice("");
    try {
      await fn();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    }
  }

  return { error, notice, setError, setNotice, run };
}

export function Feedback({ error, notice }: { error: string; notice: string }) {
  return (
    <>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="success" role="status">
          {notice}
        </p>
      )}
    </>
  );
}

export function stamp(value: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toISOString().slice(0, 16).replace("T", " ") + " UTC";
}

export function EvidenceChips({
  ids,
  caseId,
  known,
}: {
  ids: string[];
  caseId: string;
  known: Set<string>;
}) {
  if (!ids?.length)
    return <p className="muted">No evidence ids on this statement.</p>;
  return (
    <div className="chips">
      {ids.map((item) => (
        <Link key={item} className="chip" to={`/cases/${caseId}/evidence#${item}`}>
          {known.has(item) ? item : `${item} (not in case)`}
        </Link>
      ))}
    </div>
  );
}

export function ReportView({
  report,
  caseId,
  known,
}: {
  report: Report;
  caseId: string;
  known: Set<string>;
}) {
  const body = report.report || {};
  return (
    <div>
      <p className={`tag ${report.live ? "high" : ""}`}>
        {report.live ? "Live model" : "Not a live model call"} · {report.provider} ·{" "}
        {report.validation_status}
      </p>
      {report.status !== "ready" && (
        <p className="error" role="status">
          Report status: {report.status}. The case is still open for human review.
        </p>
      )}
      <p>{body.summary}</p>
      <h3>Observed facts</h3>
      {(body.observed_facts || []).length === 0 ? (
        <p>No grounded observed facts.</p>
      ) : (
        body.observed_facts!.map((fact, i) => (
          <article key={i} className="claim">
            <p>{fact.claim}</p>
            <EvidenceChips
              ids={fact.evidence_ids || []}
              caseId={caseId}
              known={known}
            />
          </article>
        ))
      )}
      <h3>Hypotheses</h3>
      {(body.hypotheses || []).map((item, i) => (
        <p key={i}>
          {item.claim} <span className="muted">Uncertainty {item.uncertainty}.</span>
        </p>
      ))}
      <h3>Benign alternatives</h3>
      {(body.alternative_explanations || []).map((item, i) => (
        <p key={i}>{item.claim}</p>
      ))}
      <h3>Missing evidence</h3>
      <ul>
        {(body.missing_evidence || []).map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
      <h3>Review steps</h3>
      <ul>
        {(body.suggested_review_steps || []).map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
      {(body.unverified_claims || []).length > 0 && (
        <>
          <h3>Unverified claims</h3>
          <ul>
            {body.unverified_claims!.map((item) => (
              <li key={item.claim}>
                {item.claim} — {item.reason}
              </li>
            ))}
          </ul>
        </>
      )}
      <h3>Limitations</h3>
      <ul>
        {(body.limitations || []).map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
    </div>
  );
}

const GRAPH_COLUMNS = ["account", "transaction", "merchant", "device", "location"];
const NODE_H = 40;
const GAP_Y = 14;
const GRAPH_PAD = 20;

type PlacedNode = Graph["nodes"][number] & {
  x: number;
  y: number;
  column: number;
  w: number;
};

function shortLabel(label: string, nodeW: number) {
  const text = label || "";
  const max = Math.max(8, Math.floor((nodeW - 28) / 7.2));
  return text.length > max ? `${text.slice(0, max - 1)}…` : text;
}

function barycenter(id: string, others: Graph["nodes"], edges: Graph["edges"]) {
  const index = new Map(others.map((node, i) => [node.id, i]));
  const hits: number[] = [];
  for (const edge of edges) {
    if (edge.from === id && index.has(edge.to)) hits.push(index.get(edge.to)!);
    if (edge.to === id && index.has(edge.from)) hits.push(index.get(edge.from)!);
  }
  if (!hits.length) return Number.POSITIVE_INFINITY;
  return hits.reduce((sum, value) => sum + value, 0) / hits.length;
}

function placeGraph(graph: Graph, viewport: number) {
  const groups = GRAPH_COLUMNS.map((type) =>
    graph.nodes.filter((node) => node.type === type),
  ).filter((nodes) => nodes.length > 0);
  const known = new Set(GRAPH_COLUMNS);
  const extra = graph.nodes.filter((node) => !known.has(node.type));
  if (extra.length) groups.push(extra);

  for (let sweep = 0; sweep < 4; sweep += 1) {
    for (let col = 1; col < groups.length; col += 1) {
      groups[col] = [...groups[col]].sort(
        (a, b) =>
          barycenter(a.id, groups[col - 1], graph.edges) -
          barycenter(b.id, groups[col - 1], graph.edges),
      );
    }
    for (let col = groups.length - 2; col >= 0; col -= 1) {
      groups[col] = [...groups[col]].sort(
        (a, b) =>
          barycenter(a.id, groups[col + 1], graph.edges) -
          barycenter(b.id, groups[col + 1], graph.edges),
      );
    }
  }

  const rowH = NODE_H + GAP_Y;
  const maxRows = Math.max(1, ...groups.map((nodes) => nodes.length));
  const columns = Math.max(1, groups.length);
  const width = Math.max(320, viewport);
  const gap = columns > 1 ? Math.min(72, Math.max(16, width * 0.025)) : 0;
  const nodeW = (width - GRAPH_PAD * 2 - (columns - 1) * gap) / columns;
  const placed = new Map<string, PlacedNode>();
  groups.forEach((nodes, column) => {
    const offset = ((maxRows - nodes.length) * rowH) / 2;
    nodes.forEach((node, row) => {
      placed.set(node.id, {
        ...node,
        column,
        w: nodeW,
        x: GRAPH_PAD + column * (nodeW + gap),
        y: GRAPH_PAD + 36 + offset + row * rowH,
      });
    });
  });

  const height = GRAPH_PAD + 36 + maxRows * rowH + 8;
  return { groups, placed, width, height, nodeW };
}

function curve(from: PlacedNode, to: PlacedNode) {
  const left = from.column <= to.column ? from : to;
  const right = from.column <= to.column ? to : from;
  const x1 = left.x + left.w;
  const y1 = left.y + NODE_H / 2;
  const x2 = right.x;
  const y2 = right.y + NODE_H / 2;
  const bend = Math.max(28, (x2 - x1) * 0.45);
  return `M ${x1} ${y1} C ${x1 + bend} ${y1}, ${x2 - bend} ${y2}, ${x2} ${y2}`;
}

export function GraphView({ graph }: { graph: Graph }) {
  const frame = useRef<HTMLDivElement>(null);
  const [active, setActive] = useState("");
  const [viewport, setViewport] = useState(0);

  useEffect(() => {
    const el = frame.current;
    if (!el) return;
    const measure = () => setViewport(el.clientWidth);
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  const { groups, placed, width, height, nodeW } = placeGraph(
    graph,
    viewport || 1100,
  );
  const headers = groups.map((nodes) => nodes[0]?.type || "");
  const linked = new Set<string>();
  if (active) {
    linked.add(active);
    for (const edge of graph.edges) {
      if (edge.from === active) linked.add(edge.to);
      if (edge.to === active) linked.add(edge.from);
    }
  }

  return (
    <>
      <div
        className="graph"
        ref={frame}
        tabIndex={0}
        role="img"
        aria-label="Observed relationship graph. Shared devices are not identity conclusions."
      >
        <svg viewBox={`0 0 ${width} ${height}`} preserveAspectRatio="xMinYMin meet">
          <title>Observed case relationships</title>
          {headers.map((column, index) => (
            <text
              key={column || index}
              x={placed.get(groups[index][0].id)?.x ?? GRAPH_PAD}
              y={22}
              fontSize="13"
              fill="#6c6b6b"
            >
              {column}
            </text>
          ))}
          {graph.edges.map((edge) => {
            const from = placed.get(edge.from);
            const to = placed.get(edge.to);
            if (!from || !to) return null;
            const on = !active || edge.from === active || edge.to === active;
            return (
              <path
                key={edge.id}
                d={curve(from, to)}
                fill="none"
                stroke={on && active ? "#b154f9" : "#d4d4d4"}
                strokeWidth={on && active ? 1.8 : 1.2}
                opacity={on ? 1 : 0.18}
              >
                <title>{edge.type || "observed"}</title>
              </path>
            );
          })}
          {[...placed.values()].map((node) => {
            const on = !active || linked.has(node.id);
            return (
              <g
                key={node.id}
                transform={`translate(${node.x} ${node.y})`}
                opacity={on ? 1 : 0.35}
                onMouseEnter={() => setActive(node.id)}
                onMouseLeave={() => setActive("")}
              >
                <title>{node.label}</title>
                <rect
                  width={node.w}
                  height={NODE_H}
                  rx="10"
                  fill="#fff"
                  stroke={active === node.id ? "#b154f9" : "#e6e6e6"}
                />
                {node.type === "device" && (
                  <circle cx="16" cy={NODE_H / 2} r="4" fill="#b154f9" />
                )}
                <text
                  x={node.type === "device" ? 28 : 12}
                  y={NODE_H / 2 + 4}
                  fontSize="13"
                  fill="#000"
                >
                  {shortLabel(node.label || node.type, nodeW)}
                </text>
              </g>
            );
          })}
        </svg>
      </div>
      <ul>
        {graph.limitations.map((item) => (
          <li key={item}>{item}</li>
        ))}
      </ul>
      <p className="muted">
        Every edge is labeled observed. Provenance is the source field on the
        transaction, not an inferred identity.
      </p>
    </>
  );
}
