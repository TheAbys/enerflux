<script setup lang="ts">
import { computed } from "vue";
import InfoHint from "./InfoHint.vue";

const props = withDefaults(defineProps<{
  label: string;
  value: string;
  detail: string;
  hint?: string;
  hintLabel?: string;
  accent?: "water" | "electricity" | "heating" | "neutral";
}>(), { accent: "neutral" });

const accentClass = computed(() => ({
  water: "border-t-energy-water",
  electricity: "border-t-energy-electricity",
  heating: "border-t-energy-heating",
  neutral: "border-t-slate-300",
}[props.accent]));
</script>

<template>
  <article class="min-w-0 rounded-xl border border-t-4 border-energy-border bg-white p-5 shadow-sm" :class="accentClass">
    <div class="flex items-start justify-between gap-2">
      <p class="text-xs font-bold uppercase tracking-[.14em] text-energy-muted">{{ label }}</p>
      <InfoHint v-if="hint" :label="hintLabel ?? label" :text="hint" />
    </div>
    <p class="mt-3 break-words font-display text-2xl font-semibold text-energy-ink sm:text-3xl">{{ value }}</p>
    <p class="mt-2 text-sm leading-5 text-energy-muted">{{ detail }}</p>
  </article>
</template>
