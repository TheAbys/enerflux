<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import type { ChartConfiguration } from "chart.js";
import BaseChart from "../components/BaseChart.vue";
import MetricCard from "../components/MetricCard.vue";
import PageHeader from "../components/PageHeader.vue";
import { fetchJSON, jsonRequest } from "../lib/api";
import { chartColor } from "../lib/chartColors";
import { intlLocale } from "../i18n";
import { translateAnalyticsStatus } from "../i18n/analytics";
import { formatDate, formatNumber, formatPricePerTonne, formatSource, formatValue, localDateTimeValue, withAlpha } from "../lib/format";
import type { LocalizedMessage } from "../lib/i18n";
import { historyRangeLabelKeys, historyRangeShortLabelKeys, historyRangeStart, historyRanges, type HistoryRange } from "../lib/timeRanges";
import type { HeatingResponse, MeasurementPayload, ReadingPoint } from "../lib/types";

const emit = defineEmits<{ updated: [value: LocalizedMessage] }>();
const { t } = useI18n({ useScope: "global" });

const rangeOptions = historyRanges;
const selectedDays = ref<HistoryRange>(30);
const data = ref<HeatingResponse | null>(null);
const hasLoadError = ref(false);
const error = computed(() => t("heating.loadFailed"));
const savingStock = ref(false);
const savingPurchase = ref(false);
const stockStatusKey = ref("");
const stockStatus = computed(() => stockStatusKey.value ? t(stockStatusKey.value) : "");
const stockStatusState = ref<"success" | "error">("success");
const purchaseStatusKey = ref("");
const purchaseStatus = computed(() => purchaseStatusKey.value ? t(purchaseStatusKey.value) : "");
const purchaseStatusState = ref<"success" | "error">("success");

const stockTimestamp = ref(localDateTimeValue());
const stockValue = ref("");
const stockNote = ref("");
const purchaseTimestamp = ref(localDateTimeValue());
const purchaseQuantity = ref("");
const purchaseCost = ref("");
const purchaseNote = ref("");
const chartWindow = ref({ from: Date.now() - 30 * 24 * 60 * 60_000, to: Date.now() });

let requestSequence = 0;
let reloadTimer: number | undefined;

const stockReadings = computed(() => data.value?.stockReadings ?? []);
const dailyValues = computed(() => data.value?.daily ?? []);
const stockUnits = computed(() => [...new Set(stockReadings.value.map((point) => point.unit || t("common.unknownSource")))]);
const stockChartUnit = computed(() => stockUnits.value.length === 1 ? stockUnits.value[0] : stockUnits.value.length ? t("heating.mixedUnits") : "—");
const stockEvents = computed(() => [...stockReadings.value]
  .sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime())
  .slice(0, 12));
const visiblePurchases = computed(() => data.value?.purchases.slice(0, 50) ?? []);
const periodCaption = computed(() => t(historyRangeLabelKeys[selectedDays.value]));
const rangeDescription = computed(() => selectedDays.value === "all"
  ? t("period.allTime")
  : t("heating.rangeDescription", { period: t(historyRangeLabelKeys[selectedDays.value]), from: formatDate(chartWindow.value.from), now: t("common.now") }));
const consumedStatus = computed(() => translateAnalyticsStatus(data.value?.consumed.status, (key) => t(key)));
const forecastStatus = computed(() => translateAnalyticsStatus(data.value?.forecastStatus, (key) => t(key)));

