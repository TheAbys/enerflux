<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(defineProps<{
  eyebrow: string;
  title: string;
  description: string;
  accent?: "water" | "electricity" | "heating" | "neutral";
}>(), { accent: "neutral" });

const accentClass = computed(() => ({
  water: "text-energy-water",
  electricity: "text-energy-electricity",
  heating: "text-energy-heating",
  neutral: "text-energy-muted",
}[props.accent]));
</script>

<template>
  <section class="flex flex-col justify-between gap-4 md:flex-row md:items-end">
    <div>
      <p class="text-xs font-bold uppercase tracking-[.18em]" :class="accentClass">{{ eyebrow }}</p>
      <h1 class="mt-2 font-display text-3xl font-semibold tracking-tight text-energy-ink sm:text-4xl">{{ title }}</h1>
      <p class="mt-2 max-w-3xl text-sm leading-6 text-energy-muted">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="shrink-0"><slot name="actions" /></div>
  </section>
</template>
