<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { ChartConfiguration } from "chart.js";
import BaseChart from "../components/BaseChart.vue";
import InfoHint from "../components/InfoHint.vue";
import PageHeader from "../components/PageHeader.vue";
import { Droplets, Flame, Zap } from "@lucide/vue";
import { intlLocale } from "../i18n";
import { translateAnalyticsStatus } from "../i18n/analytics";
import { RouterLink } from "vue-router";
import { fetchJSON } from "../lib/api";
import { formatNumber, formatSource, formatTimestamp, formatValue, withAlpha } from "../lib/format";
import type { LocalizedMessage } from "../lib/i18n";
import { chartColor, electricitySeriesColor, type ChartDomain } from "../lib/chartColors";
import { historyRangeStart } from "../lib/timeRanges";
import type { MetricPoint, OverviewResponse } from "../lib/types";

type RangeKey = "15m" | "1h" | "24h" | "7d" | "30d" | "90d" | "365d" | "all";
interface MetricOption {
  labelKey: string;
  unit: string;
  domainKey: string;
  chartDomain: ChartDomain;
}

interface ChartPoint extends MetricPoint {
  metricKey: string;
}

const emit = defineEmits<{ updated: [value: LocalizedMessage] }>();
const { t } = useI18n({ useScope: "global" });

const metrics: Record<string, MetricOption> = {
  "water.meter.reading": { labelKey: "overview.waterMeterTotal", unit: "m³", domainKey: "navigation.water", chartDomain: "water" },
  "electricity.all": { labelKey: "overview.electricityMeasurements", unit: "W · Wh", domainKey: "navigation.electricity", chartDomain: "electricity" },
  "grid.consumption.power": { labelKey: "overview.gridPower", unit: "W", domainKey: "navigation.electricity", chartDomain: "electricity" },
  "pv.power": { labelKey: "overview.solarProduction", unit: "W", domainKey: "navigation.electricity", chartDomain: "electricity" },
  "pv.dc.power": { labelKey: "overview.solarDCProduction", unit: "W", domainKey: "navigation.electricity", chartDomain: "electricity" },
  "device.power": { labelKey: "overview.devicePower", unit: "W", domainKey: "navigation.electricity", chartDomain: "electricity" },
  "device.energy.total": { labelKey: "overview.deviceCumulativeEnergy", unit: "Wh", domainKey: "navigation.electricity", chartDomain: "electricity" },
  "heating.pellet.stock": { labelKey: "overview.pelletStock", unit: "kg", domainKey: "heating.breadcrumb", chartDomain: "heating" },
  "climate.temperature.outside": { labelKey: "overview.outsideTemperature", unit: "°C", domainKey: "overview.climate", chartDomain: "neutral" },
};
const electricityMetricKeys = [
  "grid.consumption.power",
  "pv.power",
  "pv.dc.power",
  "device.power",
  "device.energy.total",
] as const;

const ranges: Record<Exclude<RangeKey, "all">, number> = {
  "15m": 15 * 60_000,
  "1h": 60 * 60_000,
  "24h": 24 * 60 * 60_000,
  "7d": 7 * 24 * 60 * 60_000,
  "30d": 30 * 24 * 60 * 60_000,
  "90d": 90 * 24 * 60 * 60_000,
  "365d": 365 * 24 * 60 * 60_000,
};
const rangeOptions: RangeKey[] = ["15m", "1h", "24h", "7d", "30d", "90d", "365d", "all"];
const rangeLabelKeys: Record<RangeKey, string> = {
  "15m": "period.short15Minutes",
  "1h": "period.shortHour",
  "24h": "period.short24Hours",
  "7d": "period.short7Days",
  "30d": "period.short30Days",
  "90d": "period.short90Days",
  "365d": "period.shortYear",
  all: "period.shortAll",
};

const overview = ref<OverviewResponse | null>(null);
const points = ref<ChartPoint[]>([]);
const selectedMetricKey = ref("water.meter.reading");
const selectedElectricityKeys = ref<string[]>([...electricityMetricKeys]);
const selectedRange = ref<RangeKey>("15m");
const autoRefresh = ref(true);
const hasOverviewError = ref(false);
const hasChartError = ref(false);
const chartWindow = ref({ from: Date.now() - ranges["15m"], to: Date.now() });

