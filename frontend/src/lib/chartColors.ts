export type ChartDomain = "water" | "electricity" | "heating" | "neutral";

const palettes: Record<ChartDomain, readonly string[]> = {
  water: ["#287da1", "#216a87", "#4b93b3", "#6baec4", "#155d7b"],
  electricity: ["#d39a35", "#b9811f", "#e0ad53", "#f0c978", "#9a6715"],
  heating: ["#4f886e", "#3b7058", "#6b9f84", "#8bb59f", "#2e5e49"],
  neutral: ["#718096", "#64748b", "#94a3b8", "#475569", "#a8b3c0"],
};

const electricitySeriesPalette = ["#d39a35", "#287da1", "#4f886e", "#8465a7", "#d06c54"] as const;

export function chartColor(domain: ChartDomain, index = 0): string {
  const palette = palettes[domain];
  return palette[index % palette.length];
}

export function electricitySeriesColor(index: number): string {
  return electricitySeriesPalette[index % electricitySeriesPalette.length];
}
