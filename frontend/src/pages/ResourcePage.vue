<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { intlLocale } from "../i18n";
import { translateAnalyticsStatus } from "../i18n/analytics";
import type { ChartConfiguration } from "chart.js";
import BaseChart from "../components/BaseChart.vue";
import MetricCard from "../components/MetricCard.vue";
import PageHeader from "../components/PageHeader.vue";
import { fetchJSON, jsonRequest } from "../lib/api";
import { chartColor } from "../lib/chartColors";
import { formatDate, formatNumber, formatSource, formatTimestamp, formatValue, localDateTimeValue, withAlpha } from "../lib/format";
import type { LocalizedMessage } from "../lib/i18n";
import { historyRangeLabelKeys, historyRangeShortLabelKeys, historyRangeStart, historyRanges, type HistoryRange } from "../lib/timeRanges";
import type { DomainResponse, MeasurementPayload, ReadingPoint } from "../lib/types";

type Domain = "water" | "electricity";

const props = defineProps<{ domain: Domain }>();
const emit = defineEmits<{ updated: [value: LocalizedMessage] }>();
const { t } = useI18n({ useScope: "global" });

const dayOptions = historyRanges;
const selectedDays = ref<HistoryRange>(props.domain === "water" ? 365 : 7);
const data = ref<DomainResponse | null>(null);
const hasLoadError = ref(false);
const loadError = computed(() => t("resource.loadInsightsFailed", {
  domain: t(isWater.value ? "navigation.water" : "navigation.electricity"),
}));
const saving = ref(false);
const entryStatusKey = ref("");
const entryStatus = computed(() => entryStatusKey.value ? t(entryStatusKey.value) : "");
const entryStatusState = ref<"success" | "error">("success");
const entryTimestamp = ref(localDateTimeValue());
const entryValue = ref("");
const chartWindow = ref({ from: historyRangeStart(selectedDays.value, Date.now()), to: Date.now() });

let requestSequence = 0;
let reloadTimer: number | undefined;

const isWater = computed(() => props.domain === "water");
const title = computed(() => t(isWater.value ? "resource.waterTitle" : "resource.electricityTitle"));
const eyebrow = computed(() => t(isWater.value ? "resource.waterEyebrow" : "resource.electricityEyebrow"));
const description = computed(() => isWater.value
  ? t("resource.waterDescription")
  : t("resource.electricityDescription"));
const readings = computed<ReadingPoint[]>(() => isWater.value ? data.value?.readings ?? [] : data.value?.powerReadings ?? []);
const latestReading = computed(() => isWater.value ? data.value?.currentReading ?? null : data.value?.currentPower ?? null);
const consumedUnit = computed(() => isWater.value ? "m³" : "kWh");
const measureLabel = computed(() => t(isWater.value ? "resource.measureWater" : "resource.measureElectricity"));
const costLabel = computed(() => t(isWater.value ? "resource.costWater" : "resource.costElectricity"));
const latestLabel = computed(() => t(isWater.value ? "resource.latestMeter" : "resource.latestGridPower"));
const readLabel = computed(() => t(isWater.value ? "resource.meterTotal" : "resource.power"));
const periodLabel = computed(() => t(historyRangeLabelKeys[selectedDays.value]));
const consumedStatus = computed(() => translateAnalyticsStatus(data.value?.consumed.status, (key) => t(key)));
const sortedReadings = computed(() => [...readings.value]
  .sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime())
  .slice(0, 250));

const readingChart = computed<ChartConfiguration | undefined>(() => {
  if (!readings.value.length) return undefined;
  const groups = new Map<string, ReadingPoint[]>();
  for (const point of readings.value) {
    const source = formatSource(point.source);
    groups.set(source, [...(groups.get(source) ?? []), point]);
  }
  const datasets = [...groups.entries()].map(([source, series], index) => {
    const color = chartColor(props.domain, index);
    return {
      label: source,
      data: series.map((point) => ({ x: new Date(point.ts).getTime(), y: Number(point.value) })),
      borderColor: color,
      backgroundColor: withAlpha(color, 0.12),
      borderWidth: 2,
      pointRadius: series.length > 300 ? 0 : 2,
      pointHoverRadius: 4,
      fill: isWater.value,
      tension: 0.15,
      spanGaps: false,
    };
  });
  const unit = isWater.value ? "m³" : "W";
  return {
    type: "line",
    data: { datasets },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      parsing: false,
      plugins: {
        decimation: { enabled: true, algorithm: "lttb", samples: 1000, threshold: 1500 },
        legend: { display: groups.size > 1, position: "bottom", align: "start" },
        tooltip: {
          callbacks: {
            title: (items) => items.length ? formatTimestamp(items[0].parsed.x) : "",
            label: (context) => ` ${context.dataset.label}: ${formatValue(context.parsed.y, unit)}`,
          },
        },
      },
      scales: {
        x: {
          type: "linear",
          grid: { display: false },
          ticks: {
            maxTicksLimit: 8,
            callback: (value) => new Intl.DateTimeFormat(intlLocale(), { day: "2-digit", month: "short", hour: "2-digit" }).format(new Date(Number(value))),
          },
        },
        y: {
          beginAtZero: false,
          ticks: { maxTicksLimit: 6, callback: (value) => formatValue(Number(value), unit) },
        },
      },
    },
  } as ChartConfiguration;
  });