let requestSequence = 0;
let reloadTimer: number | undefined;

const selectedMetric = computed(() => {
  const metric = metrics[selectedMetricKey.value];
  return { ...metric, label: t(metric.labelKey), domain: t(metric.domainKey) };
});
const isElectricityChart = computed(() => selectedMetricKey.value === "electricity.all");
const visibleMetricKeys = computed(() => isElectricityChart.value
  ? electricityMetricKeys.filter((key) => selectedElectricityKeys.value.includes(key))
  : [selectedMetricKey.value]);
const visiblePoints = computed(() => points.value.filter((point) => visibleMetricKeys.value.includes(point.metricKey)));
const heatingForecastStatus = computed(() => translateAnalyticsStatus(overview.value?.heating.forecastStatus, (key) => t(key)));
function translateDomainName(domain: string): string {
  const keys: Record<string, string> = {
    water: "overview.totalWater",
    electricity: "overview.totalElectricity",
    heating: "overview.totalHeating",
  };
  return keys[domain] ? t(keys[domain]) : domain;
}
const rangeDescription = computed(() => t(({
  "15m": "period.last15Minutes",
  "1h": "period.lastHour",
  "24h": "period.last24Hours",
  "7d": "period.last7Days",
  "30d": "period.last30Days",
  "90d": "period.last90Days",
  "365d": "period.lastYear",
  all: "period.allTime",
} as const)[selectedRange.value]));

const latestReadings = computed(() => [...visiblePoints.value]
  .sort((a, b) => new Date(b.TS).getTime() - new Date(a.TS).getTime())
  .slice(0, 10));

