<script setup lang="ts">
import {
  BarController,
  BarElement,
  CategoryScale,
  Chart,
  Decimation,
  Filler,
  Legend,
  LineController,
  LineElement,
  LinearScale,
  PointElement,
  Tooltip,
  type ChartConfiguration,
} from "chart.js";
import { onBeforeUnmount, onMounted, ref, watch } from "vue";

Chart.register(
  BarController,
  BarElement,
  CategoryScale,
  Decimation,
  Filler,
  Legend,
  LineController,
  LineElement,
  LinearScale,
  PointElement,
  Tooltip,
);

const props = defineProps<{
  config: ChartConfiguration;
  label: string;
}>();

const canvas = ref<HTMLCanvasElement | null>(null);
let chart: Chart | null = null;

function renderChart(): void {
  if (!canvas.value) return;
  chart?.destroy();
  chart = new Chart(canvas.value, props.config);
}

onMounted(renderChart);
watch(() => props.config, renderChart, { deep: true });
onBeforeUnmount(() => chart?.destroy());
</script>

<template>
  <canvas ref="canvas" :aria-label="label" role="img"></canvas>
</template>
