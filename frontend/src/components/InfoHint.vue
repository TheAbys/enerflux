<script setup lang="ts">
import { useId } from "vue";
import { CircleHelp } from "@lucide/vue";

defineProps<{
  label: string;
  text: string;
}>();

const hintId = useId();
</script>

<template>
  <span class="group relative inline-flex shrink-0">
    <button
      class="inline-flex rounded-full text-energy-muted transition hover:text-energy-ink focus:outline-none focus-visible:ring-2 focus-visible:ring-energy-water/50"
      type="button"
      :aria-label="label"
      :aria-describedby="hintId"
    >
      <CircleHelp class="h-4 w-4" aria-hidden="true" />
    </button>
    <span
      :id="hintId"
      class="pointer-events-none absolute bottom-full right-0 z-20 mb-2 w-64 translate-y-1 rounded-lg border border-energy-border bg-white p-3 text-left text-xs font-normal leading-5 text-slate-700 opacity-0 shadow-lg transition group-hover:translate-y-0 group-hover:opacity-100 group-focus-within:translate-y-0 group-focus-within:opacity-100"
      role="tooltip"
    >{{ text }}</span>
  </span>
</template>