const chartConfig = computed<ChartConfiguration | undefined>(() => {
  if (!visiblePoints.value.length) return undefined;
  const groups = new Map<string, ChartPoint[]>();
  for (const point of visiblePoints.value) {
    const source = formatSource(point.Source);
    const groupKey = isElectricityChart.value ? `${point.metricKey}\u0000${source}` : source;
    groups.set(groupKey, [...(groups.get(groupKey) ?? []), point]);
  }

  const datasets = [...groups.values()].map((series, index) => {
    const firstPoint = series[0];
    const source = formatSource(firstPoint.Source);
    const metric = metrics[firstPoint.metricKey];
    const metricIndex = electricityMetricKeys.indexOf(firstPoint.metricKey as typeof electricityMetricKeys[number]);
    const pointRadius = series.length <= 100 ? 3 : series.length <= 1500 ? 1.5 : 0;
    const color = isElectricityChart.value
      ? electricitySeriesColor(Math.max(0, metricIndex))
      : chartColor(selectedMetric.value.chartDomain, index);
    return {
      label: isElectricityChart.value ? `${t(metric.labelKey)} · ${source}` : source,
      data: series.map((point) => ({ x: new Date(point.TS).getTime(), y: Number(point.Value) })),
      borderColor: color,
      backgroundColor: withAlpha(color, 0.12),
      borderWidth: isElectricityChart.value ? 2.5 : 2,
      pointRadius,
      pointHoverRadius: 5,
      pointHitRadius: 10,
      pointBackgroundColor: color,
      pointBorderColor: "#fff",
      pointBorderWidth: pointRadius > 0 ? 1 : 0,
      hoverBorderWidth: 3,
      fill: isElectricityChart.value ? false : "start" as const,
      tension: 0.18,
      spanGaps: false,
      ...(isElectricityChart.value ? { yAxisID: metric.unit === "Wh" ? "energy" : "power" } : {}),
      _unit: metric.unit,
    };
  });

  const duration = chartWindow.value.to - chartWindow.value.from;
  const powerVisible = visibleMetricKeys.value.some((key) => metrics[key].unit === "W");
  const energyVisible = visibleMetricKeys.value.some((key) => metrics[key].unit === "Wh");
  const xScale = {
    type: "linear" as const,
    min: chartWindow.value.from,
    max: chartWindow.value.to,
    grid: { color: "#f2f5f3", drawTicks: false },
    border: { display: false },
    ticks: {
      maxTicksLimit: 8,
      maxRotation: 0,
      color: "#93a09a",
      font: { family: "DM Sans", size: 11 },
      callback: (value: string | number) => new Intl.DateTimeFormat(intlLocale(), duration <= 36 * 60 * 60_000
        ? { hour: "2-digit", minute: "2-digit" }
        : { day: "2-digit", month: "short", hour: "2-digit" }).format(new Date(Number(value))),
    },
  };
  const yScale = {
    beginAtZero: false,
    border: { display: false },
    grid: { color: "#e8eeeb", drawTicks: false },
    ticks: {
      maxTicksLimit: 6,
      padding: 9,
      color: "#93a09a",
      font: { family: "DM Sans", size: 11 },
      callback: (value: string | number) => formatValue(Number(value), selectedMetric.value.unit),
    },
  };
  const scales = isElectricityChart.value
    ? {
        x: xScale,
        ...(powerVisible ? { power: {
          type: "linear" as const,
          position: "left" as const,
          beginAtZero: false,
          title: { display: true, text: `${t("overview.powerAxis")} (W)`, color: "#788681", font: { family: "DM Sans", size: 11 } },
          grid: { color: "#e8eeeb", drawTicks: false },
          ticks: { maxTicksLimit: 6, padding: 9, color: "#93a09a", font: { family: "DM Sans", size: 11 }, callback: (value: string | number) => formatValue(Number(value), "W") },
        } } : {}),
        ...(energyVisible ? { energy: {
          type: "linear" as const,
          position: "right" as const,
          beginAtZero: false,
          title: { display: true, text: `${t("overview.energyAxis")} (Wh)`, color: "#788681", font: { family: "DM Sans", size: 11 } },
          grid: { drawOnChartArea: false },
          ticks: { maxTicksLimit: 6, padding: 9, color: "#93a09a", font: { family: "DM Sans", size: 11 }, callback: (value: string | number) => formatValue(Number(value), "Wh") },
        } } : {}),
      }
    : { x: xScale, y: yScale };
  return {
    type: "line",
    data: { datasets },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      parsing: false,
      normalized: !isElectricityChart.value,
      interaction: { mode: "nearest", intersect: false, axis: "x" },
      plugins: {
        decimation: { enabled: true, algorithm: "lttb", samples: 1400, threshold: 2000 },
        legend: {
          display: isElectricityChart.value || datasets.length > 1,
          position: "bottom",
          align: "start",
          labels: { usePointStyle: true, pointStyle: "circle", padding: 15, color: "#788681", font: { family: "DM Sans", size: 12 } },
        },
        tooltip: {
          backgroundColor: "#173632",
          titleColor: "#d9eee8",
          bodyColor: "#fff",
          borderColor: "rgba(255,255,255,.12)",
          borderWidth: 1,
          padding: 10,
          callbacks: {
            title: (items) => items.length ? formatTimestamp(items[0].parsed.x) : "",
            label: (context) => {
              const dataset = context.dataset as typeof context.dataset & { _unit?: string };
              return ` ${context.dataset.label}: ${formatValue(context.parsed.y, dataset._unit ?? selectedMetric.value.unit)}`;
            },
          },
        },
      },
      scales,
    },
  } as ChartConfiguration;
  });

function getRange(): { from: number; to: number } {
  const to = Date.now();
  const from = selectedRange.value === "all"
    ? historyRangeStart("all", to)
    : to - ranges[selectedRange.value];
  return { from, to };
}

async function loadOverview(): Promise<OverviewResponse> {
  return fetchJSON<OverviewResponse>("/api/v1/analytics/overview", { headers: { Accept: "application/json" } });
}

async function loadMetric(metricKey: string, range: { from: number; to: number }): Promise<ChartPoint[]> {
  const query = new URLSearchParams({
    key: metricKey,
    from: new Date(range.from).toISOString(),
    to: new Date(range.to).toISOString(),
    limit: "200000",
  });
  const data = await fetchJSON<MetricPoint[]>(`/api/v1/metrics/?${query}`, { headers: { Accept: "application/json" } });
  return data
    .map((point) => ({ ...point, metricKey }))
    .sort((a, b) => new Date(a.TS).getTime() - new Date(b.TS).getTime());
}