const stockChart = computed<ChartConfiguration | undefined>(() => {
  const points = stockReadings.value;
  if (!points.length) return undefined;
  const groups = new Map<string, ReadingPoint[]>();
  for (const point of points) {
    const key = `${formatSource(point.source)} · ${point.unit || t("common.unit")}`;
    groups.set(key, [...(groups.get(key) ?? []), point]);
  }
  const datasets = [...groups.entries()].map(([label, values], index) => {
    const isManual = values[0].source === "manual";
    const color = chartColor("heating", index + (isManual ? 1 : 0));
    return {
      label,
      data: values.map((point) => ({ x: new Date(point.ts).getTime(), y: Number(point.value), note: point.note || "", eventType: point.eventType || "" })),
      borderColor: color,
      backgroundColor: withAlpha(color, 0.12),
      borderWidth: 2,
      pointRadius: values.length > 350 ? 0 : 2,
      pointHoverRadius: 4,
      pointStyle: isManual ? "rectRot" : "circle",
      fill: "start" as const,
      tension: 0.12,
      spanGaps: false,
    };
  });
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
            title: (items) => items.length ? formatDate(items[0].parsed.x, true) : "",
            label: (context) => ` ${context.dataset.label}: ${formatValue(context.parsed.y, String(context.dataset.label).split(" · ").at(-1))}`,
            afterBody: (items) => {
              if (!items.length) return [];
              const raw = items[0].raw as { note?: string; eventType?: string };
              return [raw.note, raw.eventType === "stock_increase" ? t("heating.refillOrAdjustmentBoundary") : ""].filter((value): value is string => Boolean(value));
            },
          },
        },
      },
      scales: {
        x: {
          type: "linear",
          min: chartWindow.value.from,
          max: chartWindow.value.to,
          grid: { display: false },
          ticks: { maxTicksLimit: 8, callback: (value) => new Intl.DateTimeFormat(intlLocale(), { day: "2-digit", month: "short" }).format(new Date(Number(value))) },
        },
        y: { beginAtZero: false, ticks: { maxTicksLimit: 6, callback: (value) => formatNumber(Number(value), 0) } },
      },
    },
  } as ChartConfiguration;
  });

