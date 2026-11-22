<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { fetchJSON, jsonRequest } from "../lib/api";
import type { LocalizedMessage } from "../lib/i18n";
import PageHeader from "../components/PageHeader.vue";
import { RouterLink } from "vue-router";

interface SourceSettings {
  solarLogURL: string;
  solarLogPollInterval: string;
  etaPelletstockURL: string;
  etaPelletstockPollInterval: string;
  etaOutsidetempURL: string;
  etaOutsidetempPollInterval: string;
}

interface DomainSettings {
  timeZone: string;
  electricityPricePerKWh: number | null;
  waterPricePerM3: number | null;
  wastewaterPricePerM3: number | null;
  pelletTankCapacityKg: number | null;
}

const sourceSettings = reactive<SourceSettings>({
  solarLogURL: "",
  solarLogPollInterval: "",
  etaPelletstockURL: "",
  etaPelletstockPollInterval: "",
  etaOutsidetempURL: "",
  etaOutsidetempPollInterval: "",
});

const domainSettings = reactive({
  timeZone: "Europe/Berlin",
  electricityPricePerKWh: "",
  waterPricePerM3: "",
  wastewaterPricePerM3: "",
  pelletTankCapacityKg: "",
});

const { t } = useI18n({ useScope: "global" });
const sourceErrorKey = ref("");
const sourceError = computed(() => sourceErrorKey.value ? t(sourceErrorKey.value) : "");
const sourceStatusKey = ref("settings.loadingSources");
const sourceStatus = computed(() => t(sourceStatusKey.value));
const domainErrorKey = ref("");
const domainError = computed(() => domainErrorKey.value ? t(domainErrorKey.value) : "");
const domainStatusKey = ref("");
const domainStatus = computed(() => domainStatusKey.value ? t(domainStatusKey.value) : "");
const savingSources = ref(false);
const savingDomains = ref(false);
const emit = defineEmits<{ updated: [value: LocalizedMessage] }>();

function optionalNumber(value: string): number | null {
  const trimmed = value.trim();
  return trimmed === "" ? null : Number(trimmed);
}

async function loadSourceSettings(): Promise<void> {
  sourceErrorKey.value = "";
  sourceStatusKey.value = "settings.loadingSources";

  try {
    Object.assign(sourceSettings, await fetchJSON<SourceSettings>("/api/v1/configuration/sources", {
      headers: { Accept: "application/json" },
    }));
    sourceStatusKey.value = "settings.sourcesLoaded";
    emit("updated", { key: "settings.sourcesUpdated" });
  } catch {
    sourceStatusKey.value = "";
    sourceErrorKey.value = "settings.sourceLoadFailed";
  }
}

async function loadDomainSettings(): Promise<void> {
  domainErrorKey.value = "";
  try {
    const settings = await fetchJSON<DomainSettings>("/api/v1/configuration/domains", {
      headers: { Accept: "application/json" },
    });
    domainSettings.timeZone = settings.timeZone || "Europe/Berlin";
    domainSettings.electricityPricePerKWh = settings.electricityPricePerKWh?.toString() ?? "";
    domainSettings.waterPricePerM3 = settings.waterPricePerM3?.toString() ?? "";
    domainSettings.wastewaterPricePerM3 = settings.wastewaterPricePerM3?.toString() ?? "";
    domainSettings.pelletTankCapacityKg = settings.pelletTankCapacityKg?.toString() ?? "";
    emit("updated", { key: "settings.domainsUpdated" });
  } catch {
    domainErrorKey.value = "settings.domainLoadFailed";
  }
}

async function saveSourceSettings(): Promise<void> {
  sourceErrorKey.value = "";
  sourceStatusKey.value = "settings.savingSources";
  savingSources.value = true;

  try {
    await fetchJSON<{ status: string }>("/api/v1/configuration/sources", jsonRequest("PUT", sourceSettings));
    sourceStatusKey.value = "settings.sourceSaveSuccess";
    emit("updated", { key: "settings.sourceSaved" });
  } catch {
    sourceStatusKey.value = "";
    sourceErrorKey.value = "settings.sourceSaveFailed";
  } finally {
    savingSources.value = false;
  }
}

