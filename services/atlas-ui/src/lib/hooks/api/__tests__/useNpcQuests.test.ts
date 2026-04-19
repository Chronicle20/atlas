import { describe, expect, it } from "vitest";
import { deriveNpcQuestRole } from "../useNpcQuests";
import type { QuestDefinition } from "@/types/models/quest";

function buildQuest(overrides: {
  startReqNpc?: number;
  endReqNpc?: number;
  startActionNpc?: number;
  endActionNpc?: number;
}): QuestDefinition {
  return {
    id: "1",
    type: "quests",
    attributes: {
      name: "Test Quest",
      area: 0,
      autoStart: false,
      autoPreComplete: false,
      autoComplete: false,
      startRequirements: overrides.startReqNpc
        ? { npcId: overrides.startReqNpc }
        : {},
      endRequirements: overrides.endReqNpc ? { npcId: overrides.endReqNpc } : {},
      startActions: overrides.startActionNpc
        ? { npcId: overrides.startActionNpc }
        : {},
      endActions: overrides.endActionNpc ? { npcId: overrides.endActionNpc } : {},
    },
  };
}

describe("deriveNpcQuestRole", () => {
  const npcId = 1012100;

  it('returns "initiator" when only startRequirements.npcId matches', () => {
    const q = buildQuest({ startReqNpc: npcId });
    expect(deriveNpcQuestRole(q, npcId)).toBe("initiator");
  });

  it('returns "completer" when only endActions.npcId matches', () => {
    const q = buildQuest({ endActionNpc: npcId });
    expect(deriveNpcQuestRole(q, npcId)).toBe("completer");
  });

  it('returns "both" when both startRequirements and endActions match', () => {
    const q = buildQuest({ startReqNpc: npcId, endActionNpc: npcId });
    expect(deriveNpcQuestRole(q, npcId)).toBe("both");
  });

  it('returns "initiator" for outlier startActions-only match', () => {
    const q = buildQuest({ startActionNpc: npcId });
    expect(deriveNpcQuestRole(q, npcId)).toBe("initiator");
  });

  it('returns "completer" for outlier endRequirements-only match', () => {
    const q = buildQuest({ endReqNpc: npcId });
    expect(deriveNpcQuestRole(q, npcId)).toBe("completer");
  });
});