const usageChart = computed<ChartConfiguration | undefined>(() => {
  if (!dailyValues.value.length) return undefined;
  return {
    type: "bar",
    data: {
      labels: dailyValues.value.map((point) => point.date),
      datasets: [{
        label: t("heating.estimatedPelletsUsed"),
        data: dailyValues.value.map((point) => point.value),
        backgroundColor: withAlpha(chartColor("heating"), 0.72),
        hoverBackgroundColor: chartColor("heating"),
        borderRadius: 3,
        maxBarThickness: 22,
      }],
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      plugins: {
        legend: { display: false },
        tooltip: { callbacks: { label: (context) => ` ${t("heating.estimatedUse")}: ${formatValue(Number(context.parsed.y), "kg")}` } },
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
        y: { beginAtZero: true, ticks: { callback: (value) => `${formatNumber(Number(value), 1)} kg` } },
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
    data.value = await fetchJSON<HeatingResponse>(`/api/v1/analytics/heating?${query}`, { headers: { Accept: "application/json" } });
    if (request !== requestSequence) return;
    const time = new Intl.DateTimeFormat(intlLocale(), { hour: "2-digit", minute: "2-digit" }).format(new Date());
    emit("updated", { key: "app.updatedAt", params: { time } });
  } catch {
    if (request !== requestSequence) return;
    hasLoadError.value = true;
    emit("updated", { key: "common.updateFailed" });
  }
}

async function saveMeasurements(payload: MeasurementPayload[]): Promise<void> {
  await fetchJSON<{ status: string }>("/api/v1/measurements", jsonRequest("POST", payload));
}

async function saveStock(): Promise<void> {
  const value = Number(stockValue.value);
  const timestamp = new Date(stockTimestamp.value);
  if (!Number.isFinite(value) || value < 0 || !Number.isFinite(timestamp.getTime())) return;
  savingStock.value = true;
  stockStatusKey.value = "heating.stockSaving";
  stockStatusState.value = "success";
  try {
    await saveMeasurements([{
      ts: timestamp.toISOString(),
      type: "heating.pellet.stock",
      value,
      unit: "kg",
      source: "manual",
      note: stockNote.value.trim(),
    }]);
    stockStatusKey.value = "heating.stockSaved";
    stockValue.value = "";
    stockNote.value = "";
    stockTimestamp.value = localDateTimeValue();
    await refresh();
  } catch {
    stockStatusState.value = "error";
    stockStatusKey.value = "heating.saveStockFailed";
  } finally {
    savingStock.value = false;
  }
}

async function savePurchase(): Promise<void> {
  const quantity = Number(purchaseQuantity.value);
  const totalPaid = Number(purchaseCost.value);
  const timestamp = new Date(purchaseTimestamp.value);
  if (!(quantity > 0) || !(totalPaid > 0) || !Number.isFinite(timestamp.getTime())) return;
  savingPurchase.value = true;
  purchaseStatusKey.value = "heating.purchaseSaving";
  purchaseStatusState.value = "success";
  try {
    const id = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
    const source = `manual-purchase:${id}`;
    const note = purchaseNote.value.trim();
    const ts = timestamp.toISOString();
    await saveMeasurements([
      { ts, type: "heating.pellet.purchase.quantity", value: quantity, unit: "kg", source, note },
      { ts, type: "heating.pellet.purchase.cost", value: totalPaid, unit: "EUR", source, note },
    ]);
    purchaseStatusKey.value = "heating.purchaseSaved";
    purchaseQuantity.value = "";
    purchaseCost.value = "";
    purchaseNote.value = "";
    purchaseTimestamp.value = localDateTimeValue();
    await refresh();
  } catch {
    purchaseStatusState.value = "error";
    purchaseStatusKey.value = "heating.savePurchaseFailed";
  } finally {
    savingPurchase.value = false;
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
  <PageHeader :eyebrow="$t('heating.eyebrow')" :title="$t('heating.title')" :description="$t('heating.description')" accent="heating">
    <template #actions><button class="rounded-lg bg-energy-ink px-4 py-2.5 text-sm font-semibold text-white hover:bg-slate-700 focus:outline-none focus:ring-2 focus:ring-energy-ink/30" type="button" @click="refresh">{{ $t('common.refresh') }}</button></template>
  </PageHeader>

  <section class="flex flex-wrap items-center justify-between gap-4 rounded-xl border border-energy-border bg-white p-4 shadow-sm" :aria-label="$t('heating.rangeControls')">
    <div><p class="text-xs font-bold uppercase tracking-wider text-energy-muted">{{ $t('heating.consumptionPeriod') }}</p><div class="mt-2 flex flex-wrap gap-2" role="group" :aria-label="$t('heating.chooseHistoryRange')">
      <button v-for="days in rangeOptions" :key="days" class="rounded-lg px-3 py-2 text-sm font-semibold" :class="selectedDays === days ? 'bg-energy-heating text-white' : 'bg-slate-100 text-slate-600 hover:bg-slate-200'" :aria-pressed="selectedDays === days" type="button" @click="selectedDays = days; refresh()">{{ $t(historyRangeShortLabelKeys[days]) }}</button>
    </div></div><p class="text-sm text-energy-muted">{{ periodCaption }}</p>
  </section>
  <p v-if="hasLoadError" class="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">{{ error }}</p>

  <section v-if="data" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3" :aria-label="$t('heating.usageSummary')">
    <MetricCard :label="$t('heating.latestStock')" :value="data.currentStock ? formatValue(data.currentStock.value, data.currentStock.unit) : '—'" :detail="data.currentStock ? `${formatSource(data.currentStock.source)} · ${formatDate(data.currentStock.ts, data.currentStock.source !== 'manual')}` : $t('heating.noStockReadingsStored')" accent="heating" />
    <MetricCard :label="$t('heating.estimatedUsed')" :value="formatValue(data.consumed.value, data.consumed.unit)" :detail="data.consumed.value === null ? consumedStatus : $t('heating.decliningIntervals', { count: formatNumber(data.consumed.intervals, 0), hours: formatNumber(data.consumed.coveredHours, 1) })" :hint-label="$t('common.calculationInfo')" :hint="$t('calculations.pelletUse')" accent="heating" />
    <MetricCard :label="$t('heating.totalRecordedConsumed')" :value="formatValue(data.lifetimeConsumedKg, 'kg')" :detail="data.lifetimeConsumedKg === null ? $t('heating.needTwoStockReadings') : $t('heating.lifetimeDecreases')" :hint-label="$t('common.calculationInfo')" :hint="$t('calculations.pelletUse')" />
    <MetricCard :label="$t('heating.averagePerCoveredDay')" :value="formatValue(data.averageDaily30DaysKg, $t('units.kilogramsPerDay'))" :detail="data.consumedLast30DaysKg === null ? $t('heating.needThirtyDayReadings') : $t('heating.intervalsAcrossDays', { value: formatValue(data.consumedLast30DaysKg, 'kg'), intervals: formatNumber(data.last30DaysIntervals, 0), days: formatNumber(data.last30DaysCoveredHours / 24, 1) })" :hint-label="$t('common.calculationInfo')" :hint="$t('calculations.averageUse')" />
    <MetricCard :label="$t('heating.nextHourEstimate')" :value="formatValue(data.forecastNextHourKg, 'kg')" :detail="forecastStatus" :hint-label="$t('common.calculationInfo')" :hint="$t('calculations.forecast')" />
    <MetricCard :label="$t('heating.estimatedFuelCost')" :value="formatValue(data.consumed.costEur, 'EUR')" :detail="data.averagePurchasePriceEurPerKg === null ? $t('heating.recordPurchaseForCost') : `${$t('heating.weightedPurchaseAverage', { value: formatPricePerTonne(data.averagePurchasePriceEurPerKg) })} · ${$t(data.consumed.costEur === null ? 'common.noUsageEstimate' : 'heating.appliedToSelectedUse')}`" :hint-label="$t('common.calculationInfo')" :hint="$t('calculations.fuelCost')" accent="electricity" />
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="stock-chart-title">
    <div class="mb-4 flex flex-wrap items-start justify-between gap-3"><div><div class="flex items-center gap-3"><h2 id="stock-chart-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t('heating.stockHistory') }}</h2><span class="rounded-full bg-slate-100 px-2.5 py-1 text-xs font-semibold text-slate-600">{{ stockChartUnit }}</span></div><p class="mt-1 text-sm text-energy-muted">{{ $t('heating.storedObservations', { count: formatNumber(stockReadings.length, 0) }) }}</p></div></div>
    <div class="relative h-72 w-full sm:h-80"><BaseChart v-if="stockChart" :config="stockChart" :label="$t('heating.pelletStockHistoryChart')" /><div v-else class="flex h-full flex-col items-center justify-center rounded-lg bg-slate-50 text-center"><strong class="text-sm font-semibold text-slate-700">{{ $t('heating.noStockReadingsPeriod') }}</strong><span class="mt-1 text-sm text-energy-muted">{{ $t('common.chooseLongerRange') }}</span></div></div>
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="usage-chart-title">
    <div class="mb-4"><h2 id="usage-chart-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t('heating.usagePerDay') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('heating.decreasesDescription') }}</p></div>
    <div class="relative h-64 w-full sm:h-72"><BaseChart v-if="usageChart" :config="usageChart" :label="$t('heating.estimatedPelletUseChart')" /><div v-else class="flex h-full flex-col items-center justify-center rounded-lg bg-slate-50 text-center"><strong class="text-sm font-semibold text-slate-700">{{ $t('heating.noConsumptionIntervals') }}</strong><span class="mt-1 text-sm text-energy-muted">{{ $t('heating.needMoreThanOneStockReading') }}</span></div></div>
  </section>

  <div class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(20rem,.8fr)]">
    <section class="min-w-0 rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="stock-events-title">
      <div class="mb-4"><h2 id="stock-events-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t('heating.stockReadingsAndNotes') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('heating.refillResetDescription') }}</p></div>
      <div class="overflow-x-auto"><table class="min-w-full divide-y divide-slate-200 text-left text-sm"><thead class="bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500"><tr><th class="px-3 py-3">{{ $t('common.date') }}</th><th class="px-3 py-3">{{ $t('heating.stockKg') }}</th><th class="px-3 py-3">{{ $t('heating.sourceOrNote') }}</th></tr></thead><tbody class="divide-y divide-slate-100">
        <tr v-if="!stockEvents.length"><td colspan="3" class="px-3 py-8 text-center text-energy-muted">{{ $t('heating.noStockReadingsTable') }}</td></tr>
        <tr v-for="(point, index) in stockEvents" :key="`${point.ts}-${point.source}-${index}`"><td class="whitespace-nowrap px-3 py-3 text-slate-600">{{ formatDate(point.ts, point.source !== "manual") }}</td><td class="whitespace-nowrap px-3 py-3 font-semibold">{{ formatValue(point.value, point.unit) }}</td><td class="px-3 py-3 text-slate-600">{{ [formatSource(point.source), point.eventType === "stock_increase" ? $t('heating.refillOrAdjustment') : "", point.note || ""].filter(Boolean).join(" · ") }}</td></tr>
      </tbody></table></div>
    </section>

    <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="purchase-title">
      <h2 id="purchase-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t('heating.recordPurchase') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('heating.backdatePurchases') }}</p>
      <form class="mt-5 space-y-4" @submit.prevent="savePurchase">
        <label class="block text-sm font-medium text-slate-700">{{ $t('heating.purchaseDateTime') }}<input v-model="purchaseTimestamp" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" type="datetime-local" required></label>
        <div class="grid gap-4 sm:grid-cols-2"><label class="block text-sm font-medium text-slate-700">{{ $t('heating.quantityKg') }}<input v-model="purchaseQuantity" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" type="number" min="0.1" step="0.1" :placeholder="$t('heating.exampleQuantity')" required></label><label class="block text-sm font-medium text-slate-700">{{ $t('heating.totalPaid') }}<input v-model="purchaseCost" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" type="number" min="0.01" step="0.01" :placeholder="$t('heating.exampleTotalPaid')" required></label></div>
        <label class="block text-sm font-medium text-slate-700">{{ $t('common.noteOptional') }}<textarea v-model="purchaseNote" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" rows="2" maxlength="500" :placeholder="$t('heating.supplierReference')"></textarea></label>
        <button class="w-full rounded-lg bg-energy-heating px-4 py-2.5 text-sm font-semibold text-white hover:brightness-95 disabled:opacity-60" type="submit" :disabled="savingPurchase">{{ savingPurchase ? $t('common.saving') : $t('heating.savePurchase') }}</button>
        <p v-if="purchaseStatus" class="rounded-lg px-3 py-2 text-sm" :class="purchaseStatusState === 'error' ? 'bg-rose-50 text-rose-800' : 'bg-emerald-50 text-emerald-800'" role="status" aria-live="polite">{{ purchaseStatus }}</p>
      </form>
    </section>
  </div>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="stock-entry-title">
    <div class="mb-4"><h2 id="stock-entry-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t('heating.addStockAdjustment') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('heating.stockInTank') }}</p></div>
    <form class="grid gap-4 md:grid-cols-2 xl:grid-cols-[1fr_1fr_1.5fr_auto] xl:items-end" @submit.prevent="saveStock">
      <label class="block text-sm font-medium text-slate-700">{{ $t('resource.readingTime') }}<input v-model="stockTimestamp" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" type="datetime-local" required :max="localDateTimeValue()"></label>
      <label class="block text-sm font-medium text-slate-700">{{ $t('heating.stockKg') }}<input v-model="stockValue" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" type="number" min="0" step="1" :placeholder="$t('heating.exampleStock')" required></label>
      <label class="block text-sm font-medium text-slate-700">{{ $t('common.noteOptional') }}<input v-model="stockNote" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" type="text" maxlength="500" :placeholder="$t('heating.stockNotePlaceholder')"></label>
      <button class="rounded-lg bg-energy-ink px-4 py-2.5 text-sm font-semibold text-white hover:bg-slate-700 disabled:opacity-60" type="submit" :disabled="savingStock">{{ savingStock ? $t('common.saving') : $t('common.saveReading') }}</button>
    </form>
    <p v-if="stockStatus" class="mt-3 rounded-lg px-3 py-2 text-sm" :class="stockStatusState === 'error' ? 'bg-rose-50 text-rose-800' : 'bg-emerald-50 text-emerald-800'" role="status" aria-live="polite">{{ stockStatus }}</p>
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-5" aria-labelledby="purchases-table-title">
    <div class="mb-4"><h2 id="purchases-table-title" class="font-display text-lg font-semibold text-energy-ink">{{ $t('heating.recordedPurchases') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('heating.pricePerTonneDescription') }}</p></div>
    <div class="overflow-x-auto"><table class="min-w-full divide-y divide-slate-200 text-left text-sm"><thead class="bg-slate-50 text-[11px] font-bold uppercase tracking-wider text-slate-500"><tr><th class="px-3 py-3">{{ $t('common.date') }}</th><th class="px-3 py-3">{{ $t('common.quantity') }}</th><th class="px-3 py-3">{{ $t('heating.totalPaid') }}</th><th class="px-3 py-3">{{ $t('heating.pricePerTonne') }}</th><th class="px-3 py-3">{{ $t('common.note') }}</th></tr></thead><tbody class="divide-y divide-slate-100">
      <tr v-if="!visiblePurchases.length"><td colspan="5" class="px-3 py-8 text-center text-energy-muted">{{ $t('heating.noPurchasesYet') }}</td></tr>
      <tr v-for="(purchase, index) in visiblePurchases" :key="`${purchase.ts}-${index}`"><td class="whitespace-nowrap px-3 py-3">{{ formatDate(purchase.ts, true) }}</td><td class="whitespace-nowrap px-3 py-3">{{ formatValue(purchase.quantityKg, "kg") }}</td><td class="whitespace-nowrap px-3 py-3">{{ formatValue(purchase.totalCostEur, "EUR") }}</td><td class="whitespace-nowrap px-3 py-3 font-semibold">{{ formatPricePerTonne(purchase.costPerKgEur) }}</td><td class="px-3 py-3">{{ purchase.note || "" }}</td></tr>
    </tbody></table></div>
  </section>

  <footer class="flex flex-col justify-between gap-2 border-t border-energy-border pt-4 text-xs text-energy-muted sm:flex-row"><span>{{ $t('heating.usageCalculationFooter') }}</span><span>{{ rangeDescription }}</span></footer>
</template>