async function saveDomainSettings(): Promise<void> {
  domainErrorKey.value = "";
  domainStatusKey.value = "settings.savingDomains";
  savingDomains.value = true;

  const payload: DomainSettings = {
    timeZone: domainSettings.timeZone.trim(),
    electricityPricePerKWh: optionalNumber(domainSettings.electricityPricePerKWh),
    waterPricePerM3: optionalNumber(domainSettings.waterPricePerM3),
    wastewaterPricePerM3: optionalNumber(domainSettings.wastewaterPricePerM3),
    pelletTankCapacityKg: optionalNumber(domainSettings.pelletTankCapacityKg),
  };

  try {
    await fetchJSON<{ status: string }>("/api/v1/configuration/domains", jsonRequest("PUT", payload));
    domainStatusKey.value = "settings.domainSaveSuccess";
    emit("updated", { key: "settings.domainSaved" });
  } catch {
    domainStatusKey.value = "";
    domainErrorKey.value = "settings.domainSaveFailed";
  } finally {
    savingDomains.value = false;
  }
}

onMounted(() => {
  void loadSourceSettings();
  void loadDomainSettings();
});
</script>

<template>
  <PageHeader :eyebrow="$t('settings.eyebrow')" :title="$t('settings.title')" :description="$t('settings.description')">
    <template #actions><RouterLink class="rounded-lg border border-slate-300 bg-white px-4 py-2.5 text-sm font-semibold text-slate-700 hover:bg-slate-50" to="/">{{ $t('settings.overview') }}</RouterLink></template>
  </PageHeader>

  <div class="space-y-3">
    <p v-if="sourceError" class="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">{{ sourceError }}</p>
    <p v-if="sourceStatus" class="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800" role="status" aria-live="polite">{{ sourceStatus }}</p>
  </div>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-6">
    <div class="mb-5 flex flex-wrap items-start justify-between gap-3"><div><h2 class="font-display text-xl font-semibold text-energy-ink">{{ $t('settings.httpSources') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('settings.intervalHelp') }} <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">30s</code>, <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">5m</code> {{ $t('common.or') }} <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">1h</code>.</p></div><span class="rounded-full bg-energy-water/10 px-3 py-1 text-xs font-semibold text-energy-water">{{ $t('settings.postgresBacked') }}</span></div>
    <form class="space-y-5" @submit.prevent="saveSourceSettings">
      <div class="grid gap-4 lg:grid-cols-3">
        <fieldset class="rounded-xl border border-energy-border p-4"><legend class="px-1 font-display font-semibold text-energy-ink">SolarLog</legend><p class="mb-4 text-sm text-energy-muted">{{ $t('settings.solarGridDescription') }}</p><div class="space-y-4"><label class="block text-sm font-medium text-slate-700">{{ $t('settings.deviceEndpoint') }}<input v-model="sourceSettings.solarLogURL" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-energy-water focus:ring-2 focus:ring-energy-water/20" name="solarLogURL" type="url" autocomplete="url" required placeholder="http://solar-log.local/getjp"></label><label class="block text-sm font-medium text-slate-700">{{ $t('settings.pollInterval') }}<input v-model="sourceSettings.solarLogPollInterval" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-energy-water focus:ring-2 focus:ring-energy-water/20" name="solarLogPollInterval" type="text" required placeholder="30s" pattern="[0-9]+(\.[0-9]+)?(ns|us|µs|μs|ms|s|m|h)([0-9]+(\.[0-9]+)?(ns|us|µs|μs|ms|s|m|h))*"><small class="mt-1 block text-xs text-energy-muted">{{ $t('settings.pollingRange') }}</small></label></div></fieldset>
        <fieldset class="rounded-xl border border-energy-border p-4"><legend class="px-1 font-display font-semibold text-energy-ink">{{ $t('settings.etaPelletStock') }}</legend><p class="mb-4 text-sm text-energy-muted">{{ $t('settings.heatingInventory') }}</p><div class="space-y-4"><label class="block text-sm font-medium text-slate-700">{{ $t('settings.deviceEndpoint') }}<input v-model="sourceSettings.etaPelletstockURL" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" name="etaPelletstockURL" type="url" autocomplete="url" required placeholder="http://eta.local/user/var/..."></label><label class="block text-sm font-medium text-slate-700">{{ $t('settings.pollInterval') }}<input v-model="sourceSettings.etaPelletstockPollInterval" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" name="etaPelletstockPollInterval" type="text" required placeholder="1m" pattern="[0-9]+(\.[0-9]+)?(ns|us|µs|μs|ms|s|m|h)([0-9]+(\.[0-9]+)?(ns|us|µs|μs|ms|s|m|h))*"><small class="mt-1 block text-xs text-energy-muted">{{ $t('settings.pollingRange') }}</small></label></div></fieldset>
        <fieldset class="rounded-xl border border-energy-border p-4"><legend class="px-1 font-display font-semibold text-energy-ink">{{ $t('settings.etaOutsideTemperature') }}</legend><p class="mb-4 text-sm text-energy-muted">{{ $t('settings.outdoorMeasurement') }}</p><div class="space-y-4"><label class="block text-sm font-medium text-slate-700">{{ $t('settings.deviceEndpoint') }}<input v-model="sourceSettings.etaOutsidetempURL" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" name="etaOutsidetempURL" type="url" autocomplete="url" required placeholder="http://eta.local/user/var/..."></label><label class="block text-sm font-medium text-slate-700">{{ $t('settings.pollInterval') }}<input v-model="sourceSettings.etaOutsidetempPollInterval" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" name="etaOutsidetempPollInterval" type="text" required placeholder="1m" pattern="[0-9]+(\.[0-9]+)?(ns|us|µs|μs|ms|s|m|h)([0-9]+(\.[0-9]+)?(ns|us|µs|μs|ms|s|m|h))*"><small class="mt-1 block text-xs text-energy-muted">{{ $t('settings.pollingRange') }}</small></label></div></fieldset>
      </div>
      <div class="flex flex-col justify-between gap-3 border-t border-energy-border pt-4 sm:flex-row sm:items-center"><p class="max-w-2xl text-sm text-energy-muted">{{ $t('settings.sourcesSavedDescription') }}</p><button class="rounded-lg bg-energy-ink px-4 py-2.5 text-sm font-semibold text-white hover:bg-slate-700 disabled:opacity-60" type="submit" :disabled="savingSources">{{ savingSources ? $t('common.saving') : $t('settings.saveConfiguration') }}</button></div>
    </form>
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-6">
    <div class="mb-5 flex flex-wrap items-start justify-between gap-3"><div><h2 class="font-display text-xl font-semibold text-energy-ink">{{ $t('settings.resourcePrices') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('settings.resourcePricesDescription') }}</p></div><span class="rounded-full bg-energy-heating/10 px-3 py-1 text-xs font-semibold text-energy-heating">{{ $t('settings.storedInPostgres') }}</span></div>
    <div class="space-y-3"><p v-if="domainError" class="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800" role="alert">{{ domainError }}</p><p v-if="domainStatus" class="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-800" role="status" aria-live="polite">{{ domainStatus }}</p></div>
    <form class="mt-5 space-y-5" @submit.prevent="saveDomainSettings">
      <div class="grid gap-4 lg:grid-cols-2">
        <fieldset class="rounded-xl border border-energy-border p-4"><legend class="px-1 font-display font-semibold text-energy-electricity">{{ $t('settings.electricity') }}</legend><p class="mb-4 text-sm text-energy-muted">{{ $t('settings.gridTariffDescription') }}</p><label class="block text-sm font-medium text-slate-700">{{ $t('settings.importTariff') }}<input v-model="domainSettings.electricityPricePerKWh" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-electricity focus:ring-2 focus:ring-energy-electricity/20" name="electricityPricePerKWh" type="number" min="0" step="0.0001" inputmode="decimal" :placeholder="$t('common.notConfigured')"></label></fieldset>
        <fieldset class="rounded-xl border border-energy-border p-4"><legend class="px-1 font-display font-semibold text-energy-water">{{ $t('settings.water') }}</legend><p class="mb-4 text-sm text-energy-muted">{{ $t('settings.waterRatesDescription') }}</p><div class="grid gap-4 sm:grid-cols-2"><label class="block text-sm font-medium text-slate-700">{{ $t('settings.waterRate') }}<input v-model="domainSettings.waterPricePerM3" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-water focus:ring-2 focus:ring-energy-water/20" name="waterPricePerM3" type="number" min="0" step="0.0001" inputmode="decimal" :placeholder="$t('common.notConfigured')"></label><label class="block text-sm font-medium text-slate-700">{{ $t('settings.wastewaterRate') }}<input v-model="domainSettings.wastewaterPricePerM3" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-water focus:ring-2 focus:ring-energy-water/20" name="wastewaterPricePerM3" type="number" min="0" step="0.0001" inputmode="decimal" :placeholder="$t('common.notConfigured')"></label></div></fieldset>
        <fieldset class="rounded-xl border border-energy-border p-4"><legend class="px-1 font-display font-semibold text-energy-heating">{{ $t('settings.pelletHeating') }}</legend><p class="mb-4 text-sm text-energy-muted">{{ $t('settings.capacityDescription') }}</p><label class="block text-sm font-medium text-slate-700">{{ $t('settings.tankCapacity') }}<input v-model="domainSettings.pelletTankCapacityKg" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-heating focus:ring-2 focus:ring-energy-heating/20" name="pelletTankCapacityKg" type="number" min="1" step="1" inputmode="decimal" :placeholder="$t('settings.optional')"></label></fieldset>
        <fieldset class="rounded-xl border border-energy-border p-4"><legend class="px-1 font-display font-semibold text-slate-700">{{ $t('settings.timeZone') }}</legend><p class="mb-4 text-sm text-energy-muted">{{ $t('settings.timeZoneDescription') }}</p><label class="block text-sm font-medium text-slate-700">{{ $t('settings.ianaTimeZone') }}<input v-model="domainSettings.timeZone" class="mt-1.5 w-full rounded-lg border border-slate-300 px-3 py-2 outline-none focus:border-energy-ink focus:ring-2 focus:ring-energy-ink/20" name="timeZone" type="text" required placeholder="Europe/Berlin"></label></fieldset>
      </div>
      <div class="flex flex-col justify-between gap-3 border-t border-energy-border pt-4 sm:flex-row sm:items-center"><p class="max-w-2xl text-sm text-energy-muted">{{ $t('settings.missingRatesDescription') }}</p><button class="rounded-lg bg-energy-ink px-4 py-2.5 text-sm font-semibold text-white hover:bg-slate-700 disabled:opacity-60" type="submit" :disabled="savingDomains">{{ savingDomains ? $t('common.saving') : $t('settings.saveResourceSettings') }}</button></div>
    </form>
  </section>

  <section class="rounded-xl border border-energy-border bg-white p-4 shadow-sm sm:p-6" aria-labelledby="ha-settings-title">
    <div class="flex flex-wrap items-start justify-between gap-3"><div><h2 id="ha-settings-title" class="font-display text-xl font-semibold text-energy-ink">{{ $t('settings.homeAssistantExport') }}</h2><p class="mt-1 text-sm text-energy-muted">{{ $t('settings.homeAssistantDescription') }} <code class="rounded bg-slate-100 px-1.5 py-0.5 text-xs">/api/v1/integrations/home-assistant/sensors</code>.</p></div><span class="rounded-full bg-slate-100 px-3 py-1 text-xs font-semibold text-slate-600">{{ $t('settings.readOnlyExport') }}</span></div>
    <p class="mt-4 rounded-lg bg-slate-50 p-4 text-sm leading-6 text-slate-600">{{ $t('settings.homeAssistantSetup', { token: 'HOME_ASSISTANT_READ_TOKEN', file: 'secrets.yaml' }) }}</p>
  </section>
</template>