async function loadChartPoints(range: { from: number; to: number }): Promise<ChartPoint[]> {
  const metricKeys = isElectricityChart.value ? electricityMetricKeys : [selectedMetricKey.value];
  const series = await Promise.all(metricKeys.map((metricKey) => loadMetric(metricKey, range)));
  return series.flat();
}

async function refresh(): Promise<void> {
  const request = ++requestSequence;
  const range = getRange();
  chartWindow.value = range;
  hasOverviewError.value = false;
  hasChartError.value = false;
  emit("updated", { key: "common.updating" });

  const [overviewResult, chartResult] = await Promise.allSettled([
    loadOverview(),
    loadChartPoints(range),
  ]);
  if (request !== requestSequence) return;

  if (overviewResult.status === "fulfilled") overview.value = overviewResult.value;
  else hasOverviewError.value = true;

  if (chartResult.status === "fulfilled") points.value = chartResult.value ?? points.value;
  else hasChartError.value = true;

  const time = new Intl.DateTimeFormat(intlLocale(), { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(new Date());
  emit("updated", { key: "app.updatedAt", params: { time } });
}

function onVisibilityChange(): void {
  if (document.visibilityState === "visible" && autoRefresh.value) void refresh();
}

onMounted(() => {
  void refresh();
  reloadTimer = window.setInterval(() => {
    if (autoRefresh.value) void refresh();
  }, 60_000);
  document.addEventListener("visibilitychange", onVisibilityChange);
});

onBeforeUnmount(() => {
  requestSequence++;
  if (reloadTimer !== undefined) window.clearInterval(reloadTimer);
  document.removeEventListener("visibilitychange", onVisibilityChange);
});
</script>

<template>
  <PageHeader :eyebrow="$t('overview.eyebrow')" :title="$t('overview.title')" :description="$t('overview.description')">
    <template #actions><button class="rounded-lg bg-energy-ink px-4 py-2.5 text-sm font-semibold text-white hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-energy-ink/30" type="button" @click="refresh">{{ $t('common.refresh') }}</button></template>
  </PageHeader>

  <div v-if="hasOverviewError" class="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">{{ $t('overview.loadFailed') }}</div>

  <section v-if="overview" class="grid gap-4 md:grid-cols-2 xl:grid-cols-3" :aria-label="$t('overview.resourceSummaries')">
    <div class="relative"><RouterLink class="group flex h-full flex-col rounded-xl border border-energy-border border-t-4 border-t-energy-water bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md" to="/water">
      <div class="flex items-center justify-between"><span class="text-xs font-bold tracking-[.14em] text-energy-water">{{ $t('navigation.water').toLocaleUpperCase() }}</span><Droplets class="h-6 w-6 text-energy-water" aria-hidden="true" /></div>
      <h2 class="mt-3 font-display text-lg font-semibold text-energy-ink">{{ $t('navigation.water') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('overview.waterUsed30Days') }}</p>
      <p class="mt-5 font-display text-3xl font-semibold text-slate-900">{{ formatValue(overview.water.consumed.value, "m³") }}</p>
      <p class="mt-2 text-sm font-medium text-energy-muted">{{ overview.water.consumed.costEur === null ? $t('common.costNotConfigured') : $t('overview.costThirtyDays', { value: formatValue(overview.water.consumed.costEur, "EUR") }) }}</p>
      <p class="mt-1 text-sm text-slate-500">{{ overview.water.currentReading ? $t('overview.latestMeterTotal', { value: formatValue(overview.water.currentReading.value, overview.water.currentReading.unit), time: formatTimestamp(overview.water.currentReading.ts) }) : $t('overview.noMeterReadings') }}</p>
      <span class="mt-5 inline-flex items-center gap-2 text-sm font-semibold text-energy-water">{{ $t('overview.waterInsights') }} <span aria-hidden="true">→</span></span>
    </RouterLink><InfoHint class="absolute right-5 top-14 z-10" :label="$t('common.calculationInfo')" :text="$t('calculations.waterUsage')" /></div>
    <div class="relative"><RouterLink class="group flex h-full flex-col rounded-xl border border-energy-border border-t-4 border-t-energy-electricity bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md" to="/electricity">
      <div class="flex items-center justify-between"><span class="text-xs font-bold tracking-[.14em] text-energy-electricity">{{ $t('navigation.electricity').toLocaleUpperCase() }}</span><Zap class="h-6 w-6 text-energy-electricity" aria-hidden="true" /></div>
      <h2 class="mt-3 font-display text-lg font-semibold text-energy-ink">{{ $t('navigation.electricity') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('overview.gridEnergyToday') }}</p>
      <p class="mt-5 font-display text-3xl font-semibold text-slate-900">{{ formatValue(overview.electricity.consumed.value, "kWh") }}</p>
      <p class="mt-2 text-sm font-medium text-energy-muted">{{ overview.electricity.consumed.costEur === null ? $t('common.costNotConfigured') : $t('overview.costToday', { value: formatValue(overview.electricity.consumed.costEur, "EUR") }) }}</p>
      <p class="mt-1 text-sm text-slate-500">{{ overview.electricity.currentPower ? $t('overview.currentGridPower', { value: formatValue(overview.electricity.currentPower.value, overview.electricity.currentPower.unit) }) : $t('overview.noGridReadingsToday') }}</p>
      <span class="mt-5 inline-flex items-center gap-2 text-sm font-semibold text-energy-electricity">{{ $t('overview.electricityInsights') }} <span aria-hidden="true">→</span></span>
    </RouterLink><InfoHint class="absolute right-5 top-14 z-10" :label="$t('common.calculationInfo')" :text="$t('calculations.electricityUse')" /></div>
    <div class="relative"><RouterLink class="group flex h-full flex-col rounded-xl border border-energy-border border-t-4 border-t-energy-heating bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:shadow-md" to="/heating">
      <div class="flex items-center justify-between"><span class="text-xs font-bold tracking-[.14em] text-energy-heating">{{ $t('overview.pelletHeating').toLocaleUpperCase() }}</span><Flame class="h-6 w-6 text-energy-heating" aria-hidden="true" /></div>
      <h2 class="mt-3 font-display text-lg font-semibold text-energy-ink">{{ $t('overview.pelletHeating') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('overview.currentStock') }}</p>
      <p class="mt-5 font-display text-3xl font-semibold text-slate-900">{{ overview.heating.currentStock ? formatValue(overview.heating.currentStock.value, overview.heating.currentStock.unit) : "—" }}</p>
      <p class="mt-2 text-sm font-medium text-energy-muted">{{ overview.heating.consumedLast30DaysKg === null ? $t('overview.noThirtyDayEstimate') : $t('overview.thirtyDayUse', { used: formatValue(overview.heating.consumedLast30DaysKg, "kg"), average: formatValue(overview.heating.averageDaily30DaysKg, $t('units.kilogramsPerCoveredDay')) }) }}</p>
      <p class="mt-1 text-sm text-slate-500">{{ overview.heating.forecastNextHourKg === null ? heatingForecastStatus : $t('overview.nextHourEstimate', { value: formatValue(overview.heating.forecastNextHourKg, "kg") }) }}</p>
      <span class="mt-5 inline-flex items-center gap-2 text-sm font-semibold text-energy-heating">{{ $t('overview.heatingInsights') }} <span aria-hidden="true">→</span></span>
    </RouterLink><InfoHint class="absolute right-5 top-14 z-10" :label="$t('common.calculationInfo')" :text="$t('calculations.pelletUse')" /></div>
  </section>

  <section v-if="overview" class="rounded-xl border border-energy-border bg-white p-5 shadow-sm sm:p-6" :aria-label="$t('overview.totalCostAria')">
    <div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-start"><div><span class="text-xs font-bold tracking-[.14em] text-energy-muted">{{ $t('overview.allDomainsLast30Days') }}</span><div class="mt-2 flex items-center gap-2"><h2 class="font-display text-xl font-semibold text-energy-ink">{{ $t('overview.estimatedTotalCost') }}</h2><InfoHint :label="$t('common.calculationInfo')" :text="$t('calculations.totalCost')" /></div><p class="mt-1 max-w-2xl text-sm text-energy-muted">{{ $t('overview.completeCostDescription') }}</p></div><strong class="font-display text-3xl font-semibold text-energy-ink">{{ formatValue(overview.totalCost.totalEur, "EUR") }}</strong></div>
    <p class="mt-4 rounded-lg bg-slate-50 px-3 py-2 text-sm text-slate-600">{{ overview.totalCost.complete ? $t('overview.completeEstimate') : $t('overview.incompleteEstimate', { domains: overview.totalCost.missingDomains.map(translateDomainName).join(", ") || $t('overview.allDomains') }) }}</p>
    <div class="mt-4 grid gap-3 text-sm sm:grid-cols-3"><p class="flex justify-between gap-3 rounded-lg border border-energy-border px-3 py-2">{{ $t('overview.totalWater') }} <strong>{{ formatValue(overview.totalCost.waterEur, "EUR") }}</strong></p><p class="flex justify-between gap-3 rounded-lg border border-energy-border px-3 py-2">{{ $t('overview.totalElectricity') }} <strong>{{ formatValue(overview.totalCost.electricityEur, "EUR") }}</strong></p><p class="flex justify-between gap-3 rounded-lg border border-energy-border px-3 py-2">{{ $t('overview.totalHeating') }} <strong>{{ formatValue(overview.totalCost.heatingEur, "EUR") }}</strong></p></div>
  </section>

  <section class="grid gap-4 rounded-xl border border-energy-border bg-white p-4 shadow-sm lg:grid-cols-[minmax(18rem,1fr)_auto_auto] lg:items-end" :aria-label="$t('overview.measurementControls')">
    <label class="block text-xs font-bold uppercase tracking-wider text-energy-muted" for="metric">{{ $t('overview.measurement') }}
      <select id="metric" v-model="selectedMetricKey" class="mt-2 block w-full rounded-lg border border-slate-300 bg-white px-3 py-2.5 text-sm font-medium normal-case text-slate-800 outline-none focus:border-energy-water focus:ring-2 focus:ring-energy-water/20" @change="refresh">
        <optgroup :label="$t('navigation.water')"><option value="water.meter.reading">{{ $t('overview.waterMeterTotal') }}</option></optgroup>
        <optgroup :label="$t('navigation.electricity')"><option value="electricity.all">{{ $t('overview.electricityMeasurements') }}</option></optgroup>
        <optgroup :label="$t('heating.breadcrumb')"><option value="heating.pellet.stock">{{ $t('overview.pelletStock') }}</option></optgroup><optgroup :label="$t('overview.climate')"><option value="climate.temperature.outside">{{ $t('overview.outsideTemperature') }}</option></optgroup>
      </select>
    </label>
    <div><p class="text-xs font-bold uppercase tracking-wider text-energy-muted">{{ $t('overview.timeRange') }}</p><div class="mt-2 flex flex-wrap gap-1 rounded-lg bg-slate-100 p-1" role="group" :aria-label="$t('overview.chooseChartRange')">
      <button v-for="range in rangeOptions" :key="range" class="rounded-md px-3 py-2 text-sm font-semibold transition" :class="selectedRange === range ? 'bg-white text-energy-ink shadow-sm' : 'text-slate-500 hover:text-slate-900'" :aria-pressed="selectedRange === range" type="button" @click="selectedRange = range; refresh()">{{ $t(rangeLabelKeys[range]) }}</button>
    </div></div>
    <label class="flex items-center gap-3 text-sm font-medium text-slate-600"><input v-model="autoRefresh" class="h-4 w-4 rounded border-slate-300 text-energy-water focus:ring-energy-water" type="checkbox">{{ $t('overview.autoRefresh') }}</label>
    <fieldset v-if="isElectricityChart" class="space-y-2 lg:col-span-3">
      <legend class="text-xs font-bold uppercase tracking-wider text-energy-muted">{{ $t('overview.electricitySeries') }}</legend>
      <div class="flex flex-wrap gap-2">
        <label v-for="(key, index) in electricityMetricKeys" :key="key" class="flex cursor-pointer items-center gap-2 rounded-lg border border-energy-border px-3 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50">
          <input v-model="selectedElectricityKeys" class="h-4 w-4 rounded border-slate-300 text-energy-water focus:ring-energy-water" type="checkbox" :value="key">
          <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ backgroundColor: electricitySeriesColor(index) }" aria-hidden="true"></span>
          <span>{{ $t(metrics[key].labelKey) }}</span>
          <span class="text-xs text-energy-muted">{{ metrics[key].unit }}</span>
        </label>
      </div>
    </fieldset>
  </section>

  <p v-if="hasChartError" class="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">{{ $t('overview.measurementsLoadFailed', { metric: selectedMetric.label.toLowerCase() }) }}</p>
  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="chart-title">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3"><div><div class="flex items-center gap-3"><h2 id="chart-title" class="font-display text-lg font-semibold text-energy-ink">{{ selectedMetric.label }}</h2><span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-600">{{ selectedMetric.unit || "—" }}</span></div><p class="mt-1 text-sm text-energy-muted">{{ $t('overview.domainObservations', { domain: selectedMetric.domain }) }} · {{ formatTimestamp(chartWindow.from) }} – {{ formatTimestamp(chartWindow.to) }}</p></div><span class="text-sm text-energy-muted">{{ formatNumber(visiblePoints.length, 0) }} {{ visiblePoints.length === 1 ? $t('common.reading') : $t('common.readings') }}</span></div>
    <div class="relative h-72 w-full sm:h-96"><BaseChart v-if="chartConfig" :config="chartConfig" :label="$t('overview.chartOverTime', { metric: selectedMetric.label })"/><div v-else class="flex h-full items-center justify-center rounded-lg bg-slate-50 text-sm text-energy-muted">{{ isElectricityChart && selectedElectricityKeys.length === 0 ? $t('overview.selectElectricitySeries') : $t('overview.noMeasurements') }}</div></div>
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="readings-title">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3"><div><h2 id="readings-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t('overview.recentMeasurements') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('overview.latestStoredValues', { domain: selectedMetric.domain, metric: selectedMetric.label }) }}</p></div><span class="rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-bold tracking-wide text-slate-500">{{ $t('overview.latestTen') }}</span></div>
    <div class="overflow-x-auto"><table class="min-w-full divide-y divide-slate-200 text-left text-sm"><thead class="bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500"><tr><th class="px-3 py-3">{{ $t('common.time') }}</th><th v-if="isElectricityChart" class="px-3 py-3">{{ $t('overview.measurement') }}</th><th class="px-3 py-3">{{ $t('common.source') }}</th><th class="px-3 py-3">{{ $t('common.value') }}</th></tr></thead><tbody class="divide-y divide-slate-100">
      <tr v-if="!latestReadings.length"><td :colspan="isElectricityChart ? 4 : 3" class="px-3 py-8 text-center text-energy-muted">{{ $t('overview.noReadingsInRange') }}</td></tr>
      <tr v-for="(point, index) in latestReadings" :key="`${point.metricKey}-${point.TS}-${point.Source}-${index}`"><td class="whitespace-nowrap px-3 py-3 text-slate-600">{{ formatTimestamp(point.TS) }}</td><td v-if="isElectricityChart" class="px-3 py-3 text-slate-600">{{ $t(metrics[point.metricKey].labelKey) }}</td><td class="px-3 py-3 text-slate-600">{{ formatSource(point.Source) }}</td><td class="whitespace-nowrap px-3 py-3 font-semibold text-slate-800">{{ formatValue(point.Value, point.Unit || metrics[point.metricKey].unit) }}</td></tr>
    </tbody></table></div>
  </section>
  <footer class="flex flex-col justify-between gap-2 border-t border-energy-border pt-4 text-xs text-energy-muted sm:flex-row"><span>EnerFlux · {{ $t('overview.calculationsStoredLocally') }}</span><span>{{ rangeDescription }}</span></footer>
</template>
