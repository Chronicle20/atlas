import { describe, it, expect } from "vitest";
import { isSealed, isTagged, FLAG_LOCK } from "../asset-flags";
import type { Asset } from "@/services/api/inventory.service";

function asset(over: Partial<Asset["attributes"]>): Asset {
  return { type: "assets", id: "1", attributes: { flag: 0, owner: "", expiration: "", templateId: 1, id: 1, slot: 0, createdAt: "", quantity: 1, ownerId: 0, rechargeable: 0, strength: 0, dexterity: 0, intelligence: 0, luck: 0, hp: 0, mp: 0, weaponAttack: 0, magicAttack: 0, weaponDefense: 0, magicDefense: 0, accuracy: 0, avoidability: 0, hands: 0, speed: 0, jump: 0, slots: 0, levelType: 0, level: 0, experience: 0, hammersApplied: 0, equippedSince: "", cashId: "", commodityId: 0, purchaseBy: 0, petId: 0, ...over } };
}

describe("asset-flags", () => {
  it("FLAG_LOCK is 0x01", () => expect(FLAG_LOCK).toBe(0x01));
  it("isSealed true when lock bit set", () => expect(isSealed(asset({ flag: 0x01 }))).toBe(true));
  it("isSealed false when lock bit clear", () => expect(isSealed(asset({ flag: 0x02 }))).toBe(false));
  it("isTagged true when owner non-empty", () => expect(isTagged(asset({ owner: "Chronicle" }))).toBe(true));
  it("isTagged false when owner empty/whitespace", () => expect(isTagged(asset({ owner: "  " }))).toBe(false));
});
