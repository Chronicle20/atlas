import { describe, expect, it } from "vitest";
import type { Conversation } from "@/types/models/conversation";
import { deleteState, renameState, setTransitionTarget } from "../editorOps";

function conversationWithAskText(): Conversation {
  return {
    id: "c1",
    type: "conversation",
    attributes: {
      npcId: 1,
      startState: "s1",
      states: [
        {
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
          },
        },
        {
          id: "sa",
          type: "dialogue",
          dialogue: { dialogueType: "sendOk", text: "", choices: [] },
        },
        {
          id: "sb",
          type: "dialogue",
          dialogue: { dialogueType: "sendOk", text: "", choices: [] },
        },
        {
          id: "fallback",
          type: "dialogue",
          dialogue: { dialogueType: "sendOk", text: "", choices: [] },
        },
      ],
    },
  };
}

describe("editorOps — askText rename rewire", () => {
  it("rewires matches[0] and matches[2] nextState but leaves matches[1] and nextState unchanged", () => {
    const renamed = renameState(conversationWithAskText(), "sa", "sz");
    const s1 = renamed.attributes.states.find((s) => s.id === "s1")!;
    expect(s1.askText?.matches?.[0]?.nextState).toBe("sz");
    expect(s1.askText?.matches?.[2]?.nextState).toBe("sz");
    expect(s1.askText?.matches?.[1]?.nextState).toBe("sb");
    expect(s1.askText?.nextState).toBe("fallback");
  });
});

describe("editorOps — askText delete rewire", () => {
  it("rewires every matches[].nextState referencing the deleted state, matching askNumber's delete behaviour", () => {
    const deleted = deleteState(conversationWithAskText(), "sa", {
      cascade: false,
    });
    const s1 = deleted.attributes.states.find((s) => s.id === "s1")!;
    // Matches the existing askNumber pattern: required string fields are
    // left with their pre-delete value rather than nulled, since the type
    // does not allow null.
    expect(s1.askText?.matches?.[0]?.nextState).toBe("sa");
    expect(s1.askText?.matches?.[2]?.nextState).toBe("sa");
    expect(s1.askText?.matches?.[1]?.nextState).toBe("sb");
  });
});

describe("editorOps — askText setTransitionTarget addresses a specific match by index", () => {
  it("retargets match index 1 without touching matches 0, 2, or the fallback", () => {
    const conversation = conversationWithAskText();
    const source = conversation.attributes.states[0]!;
    const next = setTransitionTarget(source, "match", 1, "sx");
    expect(next.askText?.matches?.[1]?.nextState).toBe("sx");
    expect(next.askText?.matches?.[0]?.nextState).toBe("sa");
    expect(next.askText?.matches?.[2]?.nextState).toBe("sa");
    expect(next.askText?.nextState).toBe("fallback");
  });

  it("retargets the fallback edge without touching any match", () => {
    const conversation = conversationWithAskText();
    const source = conversation.attributes.states[0]!;
    const next = setTransitionTarget(source, "fallback", 0, "sx");
    expect(next.askText?.nextState).toBe("sx");
    expect(next.askText?.matches?.[0]?.nextState).toBe("sa");
    expect(next.askText?.matches?.[1]?.nextState).toBe("sb");
    expect(next.askText?.matches?.[2]?.nextState).toBe("sa");
  });
});
