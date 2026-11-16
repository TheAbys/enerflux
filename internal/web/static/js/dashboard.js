const METRICS = {
    "water.meter.reading": { label: "Water meter total", unit: "m³", domain: "Water" },
    "grid.consumption.power": { label: "Grid power", unit: "W", domain: "Electricity" },
    "pv.power": { label: "Solar production", unit: "W", domain: "Electricity" },
    "pv.dc.power": { label: "Solar DC production", unit: "W", domain: "Electricity" },
    "device.power": { label: "Device power", unit: "W", domain: "Electricity" },
    "device.energy.total": { label: "Device cumulative energy", unit: "Wh", domain: "Electricity" },
    "heating.pellet.stock": { label: "Pellet stock", unit: "kg", domain: "Heating · pellets" },
    "climate.temperature.outside": { label: "Outside temperature", unit: "°C", domain: "Climate" }
};

const RANGES = { "15m": 15 * 60 * 1000, "1h": 60 * 60 * 1000, "24h": 24 * 60 * 60 * 1000, "7d": 7 * 24 * 60 * 60 * 1000 };
const COLORS = ["#159b83", "#5d82d8", "#df9f35", "#916ce0", "#df6e74", "#2da4b9"];
let chart = null;
let selectedRange = "15m";
let reloadTimer = null;
let requestSequence = 0;

function formatNumber(value, digits = 2) {
    const number = Number(value);
    return Number.isFinite(number) ? new Intl.NumberFormat(undefined, { maximumFractionDigits: digits }).format(number) : "—";
}

function formatValue(value, unit) {
    return value === null || value === undefined ? "—" : `${formatNumber(value, 2)}${unit ? ` ${unit}` : ""}`;
}

