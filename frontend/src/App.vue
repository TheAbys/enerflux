<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { RouterLink, RouterView, useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import AppSidebar from "./components/AppSidebar.vue";
import type { LocalizedMessage } from "./lib/i18n";
import type { PageMeta } from "./router";

const route = useRoute();
const { t, locale } = useI18n({ useScope: "global" });
const page = computed<PageMeta>(() => route.meta.titleKey
  ? route.meta as PageMeta
  : {
      titleKey: "navigation.overview",
      labelKey: "navigation.overview",
      rootKey: "app.workspace",
      statusKey: "app.resourceSummary",
      dot: true,
      updated: true,
    });
const updatedLabel = ref<LocalizedMessage>({ key: "common.loading" });

watch([() => page.value.titleKey, locale], ([titleKey]) => {
  document.title = `EnerFlux · ${t(titleKey)}`;
}, { immediate: true });

watch(() => page.value.titleKey, () => {
  updatedLabel.value = { key: "common.loading" };
}, { immediate: true });

function updateLabel(value: LocalizedMessage): void {
  updatedLabel.value = value;
}
</script>

<template>
  <div class="min-h-screen bg-energy-page font-sans text-slate-800 antialiased lg:flex">
    <AppSidebar />

    <main class="min-w-0 flex-1 px-4 py-5 sm:px-6 sm:py-7 lg:px-8 lg:py-8">
      <header class="mb-6 flex flex-wrap items-center justify-between gap-3 border-b border-energy-border pb-4">
        <div class="flex items-center gap-2 text-sm">
          <RouterLink v-if="page.rootTo" class="text-energy-muted transition hover:text-energy-ink" :to="page.rootTo">{{ t(page.rootKey) }}</RouterLink>
          <span v-else class="text-energy-muted">{{ t(page.rootKey) }}</span>
          <span class="text-slate-300" aria-hidden="true">/</span>
          <strong class="font-semibold text-energy-ink">{{ t(page.labelKey) }}</strong>
        </div>
        <div class="flex items-center gap-3 text-xs text-energy-muted">
          <span class="inline-flex items-center gap-2">
            <span v-if="page.dot" class="h-2 w-2 rounded-full bg-energy-heating" aria-hidden="true"></span>
            {{ t(page.statusKey) }}
          </span>
          <template v-if="page.updated">
            <span v-if="route.path === '/'" class="h-4 border-l border-energy-border" aria-hidden="true"></span>
            <span>{{ updatedLabel.params ? t(updatedLabel.key, updatedLabel.params) : t(updatedLabel.key) }}</span>
          </template>
        </div>
      </header>

      <RouterView v-slot="{ Component }">
        <div class="space-y-6">
          <component :is="Component" @updated="updateLabel" />
        </div>
      </RouterView>
    </main>
  </div>
</template>
