let currentMetric = null;
let reloadTimer = null;

async function loadMetric(canvasId, metricKey) {
    currentMetric = metricKey;

    const response = await fetch(`/api/v1/metrics/?key=${encodeURIComponent(metricKey)}&limit=300`);
    const data = await response.json();

    renderChart(canvasId, data, metricKey);
}

function startAutoReload(canvasId) {
    if (reloadTimer) {
        clearInterval(reloadTimer);
    }

    reloadTimer = setInterval(() => {
        if (currentMetric) {
            loadMetric(canvasId, currentMetric);
        }
    }, 60000);
}

document.addEventListener("DOMContentLoaded", () => {
    const canvasId = "chart";
    const select = document.getElementById("metric");

    loadMetric(canvasId, select.value);
    startAutoReload(canvasId);

    select.addEventListener("change", e => {
        loadMetric(canvasId, e.target.value);
    });
});

let charts = {};

function renderChart(canvasId, data, title) {
    const timestamps = [...new Set(data.map(x => x.TS))]
        .sort((a, b) => new Date(a) - new Date(b));

    const labels = timestamps.map(ts =>
        new Date(ts).toLocaleTimeString()
    );

    const sources = [...new Set(data.map(x => x.Source))];

    const datasets = sources.map(source => {
        const valuesByTimestamp = new Map(
            data
                .filter(x => x.Source === source)
                .map(x => [x.TS, x.Value])
        );

        return {
            label: source,
            data: timestamps.map(ts => valuesByTimestamp.get(ts) ?? null),
            tension: 0.2,
            spanGaps: true
        };
    });

    const ctx = document.getElementById(canvasId);

    if (charts[canvasId]) {
        charts[canvasId].destroy();
    }

    charts[canvasId] = new Chart(ctx, {
        type: "line",
        data: {
            labels,
            datasets
        },
        options: {
            plugins: {
                title: {
                    display: true,
                    text: title
                }
            }
        }
    });
}