import { describe, expect, it } from "vitest";
import { availableActions } from "./case-actions";
describe("permitted case actions", () => {
  it("keeps viewers read-only for every status", () => {
    for (const status of [
      "open",
      "in_review",
      "escalated",
      "resolved",
      "closed",
    ])
      expect(availableActions("viewer", status)).toEqual([]);
  });
  it("does not offer premature resolution or repeat review", () => {
    expect(availableActions("senior_investigator", "open")).not.toContain(
      "resolve",
    );
    expect(availableActions("investigator", "in_review")).toEqual(["assign"]);
    expect(availableActions("senior_investigator", "resolved")).toEqual([
      "close",
    ]);
    expect(availableActions("admin", "closed")).toEqual([]);
  });
  it("only grants senior roles escalation and resolution", () => {
    expect(availableActions("investigator", "open")).not.toContain("escalate");
    expect(availableActions("senior_investigator", "in_review")).toContain(
      "resolve",
    );
    expect(availableActions("unknown", "open")).toEqual([]);
  });
});
