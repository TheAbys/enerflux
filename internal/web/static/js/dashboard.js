async function loadMetric(canvasId, metricKey) {
    const response = await fetch(`/api/v1/metrics/?key=${encodeURIComponent(metricKey)}`);
    const data = await response.json();

    renderChart(canvasId, data, metricKey);
}

let charts = {};

function renderChart(canvasId, data, title) {
    const labels = data.map(x =>
        new Date(x.TS).toLocaleTimeString()
    );

    const values = data.map(x => x.Value);

    const ctx = document.getElementById(canvasId);

    if (charts[canvasId]) {
        charts[canvasId].destroy();
    }

    charts[canvasId] = new Chart(ctx, {
        type: "line",
        data: {
            labels,
            datasets: [{
                label: title,
                data: values,
                tension: 0.2
            }]
        }
    });
}