const dailyChart = computed<ChartConfiguration | undefined>(() => {
  const daily = data.value?.daily ?? [];
  if (!daily.length) return undefined;
  const unit = consumedUnit.value;
  const color = chartColor(props.domain);
  return {
    type: "bar",
    data: {
      labels: daily.map((point) => point.date),
      datasets: [{
        label: t(isWater.value ? "resource.measureWater" : "resource.measureElectricity"),
        data: daily.map((point) => point.value),
        backgroundColor: withAlpha(color, 0.75),
        hoverBackgroundColor: color,
        borderRadius: 3,
        maxBarThickness: 24,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      plugins: {
        legend: { display: false },
        tooltip: {
          callbacks: {
            title: (items) => items.length ? String(items[0].label) : "",
            label: (context) => ` ${formatNumber(Number(context.parsed.y))} ${unit}`,
          },
        },
      },
      scales: {
        x: {
          grid: { display: false },
          ticks: {
            maxTicksLimit: 10,
            callback: function (value) {
              const date = new Date(`${this.getLabelForValue(Number(value))}T12:00:00`);
              return new Intl.DateTimeFormat(intlLocale(), { day: "2-digit", month: "short" }).format(date);
            },
          },
        },
        y: { beginAtZero: true, ticks: { callback: (value) => `${formatNumber(Number(value))} ${unit}` } },
      },
    },
  } as ChartConfiguration;
  });

async function refresh(): Promise<void> {
  const request = ++requestSequence;
  const to = Date.now();
  const from = historyRangeStart(selectedDays.value, to);
  chartWindow.value = { from, to };
  hasLoadError.value = false;
  emit("updated", { key: "common.updating" });
  const query = new URLSearchParams({ from: new Date(from).toISOString(), to: new Date(to).toISOString() });
  try {
    const result = await fetchJSON<DomainResponse>(`/api/v1/analytics/${props.domain}?${query}`, { headers: { Accept: "application/json" } });
    if (request !== requestSequence) return;
    data.value = result;
    const time = new Intl.DateTimeFormat(intlLocale(), { hour: "2-digit", minute: "2-digit" }).format(new Date());
    emit("updated", { key: "app.updatedAt", params: { time } });
  } catch {
    if (request !== requestSequence) return;
    hasLoadError.value = true;
    emit("updated", { key: "common.updateFailed" });
  }
}

async function submitWaterReading(): Promise<void> {
  const value = Number(entryValue.value);
  const timestamp = new Date(entryTimestamp.value);
  if (!Number.isFinite(value) || value < 0 || !Number.isFinite(timestamp.getTime())) return;

  saving.value = true;
  entryStatusKey.value = "resource.savingWater";
  entryStatusState.value = "success";
  try {
    const payload: MeasurementPayload[] = [{
      ts: timestamp.toISOString(),
      type: "water.meter.reading",
      value,
      unit: "m³",
      source: "manual",
    }];
    await fetchJSON<{ status: string }>("/api/v1/measurements", jsonRequest("POST", payload));
    entryValue.value = "";
    entryTimestamp.value = localDateTimeValue();
    entryStatusKey.value = "resource.waterSaved";
    await refresh();
  } catch {
    entryStatusState.value = "error";
    entryStatusKey.value = "resource.saveWaterFailed";
  } finally {
    saving.value = false;
  }
}

function onVisibilityChange(): void {
  if (document.visibilityState === "visible") void refresh();
}

onMounted(() => {
  void refresh();
  reloadTimer = window.setInterval(() => void refresh(), 60_000);
  document.addEventListener("visibilitychange", onVisibilityChange);
});

onBeforeUnmount(() => {
  requestSequence++;
  if (reloadTimer !== undefined) window.clearInterval(reloadTimer);
  document.removeEventListener("visibilitychange", onVisibilityChange);
});
</script>

<template>
  <PageHeader :eyebrow="eyebrow" :title="title" :description="description" :accent="isWater ? 'water' : 'electricity'">
    <template #actions><button class="rounded-lg bg-energy-ink px-4 py-2.5 text-sm font-semibold text-white hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-energy-ink/30" type="button" @click="refresh">{{ $t('common.refresh') }}</button></template>
  </PageHeader>

  <section class="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-energy-border bg-white p-4 shadow-sm" :aria-label="$t('resource.periodSummary', { domain: title })">
    <div><p class="text-xs font-bold uppercase tracking-wider text-energy-muted">{{ $t('resource.reportingPeriod') }}</p><div class="mt-2 flex flex-wrap gap-2" role="group" :aria-label="$t('resource.choosePeriod', { domain: title.toLowerCase() })">
      <button v-for="days in dayOptions" :key="days" class="rounded-lg px-3 py-2 text-sm font-semibold" :class="selectedDays === days ? 'bg-energy-ink text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'" :aria-pressed="selectedDays === days" type="button" @click="selectedDays = days; refresh()">{{ $t(historyRangeShortLabelKeys[days]) }}</button>
    </div></div><p class="text-sm text-energy-muted">{{ periodLabel }}</p>
  </section>

  <p v-if="hasLoadError" class="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">{{ loadError }}</p>

  <section class="grid gap-4 md:grid-cols-3" :aria-label="$t('resource.periodSummary', { domain: title })">
    <MetricCard :label="measureLabel" :value="formatValue(data?.consumed.value, consumedUnit)" :detail="!data ? $t('common.waitingForReadings') : data.consumed.value === null ? consumedStatus : $t('resource.intervalsCovered', { count: formatNumber(data.consumed.intervals, 0), hours: formatNumber(data.consumed.coveredHours, 1) })" :hint-label="$t('common.calculationInfo')" :hint="$t(isWater ? 'calculations.waterUsage' : 'calculations.electricityUse')" :accent="isWater ? 'water' : 'electricity'" />
    <MetricCard :label="costLabel" :value="formatValue(data?.consumed.costEur, 'EUR')" :detail="!data ? $t('resource.loadingCostEstimate') : data.consumed.pricePerUnit === null ? $t(isWater ? 'resource.configureWaterRates' : 'resource.configureElectricityRate') : `${formatValue(data.consumed.pricePerUnit, data.consumed.priceUnit)} · ${$t(data.consumed.costEur === null ? 'resource.noUsageEstimatePeriod' : 'resource.estimatedSelectedPeriod')}`" :hint-label="$t('common.calculationInfo')" :hint="$t('calculations.cost')" />
    <MetricCard :label="latestLabel" :value="latestReading ? formatValue(latestReading.value, latestReading.unit) : '—'" :detail="latestReading ? `${formatSource(latestReading.source)} · ${formatDate(latestReading.ts, true)}` : $t(isWater ? 'overview.noMeterReadings' : 'resource.noPowerReadings')" />
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="reading-chart-title">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3"><div><div class="flex items-center gap-3"><h2 id="reading-chart-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t(isWater ? 'resource.waterMeterHistory' : 'resource.gridPowerHistory') }}</h2><span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-600">{{ isWater ? "m³ total" : "W" }}</span></div><p class="mt-1 text-sm text-energy-muted">{{ $t(isWater ? 'resource.cumulativeMeterReadings' : 'resource.powerReadingsFromSource') }}</p></div><span class="text-sm text-energy-muted">{{ formatNumber(readings.length, 0) }} {{ readings.length === 1 ? $t('common.reading') : $t('common.readings') }}</span></div>
    <div class="relative h-72 w-full sm:h-96"><BaseChart v-if="readingChart" :config="readingChart" :label="$t('resource.readingsOverTime', { title })"/><div v-else class="flex h-full items-center justify-center rounded-lg bg-slate-50 text-center text-sm text-energy-muted">{{ $t(isWater ? 'resource.noWaterReadingsRange' : 'resource.noGridPowerReadingsRange') }}</div></div>
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5">
    <div class="mb-4"><h2 class="font-display text-lg font-semibold text-energy-ink">{{ $t(isWater ? 'resource.dailyWaterUse' : 'resource.dailyGridEnergy') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t(isWater ? 'resource.waterDifferencesDescription' : 'resource.gridIntegrationDescription') }}</p></div>
    <div class="relative h-64 w-full sm:h-72"><BaseChart v-if="dailyChart" :config="dailyChart" :label="$t('resource.dailyEstimateChart', { title })"/><div v-else class="flex h-full items-center justify-center rounded-lg bg-slate-50 px-4 text-center text-sm text-energy-muted">{{ $t(isWater ? 'resource.noWaterIntervals' : 'resource.noCompletePowerIntervals') }}</div></div>
  </section>

  <section v-if="isWater" class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="water-entry-title">
    <div class="mb-4"><h2 id="water-entry-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t('resource.addWaterReading') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('resource.enterCumulativeValue') }}</p></div>
    <form class="grid gap-4 md:grid-cols-[1fr_1fr_auto] md:items-end" @submit.prevent="submitWaterReading">
      <label class="block text-sm font-medium text-slate-700">{{ $t('resource.readingTime') }}<input v-model="entryTimestamp" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-water focus:ring-2 focus:ring-energy-water/20" type="datetime-local" required :max="localDateTimeValue()"></label>
      <label class="block text-sm font-medium text-slate-700">{{ $t('resource.meterTotal') }}<input v-model="entryValue" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-water focus:ring-2 focus:ring-energy-water/20" type="number" min="0" step="0.001" :placeholder="$t('resource.exampleWaterValue')" required></label>
      <button class="rounded-lg bg-energy-water px-4 py-2.5 text-sm font-semibold text-white hover:brightness-95 disabled:opacity-60" type="submit" :disabled="saving">{{ saving ? $t('common.saving') : $t('resource.saveReading') }}</button>
    </form>
    <p v-if="entryStatus" class="mt-3 rounded-lg px-3 py-2 text-sm" :class="entryStatusState === 'error' ? 'bg-rose-50 text-rose-800' : 'bg-emerald-50 text-emerald-800'" role="status" aria-live="polite">{{ entryStatus }}</p>
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" :aria-labelledby="isWater ? 'water-readings-title' : 'electricity-readings-title'">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3"><div><h2 :id="isWater ? 'water-readings-title' : 'electricity-readings-title'" class="font-display text-lg font-semibold text-energy-ink">{{ $t(isWater ? 'resource.readingsThisPeriod' : 'resource.powerReadingsThisPeriod') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t(isWater ? 'resource.originalWaterObservations' : 'resource.calculatedPowerMeasurements') }}</p></div><span class="rounded-full bg-slate-100 px-2.5 py-1 text-[11px] font-bold tracking-wide text-slate-500">{{ formatNumber(readings.length, 0) }} {{ $t('resource.readingCountUpper') }}</span></div>
    <div class="overflow-x-auto"><table class="min-w-full divide-y divide-slate-200 text-left text-sm"><thead class="bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500"><tr><th class="px-3 py-3">{{ $t('common.time') }}</th><th class="px-3 py-3">{{ readLabel }}</th><th class="px-3 py-3">{{ $t('common.source') }}</th></tr></thead><tbody class="divide-y divide-slate-100">
      <tr v-if="!sortedReadings.length"><td colspan="3" class="px-3 py-8 text-center text-energy-muted">{{ $t('resource.waitingForReadings') }}</td></tr>
      <tr v-for="(point, index) in sortedReadings" :key="`${point.ts}-${point.source}-${index}`"><td class="whitespace-nowrap px-3 py-3 text-slate-600">{{ formatDate(point.ts, true) }}</td><td class="whitespace-nowrap px-3 py-3 font-semibold text-slate-800">{{ formatValue(point.value, point.unit) }}</td><td class="px-3 py-3 text-slate-600">{{ formatSource(point.source) }}</td></tr>
    </tbody></table></div>
  </section>
  <footer class="flex flex-col justify-between gap-2 border-t border-energy-border pt-4 text-xs text-energy-muted sm:flex-row"><span>{{ $t(isWater ? 'resource.waterIndependentCosts' : 'resource.electricityIndependentCosts') }}</span><span>{{ formatDate(chartWindow.from) }} – {{ $t('common.now') }}</span></footer>
</template>
