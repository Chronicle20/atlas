import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { Template } from "@/types/models/template";

// --- mocks ---

const updateMutateMock = vi.fn();

const samplePreset = {
    id: "preset-1",
    attributes: {
        name: "Warrior Start",
        description: "Default warrior preset",
        tags: ["warrior"],
        jobId: 100,
        gender: 0,
        face: 20000,
        hair: 30030,
        hairColor: 0,
        skinColor: 0,
        mapId: 100000000,
        level: 1,
        meso: 500,
        gm: 0,
        stats: { str: 13, dex: 4, int: 4, luk: 4, hp: 50, mp: 5 },
        defaultName: "",
        equipment: [],
        inventory: [],
        skills: [],
    },
};

const sampleTemplate: Template = {
    id: "tpl-1",
    attributes: {
        region: "GMS",
        majorVersion: 83,
        minorVersion: 1,
        usesPin: false,
        characters: {
            templates: [],
            // presets lives here at runtime even though the TS type doesn't declare it
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
        } as any,
        npcs: [],
        socket: { handlers: [], writers: [] },
        worlds: [],
    },
};

// Template with presets populated
const templateWithPresets: Template = {
    ...sampleTemplate,
    attributes: {
        ...sampleTemplate.attributes,
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        characters: { templates: [], presets: [samplePreset] } as any,
    },
};

vi.mock("@/lib/hooks/api/useTemplates", () => ({
    useTemplate: () => ({
        data: templateWithPresets,
        isLoading: false,
        error: null,
    }),
    useUpdateTemplate: () => ({
        mutate: updateMutateMock,
        isPending: false,
    }),
}));

vi.mock("sonner", () => ({
    toast: {
        success: vi.fn(),
        error: vi.fn(),
    },
}));

vi.mock("@/components/common", () => ({
    LoadingSpinner: () => <div>Loading...</div>,
    ErrorDisplay: ({ error }: { error: string }) => <div>Error: {error}</div>,
}));

// --- helpers ---

import { TemplatesPresetsForm } from "@/pages/templates-character-presets-form";

function renderForm() {
    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    return render(
        <QueryClientProvider client={qc}>
            <MemoryRouter initialEntries={["/templates/tpl-1/presets"]}>
                <Routes>
                    <Route path="/templates/:id/presets" element={<TemplatesPresetsForm />} />
                </Routes>
            </MemoryRouter>
        </QueryClientProvider>,
    );
}

// --- tests ---

describe("TemplatesPresetsForm", () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("renders without crashing and shows Add preset button", () => {
        renderForm();
        expect(screen.getByRole("button", { name: /add preset/i })).toBeInTheDocument();
    });

    it("loads existing presets from the template", async () => {
        renderForm();
        // The preset name should appear as a toggle header
        await waitFor(() => {
            expect(screen.getByText("Warrior Start")).toBeInTheDocument();
        });
    });

    it("appends a new preset card when Add preset is clicked", async () => {
        renderForm();
        const user = userEvent.setup();

        const addButton = screen.getByRole("button", { name: /add preset/i });
        await user.click(addButton);

        // After appending, we should see "New preset" appear
        await waitFor(() => {
            expect(screen.getByText("New preset")).toBeInTheDocument();
        });
    });

    it("calls updateTemplate mutation on Save with merged characters document", async () => {
        renderForm();
        const user = userEvent.setup();

        const saveButton = screen.getByRole("button", { name: /save/i });
        await user.click(saveButton);

        await waitFor(() => {
            expect(updateMutateMock).toHaveBeenCalledTimes(1);
        });

        const [mutateArg] = updateMutateMock.mock.calls[0]!;
        expect(mutateArg.id).toBe("tpl-1");
        // Verify the characters object includes the presets key
        expect(mutateArg.updates.characters).toHaveProperty("presets");
        // Verify existing templates is also preserved
        expect(mutateArg.updates.characters).toHaveProperty("templates");
    });
});
