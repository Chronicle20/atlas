import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Navigate, Route, Routes, useParams } from "react-router-dom";

vi.mock("@/context/tenant-context", () => ({
  useTenant: () => ({ activeTenant: null }),
}));

function GachaponDetailRedirect() {
  const { id } = useParams();
  return <Navigate to={`/reward-pools/${id}`} replace />;
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/reward-pools" element={<div>reward pools list</div>} />
        <Route path="/reward-pools/:id" element={<div>reward pool detail</div>} />
        <Route path="/gachapons" element={<Navigate to="/reward-pools" replace />} />
        <Route path="/gachapons/:id" element={<GachaponDetailRedirect />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("gachapons → reward-pools redirects", () => {
  it("redirects the list route", () => {
    renderAt("/gachapons");
    expect(screen.getByText("reward pools list")).toBeInTheDocument();
  });
  it("redirects a deep link preserving the id", () => {
    renderAt("/gachapons/4170001");
    expect(screen.getByText("reward pool detail")).toBeInTheDocument();
  });
});
