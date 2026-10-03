const permissions: Record<string, string[]> = {
  investigator: ["assign", "start_review"],
  senior_investigator: [
    "assign",
    "start_review",
    "escalate",
    "resolve",
    "close",
  ],
  admin: ["assign", "start_review", "escalate", "resolve", "close"],
};
const transitions: Record<string, string[]> = {
  open: ["assign", "start_review", "escalate"],
  in_review: ["assign", "escalate", "resolve"],
  escalated: ["assign", "resolve"],
  resolved: ["close"],
  closed: [],
};
export function availableActions(role: string, status: string): string[] {
  return (permissions[role] || []).filter((action) =>
    (transitions[status] || []).includes(action),
  );
}
export const actionLabels: Record<string, string> = {
  assign: "Assign to me",
  start_review: "Start review",
  escalate: "Escalate case",
  resolve: "Resolve case",
  close: "Close case",
};
