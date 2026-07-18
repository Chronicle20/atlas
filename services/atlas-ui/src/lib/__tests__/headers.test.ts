import { describe, it, expect } from "vitest";
import { canonicalHeaders, CANONICAL_TENANT_ID } from "@/lib/headers";

describe("canonicalHeaders", () => {
  it("produces nil-UUID tenant, selection-derived version headers, and the operator header", () => {
    const headers = canonicalHeaders({
      region: "GMS",
      majorVersion: 83,
      minorVersion: 1,
    });
    expect(CANONICAL_TENANT_ID).toBe("00000000-0000-0000-0000-000000000000");
    expect(headers.get("TENANT_ID")).toBe(CANONICAL_TENANT_ID);
    expect(headers.get("REGION")).toBe("GMS");
    expect(headers.get("MAJOR_VERSION")).toBe("83");
    expect(headers.get("MINOR_VERSION")).toBe("1");
    expect(headers.get("X-Atlas-Operator")).toBe("1");
  });
});
