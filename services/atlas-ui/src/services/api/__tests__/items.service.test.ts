import { describe, expect, it } from "vitest";
import { buildItemSearchQuery } from "../items.service";

describe("buildItemSearchQuery", () => {
  it("emits page-only query when only pagination is provided", () => {
    expect(buildItemSearchQuery({ pageNumber: 1, pageSize: 50 })).toBe(
      "?page%5Bnumber%5D=1&page%5Bsize%5D=50",
    );
  });

  it("emits ?search= when q is set", () => {
    expect(buildItemSearchQuery({ q: "scroll", pageNumber: 1, pageSize: 50 })).toBe(
      "?search=scroll&page%5Bnumber%5D=1&page%5Bsize%5D=50",
    );
  });

  it("emits filter[compartment]= and filter[subcategory]=", () => {
    expect(
      buildItemSearchQuery({
        compartment: "equipment",
        subcategory: "bow",
        pageNumber: 1,
        pageSize: 50,
      }),
    ).toBe(
      "?filter%5Bcompartment%5D=equipment&filter%5Bsubcategory%5D=bow&page%5Bnumber%5D=1&page%5Bsize%5D=50",
    );
  });

  it("emits filter[class]=any for the All Classes toggle", () => {
    expect(
      buildItemSearchQuery({
        compartment: "equipment",
        classes: ["any"],
        pageNumber: 1,
        pageSize: 50,
      }),
    ).toBe(
      "?filter%5Bcompartment%5D=equipment&filter%5Bclass%5D=any&page%5Bnumber%5D=1&page%5Bsize%5D=50",
    );
  });

  it("alphabetises and comma-joins per-class selection", () => {
    expect(
      buildItemSearchQuery({
        compartment: "equipment",
        classes: ["warrior", "bowman"],
        pageNumber: 1,
        pageSize: 50,
      }),
    ).toBe(
      "?filter%5Bcompartment%5D=equipment&filter%5Bclass%5D=bowman%2Cwarrior&page%5Bnumber%5D=1&page%5Bsize%5D=50",
    );
  });

  it("omits filter[class] when classes is empty", () => {
    expect(
      buildItemSearchQuery({
        compartment: "equipment",
        classes: [],
        pageNumber: 1,
        pageSize: 50,
      }),
    ).toBe("?filter%5Bcompartment%5D=equipment&page%5Bnumber%5D=1&page%5Bsize%5D=50");
  });

  it("combines search + compartment + subcategory + classes", () => {
    expect(
      buildItemSearchQuery({
        q: "fire",
        compartment: "equipment",
        subcategory: "wand",
        classes: ["magician"],
        pageNumber: 1,
        pageSize: 50,
      }),
    ).toBe(
      "?search=fire&filter%5Bcompartment%5D=equipment&filter%5Bsubcategory%5D=wand&filter%5Bclass%5D=magician&page%5Bnumber%5D=1&page%5Bsize%5D=50",
    );
  });

  it("emits the requested page number and size", () => {
    expect(buildItemSearchQuery({ pageNumber: 3, pageSize: 25 })).toBe(
      "?page%5Bnumber%5D=3&page%5Bsize%5D=25",
    );
  });

  it("combines search with non-default pagination", () => {
    expect(buildItemSearchQuery({ q: "bow", pageNumber: 2, pageSize: 50 })).toBe(
      "?search=bow&page%5Bnumber%5D=2&page%5Bsize%5D=50",
    );
  });
});