function formatTimestamp(timestamp, includeDate = true) {
    const date = new Date(timestamp);
    return new Intl.DateTimeFormat(undefined, includeDate
        ? { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", second: "2-digit" }
        : { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(date);
}

function selectedMetric() {
    const key = document.getElementById("metric").value;
    return { key, ...(METRICS[key] || { label: key, unit: "", domain: "Measurement" }) };
}

function chartWindow() {
    const to = Date.now();
    return { from: to - RANGES[selectedRange], to };
}

function sourceColor(source, index) {
    let hash = 0;
    for (let i = 0; i < source.length; i += 1) hash = ((hash << 5) - hash + source.charCodeAt(i)) | 0;
    return COLORS[Math.abs(hash + index) % COLORS.length];
}

function withAlpha(hex, alpha) {
    const value = hex.replace("#", "");
    const red = parseInt(value.slice(0, 2), 16);
    const green = parseInt(value.slice(2, 4), 16);
    const blue = parseInt(value.slice(4, 6), 16);
    return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}

function setDomainSummary(data) {
    const water = data.water;
    document.getElementById("overview-water-value").textContent = formatValue(water.consumed.value, "m³");
    document.getElementById("overview-water-cost").textContent = water.consumed.costEur === null ? "Water rates not configured" : `${formatValue(water.consumed.costEur, "EUR")} estimated · 30 days`;
    document.getElementById("overview-water-reading").textContent = water.currentReading
        ? `Latest meter total ${formatValue(water.currentReading.value, water.currentReading.unit)} · ${formatTimestamp(water.currentReading.ts)}`
        : "No meter readings stored";

    const electricity = data.electricity;
    document.getElementById("overview-electricity-value").textContent = formatValue(electricity.consumed.value, "kWh");
    document.getElementById("overview-electricity-cost").textContent = electricity.consumed.costEur === null ? "Electricity tariff not configured" : `${formatValue(electricity.consumed.costEur, "EUR")} estimated · today`;
    document.getElementById("overview-electricity-power").textContent = electricity.currentPower
        ? `Current grid power ${formatValue(electricity.currentPower.value, electricity.currentPower.unit)}`
        : "No grid power readings today";

    const heating = data.heating;
    document.getElementById("overview-heating-stock").textContent = heating.currentStock
        ? formatValue(heating.currentStock.value, heating.currentStock.unit)
        : "—";
    document.getElementById("overview-heating-usage").textContent = heating.consumedLast30DaysKg === null
        ? "Last 30 days: no estimate"
        : `Last 30 days: ${formatValue(heating.consumedLast30DaysKg, "kg")} · ${formatValue(heating.averageDaily30DaysKg, "kg/covered day")}`;
    document.getElementById("overview-heating-forecast").textContent = heating.forecastNextHourKg === null
        ? heating.forecastStatus
        : `Next hour estimate ${formatValue(heating.forecastNextHourKg, "kg")}`;

    const totalCost = data.totalCost;
    document.getElementById("total-cost-value").textContent = formatValue(totalCost.totalEur, "EUR");
    document.getElementById("total-cost-caption").textContent = totalCost.complete
        ? "Complete estimate across water, electricity, and pellet heating."
        : `Incomplete estimate — add rates and readings for ${totalCost.missingDomains.join(", ") || "all domains"}.`;
    document.getElementById("total-cost-water").textContent = formatValue(totalCost.waterEur, "EUR");
    document.getElementById("total-cost-electricity").textContent = formatValue(totalCost.electricityEur, "EUR");
    document.getElementById("total-cost-heating").textContent = formatValue(totalCost.heatingEur, "EUR");
}

function renderReadings(points, unit) {
    const body = document.getElementById("readings-body");
    body.replaceChildren();
    const latest = [...points].sort((a, b) => new Date(b.TS) - new Date(a.TS)).slice(0, 10);
    if (!latest.length) {
        const row = document.createElement("tr");
        row.className = "empty-row";
        const cell = document.createElement("td");
        cell.colSpan = 3;
        cell.textContent = "No readings in this range";
        row.appendChild(cell);
        body.appendChild(row);
        return;
    }
    for (const point of latest) {
        const row = document.createElement("tr");
        const time = document.createElement("td");
        time.textContent = formatTimestamp(point.TS);
        const source = document.createElement("td");
        source.textContent = point.Source || "Unknown source";
        const value = document.createElement("td");
        value.textContent = formatValue(point.Value, point.Unit || unit);
        row.append(time, source, value);
        body.appendChild(row);
    }
}

function renderChart(points, metric, range) {
    if (chart) chart.destroy();
    chart = null;
    const canvas = document.getElementById("chart");
    const empty = document.getElementById("chart-empty");
    document.getElementById("chart-title").textContent = metric.label;
    document.getElementById("chart-unit").textContent = metric.unit || "—";
    document.getElementById("chart-subtitle").textContent = `${metric.domain} observations · ${formatTimestamp(range.from)} – ${formatTimestamp(range.to)}`;
    document.getElementById("sample-count").textContent = `${formatNumber(points.length, 0)} ${points.length === 1 ? "reading" : "readings"}`;
    document.getElementById("readings-description").textContent = `Latest stored ${metric.domain.toLowerCase()} values for ${metric.label.toLowerCase()}`;
    empty.hidden = points.length > 0;
    if (!points.length) return;

    const groups = new Map();
    for (const point of points) {
        const source = point.Source || "unknown";
        if (!groups.has(source)) groups.set(source, []);
        groups.get(source).push(point);
    }
    const datasets = [...groups.entries()].map(([source, series], index) => {
        const color = sourceColor(source, index);
        return {
            label: source,
            data: series.map(point => ({ x: new Date(point.TS).getTime(), y: Number(point.Value) })),
            borderColor: color,
            backgroundColor: withAlpha(color, 0.12),
            borderWidth: 2,
            pointRadius: 0,
            pointHoverRadius: 4,
            pointHitRadius: 8,
            fill: "start",
            tension: 0.18,
            spanGaps: false
        };
    });
    const duration = range.to - range.from;
    chart = new Chart(canvas, {
        type: "line",
        data: { datasets },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            parsing: false,
            normalized: true,
            interaction: { mode: "nearest", intersect: false, axis: "x" },
            plugins: {
                decimation: { enabled: true, algorithm: "lttb", samples: 1400, threshold: 2000 },
                legend: { display: groups.size > 1, position: "bottom", align: "start", labels: { usePointStyle: true, pointStyle: "circle", padding: 15, color: "#788681", font: { family: "DM Sans", size: 12 } } },
                tooltip: {
                    backgroundColor: "#173632",
                    titleColor: "#d9eee8",
                    bodyColor: "#fff",
                    borderColor: "rgba(255,255,255,.12)",
                    borderWidth: 1,
                    padding: 10,
                    callbacks: {
                        title: items => items.length ? formatTimestamp(items[0].parsed.x) : "",
                        label: context => ` ${context.dataset.label}: ${formatValue(context.parsed.y, metric.unit)}`
                    }
                }
            },
            scales: {
                x: { type: "linear", min: range.from, max: range.to, grid: { display: false }, border: { display: false }, ticks: { maxTicksLimit: 8, maxRotation: 0, color: "#93a09a", font: { family: "DM Sans", size: 11 }, callback: value => new Intl.DateTimeFormat(undefined, duration <= 36 * 60 * 60 * 1000 ? { hour: "2-digit", minute: "2-digit" } : { day: "2-digit", month: "short", hour: "2-digit" }).format(new Date(Number(value))) } },
                y: { beginAtZero: false, border: { display: false }, grid: { color: "#edf1ef", drawTicks: false }, ticks: { maxTicksLimit: 6, padding: 9, color: "#93a09a", font: { family: "DM Sans", size: 11 }, callback: value => formatValue(Number(value), metric.unit) } }
            }
        }
    });
}

async function loadOverview() {
    const response = await fetch("/api/v1/analytics/overview", { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error("Could not load resource overview.");
    setDomainSummary(await response.json());
}

async function loadChart(request) {
    const metric = selectedMetric();
    const range = chartWindow();
    const query = new URLSearchParams({ key: metric.key, from: new Date(range.from).toISOString(), to: new Date(range.to).toISOString(), limit: "200000" });
    const response = await fetch(`/api/v1/metrics/?${query}`, { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error(`Could not load ${metric.label.toLowerCase()} measurements.`);
    const points = await response.json();
    if (request !== requestSequence) return;
    const chronological = Array.isArray(points) ? points.sort((a, b) => new Date(a.TS) - new Date(b.TS)) : [];
    renderChart(chronological, metric, range);
    renderReadings(chronological, metric.unit);
    document.getElementById("range-description").textContent = `Last ${selectedRange}`;
}

async function refreshDashboard() {
    const request = ++requestSequence;
    document.getElementById("overview-error").hidden = true;
    document.getElementById("chart-error").hidden = true;
    document.getElementById("last-updated").textContent = "Updating…";
    const results = await Promise.allSettled([loadOverview(), loadChart(request)]);
    if (request !== requestSequence) return;
    const overviewError = results[0].status === "rejected" ? results[0].reason : null;
    const chartError = results[1].status === "rejected" ? results[1].reason : null;
    document.getElementById("overview-error").textContent = overviewError?.message || "";
    document.getElementById("overview-error").hidden = !overviewError;
    document.getElementById("chart-error").textContent = chartError?.message || "";
    document.getElementById("chart-error").hidden = !chartError;
    document.getElementById("last-updated").textContent = `Updated ${new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(new Date())}`;
}

function setupControls() {
    document.querySelectorAll(".range-option").forEach(button => button.addEventListener("click", () => {
        selectedRange = button.dataset.range;
        document.querySelectorAll(".range-option").forEach(item => {
            const active = item === button;
            item.classList.toggle("active", active);
            item.setAttribute("aria-pressed", String(active));
        });
        refreshDashboard();
    }));
    document.getElementById("metric").addEventListener("change", refreshDashboard);
    document.getElementById("refresh-button").addEventListener("click", refreshDashboard);
    document.getElementById("auto-refresh").addEventListener("change", () => {
        if (reloadTimer) clearInterval(reloadTimer);
        reloadTimer = document.getElementById("auto-refresh").checked ? setInterval(refreshDashboard, 60000) : null;
    });
}

document.addEventListener("DOMContentLoaded", () => {
    setupControls();
    refreshDashboard();
    reloadTimer = setInterval(refreshDashboard, 60000);
    document.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "visible" && document.getElementById("auto-refresh").checked) refreshDashboard();
    });
});
