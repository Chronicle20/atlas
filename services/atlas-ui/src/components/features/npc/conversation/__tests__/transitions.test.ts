import { describe, expect, it } from "vitest";
import type { ConversationState } from "@/types/models/conversation";
import { getTransitions } from "../transitions";

function askTextState(
  overrides: Partial<NonNullable<ConversationState["askText"]>> = {},
): ConversationState {
  return {
    id: "s1",
    type: "askText",
    askText: {
      text: "What is the password?",
      defaultText: "",
      minLength: 0,
      maxLength: 32,
      contextKey: "answer",
      matches: [
        { value: "a", nextState: "sa" },
        { valueFromContext: "{context.pw}", nextState: "sb" },
        { value: "c", nextState: "sa" },
      ],
      nextState: "fallback",
      ...overrides,
    },
  };
}

describe("getTransitions — askText", () => {
  it("emits one edge per match plus the fallback edge", () => {
    const edges = getTransitions(askTextState());
    expect(edges).toHaveLength(4);
  });

  it("orders edge targets as match order then fallback", () => {
    const edges = getTransitions(askTextState());
    expect(edges.map((e) => e.target)).toEqual(["sa", "sb", "sa", "fallback"]);
  });

  it("labels each match edge by value or context reference, and labels the fallback edge", () => {
    const edges = getTransitions(askTextState());
    expect(edges.map((e) => e.label)).toEqual([
      "a",
      "{context.pw}",
      "c",
      "fallback",
    ]);
  });

  it("does not deduplicate two matches pointing at the same target", () => {
    const edges = getTransitions(askTextState());
    const toSa = edges.filter((e) => e.target === "sa");
    expect(toSa).toHaveLength(2);
  });

  it("with no matches, produces exactly one edge, to the fallback target", () => {
    const edges = getTransitions(askTextState({ matches: [] }));
    expect(edges).toHaveLength(1);
    expect(edges[0]?.target).toBe("fallback");
  });
});
