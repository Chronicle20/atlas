import { describe, expect, it } from "vitest";
import { useState } from "react";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ConversationInspector } from "../ConversationInspector";
import { analyze } from "../graphAnalysis";
import type {
  Conversation,
  ConversationState,
} from "@/types/models/conversation";

// Radix Select relies on DOM APIs jsdom does not implement.
Element.prototype.hasPointerCapture ||= () => false;
Element.prototype.scrollIntoView ||= () => {};

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

function Harness() {
  const [conversation, setConversation] = useState(conversationWithAskText());
  const analysis = analyze(conversation);

  const onUpdateState = (id: string, next: ConversationState) => {
    setConversation((c) => ({
      ...c,
      attributes: {
        ...c.attributes,
        states: c.attributes.states.map((s) => (s.id === id ? next : s)),
      },
    }));
  };

  return (
    <>
      <ConversationInspector
        conversation={conversation}
        selectedStateId="s1"
        onSelect={() => {}}
        analysis={analysis}
        onUpdateState={onUpdateState}
        onRename={() => {}}
        onDelete={() => {}}
        onSwitchType={() => {}}
        onAddChild={() => {}}
        onInsertBetween={() => {}}
        onInsertBefore={() => {}}
        onClearTransition={() => {}}
        readOnly={false}
      />
      <output data-testid="matches">
        {JSON.stringify(conversation.attributes.states[0]?.askText?.matches)}
      </output>
    </>
  );
}

function matches() {
  return JSON.parse(screen.getByTestId("matches").textContent ?? "[]");
}

describe("AskTextForm matches editor", () => {
  it("adds a match with an empty literal value", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    await user.click(screen.getByRole("button", { name: /add/i }));

    expect(matches()).toHaveLength(3);
    expect(matches()[2]).toEqual({ value: "", nextState: "" });
  });

  it("removes a match by index, preserving the order of the rest", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    const removeButtons = screen.getAllByRole("button", {
      name: "Remove match",
    });
    await user.click(removeButtons[0]!);

    const remaining = matches();
    expect(remaining).toHaveLength(1);
    expect(remaining[0]).toEqual({
      valueFromContext: "{context.pw}",
      nextState: "sb",
    });
  });

  it("moves an entry down, preserving the order of the untouched entries", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    const moveDownButtons = screen.getAllByRole("button", {
      name: "Move down",
    });
    await user.click(moveDownButtons[0]!);

    const reordered = matches();
    expect(reordered).toHaveLength(2);
    expect(reordered[0]).toEqual({
      valueFromContext: "{context.pw}",
      nextState: "sb",
    });
    expect(reordered[1]).toEqual({ value: "a", nextState: "sa" });
  });

  it("moves an entry back up, restoring the original order", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    const moveDownButtons = screen.getAllByRole("button", {
      name: "Move down",
    });
    await user.click(moveDownButtons[0]!);
    const moveUpButtons = screen.getAllByRole("button", { name: "Move up" });
    await user.click(moveUpButtons[1]!);

    const reordered = matches();
    expect(reordered[0]).toEqual({ value: "a", nextState: "sa" });
    expect(reordered[1]).toEqual({
      valueFromContext: "{context.pw}",
      nextState: "sb",
    });
  });

  it("clears value when a row switches from literal to context", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    // Combobox order: [0] state-type select, [1] fallback picker,
    // [2] row0 source, [3] row0 nextState, [4] row1 source, [5] row1 nextState.
    const boxes = screen.getAllByRole("combobox");
    await user.click(boxes[2]!);
    await user.click(await screen.findByRole("option", { name: "context" }));

    const first = matches()[0];
    expect(first.value).toBeUndefined();
    expect(first.valueFromContext).toBe("");
    expect(first.nextState).toBe("sa");
  });

  it("clears valueFromContext when a row switches from context to literal", async () => {
    const user = userEvent.setup();
    render(<Harness />);

    // The second row starts as a context row (index 4 — see combobox order above).
    const boxes = screen.getAllByRole("combobox");
    await user.click(boxes[4]!);
    await user.click(await screen.findByRole("option", { name: "value" }));

    const second = matches()[1];
    expect(second.valueFromContext).toBeUndefined();
    expect(second.value).toBe("");
    expect(second.nextState).toBe("sb");
  });
});
