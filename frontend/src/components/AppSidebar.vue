<script setup lang="ts">
import { RouterLink, useRoute } from "vue-router";
import { useI18n } from "vue-i18n";
import { Droplets, Flame, House, Settings2, Zap } from "@lucide/vue";

const route = useRoute();
const { locale, t } = useI18n({ useScope: "global" });

const links = [
  { to: "/", labelKey: "navigation.overview", icon: House },
  { to: "/water", labelKey: "navigation.water", icon: Droplets },
  { to: "/electricity", labelKey: "navigation.electricity", icon: Zap },
  { to: "/heating", labelKey: "navigation.heating", icon: Flame },
  { to: "/settings", labelKey: "navigation.configuration", icon: Settings2 },
] as const;

function linkClass(to: string): string {
  return route.path === to
    ? "bg-energy-ink text-white"
    : "text-slate-600 hover:bg-slate-100 hover:text-slate-900";
}
</script>

<template>
  <aside class="flex shrink-0 flex-col border-b border-slate-200 bg-white px-4 py-3 lg:sticky lg:top-0 lg:h-screen lg:w-64 lg:border-b-0 lg:border-r lg:px-5 lg:py-6" :aria-label="t('navigation.main')">
    <div class="flex items-center justify-between lg:block">
      <RouterLink class="flex items-center gap-2 text-energy-ink" to="/" :aria-label="t('navigation.home')">
        <svg class="h-8 w-8 text-energy-water" viewBox="0 0 32 32" fill="none" aria-hidden="true"><path d="M17.8 3.5 7.1 17.1h7l-.9 11.4 11.7-15h-7.2l.1-10Z" fill="currentColor" /></svg>
        <span class="font-display text-xl font-bold tracking-tight">enerflux<span class="text-energy-water">.</span></span>
      </RouterLink>
      <p class="mt-8 hidden text-xs font-bold tracking-[.18em] text-slate-400 lg:block">{{ t('navigation.workspace') }}</p>
      <label class="ml-4 block lg:ml-0 lg:mt-4">
        <span class="sr-only">{{ t('navigation.language') }}</span>
        <select v-model="locale" class="rounded-lg border border-slate-300 bg-white px-2.5 py-2 text-sm font-medium text-slate-700 outline-none focus:border-energy-water focus:ring-2 focus:ring-energy-water/20" :aria-label="t('navigation.language')">
          <option value="de">{{ t('navigation.german') }}</option>
          <option value="en">{{ t('navigation.english') }}</option>
        </select>
      </label>
    </div>

    <nav class="mt-3 flex gap-1 overflow-x-auto lg:mt-5 lg:flex-col" :aria-label="t('navigation.pages')">
      <RouterLink
        v-for="link in links"
        :key="link.to"
        :to="link.to"
        :aria-current="route.path === link.to ? 'page' : undefined"
        class="flex min-w-max items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-semibold transition"
        :class="linkClass(link.to)"
      >
        <component :is="link.icon" class="h-5 w-5 shrink-0" aria-hidden="true" />
        <span>{{ t(link.labelKey) }}</span>
      </RouterLink>
    </nav>

    <div class="mt-auto hidden border-t border-slate-200 pt-5 lg:block">
      <div class="flex items-center gap-3"><span class="h-2.5 w-2.5 rounded-full bg-energy-heating"></span><span><strong class="block text-sm text-slate-800">EnerFlux</strong><small class="text-xs text-slate-500">{{ t('navigation.domains') }}</small></span></div>
      <div class="mt-5 text-[10px] font-bold tracking-[.18em] text-slate-400">{{ t('navigation.tagline') }}</div>
    </div>
  </aside>
</template>
