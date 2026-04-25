import { describe, expect, it } from "vitest";
import { buildItemSearchQuery } from "../items.service";

describe("buildItemSearchQuery", () => {
  it("emits empty string for an empty filter object", () => {
    expect(buildItemSearchQuery({})).toBe("");
  });

  it("emits ?search= when q is set", () => {
    expect(buildItemSearchQuery({ q: "scroll" })).toBe("?search=scroll");
  });

  it("emits filter[compartment]= and filter[subcategory]=", () => {
    expect(buildItemSearchQuery({ compartment: "equipment", subcategory: "bow" }))
      .toBe("?filter%5Bcompartment%5D=equipment&filter%5Bsubcategory%5D=bow");
  });

  it("emits filter[class]=any for the All Classes toggle", () => {
    expect(buildItemSearchQuery({ compartment: "equipment", classes: ["any"] }))
      .toBe("?filter%5Bcompartment%5D=equipment&filter%5Bclass%5D=any");
  });

  it("alphabetises and comma-joins per-class selection", () => {
    expect(buildItemSearchQuery({ compartment: "equipment", classes: ["warrior", "bowman"] }))
      .toBe("?filter%5Bcompartment%5D=equipment&filter%5Bclass%5D=bowman%2Cwarrior");
  });

  it("omits filter[class] when classes is empty", () => {
    expect(buildItemSearchQuery({ compartment: "equipment", classes: [] }))
      .toBe("?filter%5Bcompartment%5D=equipment");
  });

  it("combines search + compartment + subcategory + classes", () => {
    expect(
      buildItemSearchQuery({
        q: "fire",
        compartment: "equipment",
        subcategory: "wand",
        classes: ["magician"],
      }),
    ).toBe("?search=fire&filter%5Bcompartment%5D=equipment&filter%5Bsubcategory%5D=wand&filter%5Bclass%5D=magician");
  });
});
