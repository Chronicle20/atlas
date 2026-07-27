import { render } from "@testing-library/react";
import { SealIcon } from "@/components/seal-icon";
import type { Tenant } from "@/services/api/tenants.service";

const tenant = {
  id: "t1",
  attributes: { region: "GMS", majorVersion: 83, minorVersion: 1 },
} as unknown as Tenant;

describe("SealIcon", () => {
  it("renders the item-protector game asset when a tenant is given", () => {
    const { getByTestId } = render(<SealIcon tenant={tenant} />);
    const el = getByTestId("seal-icon");
    expect(el.tagName).toBe("IMG");
    expect(el.getAttribute("src")).toContain(
      "t1/GMS/83.1/ui/item-protector/icon.png",
    );
  });

  it("falls back to the lucide lock when no tenant is resolvable", () => {
    const { getByTestId } = render(<SealIcon tenant={null} />);
    // lucide renders an <svg>, not the game <img>.
    expect(getByTestId("seal-icon").tagName).not.toBe("IMG");
  });
});
