import { fetchAll } from "@/services/api/pagination";

const BASE_PATH = "/api/data/cosmetics";

// JSON:API row shape of /api/data/cosmetics/faces|hairs (live-verified in the
// PRD: 536 faces / 1520 hairs, attributes carry only {cash}).
interface CosmeticData {
  id: string;
  attributes: { cash: boolean };
}

async function getAllIds(kind: "faces" | "hairs"): Promise<number[]> {
  const rows = await fetchAll<CosmeticData>(`${BASE_PATH}/${kind}`);
  return rows
    .map((row) => Number.parseInt(row.id, 10))
    .filter((id) => Number.isFinite(id))
    .sort((a, b) => a - b);
}

export const cosmeticsService = {
  getAllFaceIds: (): Promise<number[]> => getAllIds("faces"),
  getAllHairIds: (): Promise<number[]> => getAllIds("hairs"),
};
