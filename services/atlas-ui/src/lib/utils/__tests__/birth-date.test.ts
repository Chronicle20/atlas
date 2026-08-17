import { describe, it, expect } from "vitest";
import {
  birthDateToInput,
  formatBirthDate,
  inputToBirthDate,
} from "../birth-date";

describe("birthDateToInput", () => {
  it("splits a yyyymmdd integer into the input's yyyy-mm-dd", () => {
    expect(birthDateToInput(19900102)).toBe("1990-01-02");
  });

  it("treats 0 and undefined as unset", () => {
    expect(birthDateToInput(0)).toBe("");
    expect(birthDateToInput(undefined)).toBe("");
  });
});

describe("inputToBirthDate", () => {
  it("packs yyyy-mm-dd into a yyyymmdd integer", () => {
    expect(inputToBirthDate("1990-01-02")).toBe(19900102);
    expect(inputToBirthDate("2000-12-31")).toBe(20001231);
  });

  it("round-trips through birthDateToInput", () => {
    expect(inputToBirthDate(birthDateToInput(19871105))).toBe(19871105);
  });

  it("rejects an empty or malformed value", () => {
    expect(inputToBirthDate("")).toBeNull();
    expect(inputToBirthDate("1990-1-2")).toBeNull();
    expect(inputToBirthDate("not a date")).toBeNull();
  });

  it("rejects dates that do not exist on the calendar", () => {
    expect(inputToBirthDate("1990-13-01")).toBeNull();
    expect(inputToBirthDate("1990-00-01")).toBeNull();
    expect(inputToBirthDate("1990-02-30")).toBeNull();
    expect(inputToBirthDate("1990-04-31")).toBeNull();
    expect(inputToBirthDate("1900-02-29")).toBeNull(); // 1900 is not a leap year
  });

  it("accepts a real leap day", () => {
    expect(inputToBirthDate("2000-02-29")).toBe(20000229);
    expect(inputToBirthDate("1996-02-29")).toBe(19960229);
  });

  // 0 would round-trip through atlas-account's PATCH as "no change", so a bad
  // input must be rejected rather than coerced to a falsy number.
  it("never returns 0 for a rejected value", () => {
    for (const bad of ["", "1990-02-30", "abc"]) {
      expect(inputToBirthDate(bad)).not.toBe(0);
    }
  });
});

describe("formatBirthDate", () => {
  it("renders a set date", () => {
    expect(formatBirthDate(19900102)).toBe("1990-01-02");
  });

  it("says so when unset", () => {
    expect(formatBirthDate(0)).toBe("Not set");
    expect(formatBirthDate(undefined)).toBe("Not set");
  });
});
