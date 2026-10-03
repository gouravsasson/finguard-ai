import { describe, expect, it } from "vitest";
import { money, unknownCitations } from "./claims";

describe("citations", () => {
  it("flags ids that are not in the case evidence set", () => {
    const known = new Set(["ev_a"]);
    expect(unknownCitations(["ev_a", "ev_missing", "ev_missing"], known)).toEqual(["ev_missing"]);
  });

  it("formats minor units without calling them a score", () => {
    expect(money(2500)).toBe("USD 25.00");
  });
});
