const messageKeys: Record<string, string> = {
  "Estimated by spreading meter changes across the time between readings.": "resource.waterStatusEstimated",
  "Need consecutive water-meter readings overlapping this period.": "resource.waterStatusMissing",
  "Estimated from measured grid power; gaps over 15 minutes are omitted.": "resource.electricityStatusEstimated",
  "No complete grid-power intervals in this period.": "resource.electricityStatusMissing",
  "Estimated from stock decreases; refills are treated as boundaries.": "heating.statusEstimated",
  "Need two stock readings spanning this period.": "heating.statusMissingReadings",
  "Forecast unavailable because recent ETA readings could not be loaded.": "heating.forecastUnavailable",
  "Need at least two recent ETA stock readings.": "heating.forecastNeedTwoReadings",
  "Recent ETA stock readings are too far apart.": "heating.forecastReadingsApart",
  "Latest ETA stock reading is older than 20 minutes.": "heating.forecastStale",
  "Need at least two hours of ETA stock history.": "heating.forecastNeedTwoHours",
  "No declining ETA stock intervals in the forecast window.": "heating.forecastNoDecline",
  "Estimated from the recent ETA stock decline.": "heating.forecastEstimated",
};

export function translateAnalyticsStatus(
  message: string | null | undefined,
  translate: (key: string) => string,
): string {
  if (!message) return "";
  const key = messageKeys[message];
  return key ? translate(key) : message;
}
