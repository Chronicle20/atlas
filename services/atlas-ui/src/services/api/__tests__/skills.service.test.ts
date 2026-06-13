import { describe, it, expect, vi, beforeEach } from "vitest";

const getOneMock = vi.fn();
vi.mock("@/lib/api/client", () => ({
  api: { getOne: (...args: unknown[]) => getOneMock(...args) },
}));

import { skillsService } from "@/services/api/skills.service";

describe("skillsService.getSkillById", () => {
  beforeEach(() => vi.clearAllMocks());

  it("maps maxLevel and effects from the resource", async () => {
    getOneMock.mockResolvedValue({
      id: "1101004",
      type: "skills",
      attributes: {
        name: "Iron Body",
        description: "Hardens the body.",
        action: false,
        element: "",
        animationTime: 0,
        maxLevel: 20,
        effects: [{ weaponDefense: 16, statups: [{ type: "WeaponDefense", amount: 16 }] }],
      },
    });

    const def = await skillsService.getSkillById("1101004");

    expect(def.maxLevel).toBe(20);
    expect(def.effects[0]?.weaponDefense).toBe(16);
    expect(def.effects[0]?.statups?.[0]).toEqual({ type: "WeaponDefense", amount: 16 });
  });

  it("defaults maxLevel to undefined and effects to [] when absent", async () => {
    getOneMock.mockResolvedValue({
      id: "9",
      type: "skills",
      attributes: { name: "X", action: true, element: "", animationTime: 0 },
    });
    const def = await skillsService.getSkillById("9");
    expect(def.maxLevel).toBeUndefined();
    expect(def.effects).toEqual([]);
  });
});
