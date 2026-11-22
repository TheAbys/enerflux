export const historyRanges = [1, 7, 30, 90, 365, "all"] as const;

export type HistoryRange = typeof historyRanges[number];

export const historyRangeLabelKeys: Record<HistoryRange, string> = {
  1: "period.last24Hours",
  7: "period.last7Days",
  30: "period.last30Days",
  90: "period.last90Days",
  365: "period.lastYear",
  all: "period.allTime",
};

export const historyRangeShortLabelKeys: Record<HistoryRange, string> = {
  1: "period.short24Hours",
  7: "period.short7Days",
  30: "period.short30Days",
  90: "period.short90Days",
  365: "period.shortYear",
  all: "period.shortAll",
};

export function historyRangeStart(range: HistoryRange, end: number): number {
  if (range === "all") return Date.parse("2000-01-01T00:00:00Z");
  return end - range * 24 * 60 * 60_000;
}
