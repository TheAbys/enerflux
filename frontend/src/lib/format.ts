import { i18n, intlLocale } from "../i18n";

export function formatNumber(value: number | null | undefined, digits = 2): string {
  if (value === null || value === undefined || !Number.isFinite(Number(value))) return "—";
  return new Intl.NumberFormat(intlLocale(), { maximumFractionDigits: digits }).format(Number(value));
}

export function formatValue(value: number | null | undefined, unit = "", digits = 2): string {
  if (value === null || value === undefined) return "—";
  return `${formatNumber(value, digits)}${unit ? ` ${unit}` : ""}`;
}

export function formatSource(source: string | null | undefined): string {
  const value = source?.trim();
  if (!value) return i18n.global.t("common.unknownSource");
  if (value === "manual") return i18n.global.t("common.manual");
  if (value.startsWith("manual-purchase:")) return i18n.global.t("common.manualPurchase");
  return value;
}

/** Converts an internally stored EUR/kg price to EUR/t for display. */
export function formatPricePerTonne(valueEurPerKg: number | null | undefined): string {
  return formatValue(valueEurPerKg === null || valueEurPerKg === undefined ? null : valueEurPerKg * 1000, "EUR/t");
}

export function formatTimestamp(value: string | number | Date | null | undefined, includeDate = true): string {
  if (value === null || value === undefined) return "—";
  const date = new Date(value);
  const options: Intl.DateTimeFormatOptions = includeDate
    ? { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }
    : { hour: "2-digit", minute: "2-digit", second: "2-digit" };
  return new Intl.DateTimeFormat(intlLocale(), options).format(date);
}

export function formatDate(value: string | number | Date | null | undefined, includeTime = false): string {
  if (value === null || value === undefined) return "—";
  const date = new Date(value);
  const options: Intl.DateTimeFormatOptions = includeTime
    ? { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }
    : { day: "2-digit", month: "short", year: "numeric" };
  return new Intl.DateTimeFormat(intlLocale(), options).format(date);
}

export function localDateTimeValue(timestamp = Date.now()): string {
  const date = new Date(timestamp);
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}

export function withAlpha(hex: string, alpha: number): string {
  const value = hex.replace("#", "");
  const red = parseInt(value.slice(0, 2), 16);
  const green = parseInt(value.slice(2, 4), 16);
  const blue = parseInt(value.slice(4, 6), 16);
  return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}
