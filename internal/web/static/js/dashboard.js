const METRICS = {
    "grid.consumption.power": { label: "Grid consumption", group: "Power", unit: "W" },
    "pv.power": { label: "Solar production", group: "Power", unit: "W" },
    "pv.dc.power": { label: "Solar DC production", group: "Power", unit: "W" },
    "device.power": { label: "Device power", group: "Power", unit: "W" },
    "heating.pellet.stock": { label: "Pellet stock", group: "Heating & climate", unit: "" },
    "climate.temperature.outside": { label: "Outside temperature", group: "Heating & climate", unit: "°C" },
    "device.energy.total": { label: "Device energy total", group: "Energy", unit: "Wh" },
    "energy.yield.today": { label: "Solar yield today", group: "Energy", unit: "Wh" },
    "energy.yield.yesterday": { label: "Solar yield yesterday", group: "Energy", unit: "Wh" }
};

const SERIES_COLORS = ["#159b83", "#5d82d8", "#df9f35", "#916ce0", "#df6e74", "#2da4b9"];
const REFRESH_INTERVAL_MS = 60000;
const MAX_INTERPOLATION_GAP_MS = 15 * 60 * 1000;
const PELLET_CAPACITY_KEY = "enerflux.pelletCapacityKg";

let chart = null;
let currentRange = { from: Date.now() - 15 * 60 * 1000, to: Date.now(), label: "Last 15 minutes", live: true };
let reloadTimer = null;
let requestSequence = 0;
let cachedPelletPoints = [];

function formatNumber(value, maximumFractionDigits = 2) {
    if (!Number.isFinite(value)) return "—";
    return new Intl.NumberFormat(undefined, { maximumFractionDigits, minimumFractionDigits: 0 }).format(value);
}

function formatValue(value, unit = "") {
    if (!Number.isFinite(Number(value))) return "—";
    const numeric = Number(value);
    const digits = Math.abs(numeric) < 10 && !Number.isInteger(numeric) ? 2 : 1;
    return `${formatNumber(numeric, digits)}${unit ? ` ${unit}` : ""}`;
}

function formatEnergy(kWh) {
    if (!Number.isFinite(kWh)) return "—";
    if (Math.abs(kWh) < 0.05) return `${formatNumber(kWh * 1000, 0)} Wh`;
    return `${formatNumber(kWh, 2)} kWh`;
}

function formatTimestamp(timestamp, includeDate = true) {
    const date = new Date(timestamp);
    return new Intl.DateTimeFormat(undefined, includeDate
        ? { day: "2-digit", month: "short", hour: "2-digit", minute: "2-digit", second: "2-digit" }
        : { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(date);
}

function formatAxisTimestamp(timestamp, duration) {
    const date = new Date(timestamp);
    const options = duration <= 36 * 60 * 60 * 1000
        ? { hour: "2-digit", minute: "2-digit" }
        : { day: "2-digit", month: "short", hour: "2-digit" };
    return new Intl.DateTimeFormat(undefined, options).format(date);
}

function metricLabel(key) {
    return METRICS[key]?.label || key;
}

function currentMetricKey() {
    return document.getElementById("metric").value;
}

async function fetchMetric(key, from, to) {
    const query = new URLSearchParams({
        key,
        from: new Date(from).toISOString(),
        to: new Date(to).toISOString(),
        limit: "200000"
    });
    const response = await fetch(`/api/v1/metrics/?${query.toString()}`, { headers: { Accept: "application/json" } });
    if (!response.ok) {
        let message = `Could not load ${metricLabel(key)}.`;
        try {
            const body = await response.json();
            if (body.error) message = body.error;
        } catch (_) {
            // Keep the useful fallback if the response is not JSON.
        }
        throw new Error(message);
    }
    const rows = await response.json();
    return Array.isArray(rows) ? rows.sort((a, b) => new Date(a.TS) - new Date(b.TS)) : [];
}

function groupBySource(points) {
    const groups = new Map();
    for (const point of points) {
        const source = point.Source || "unknown";
        if (!groups.has(source)) groups.set(source, []);
        groups.get(source).push(point);
    }
    for (const series of groups.values()) {
        series.sort((a, b) => new Date(a.TS) - new Date(b.TS));
    }
    return groups;
}

function colorForSource(source, index) {
    let hash = 0;
    for (let i = 0; i < source.length; i += 1) hash = ((hash << 5) - hash + source.charCodeAt(i)) | 0;
    return SERIES_COLORS[Math.abs(hash + index) % SERIES_COLORS.length];
}

function withAlpha(hex, alpha) {
    const value = hex.replace("#", "");
    const red = parseInt(value.slice(0, 2), 16);
    const green = parseInt(value.slice(2, 4), 16);
    const blue = parseInt(value.slice(4, 6), 16);
    return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}

function renderChart(points, key, range) {
    const canvas = document.getElementById("chart");
    const emptyState = document.getElementById("chart-empty");
    const duration = range.to - range.from;
    const groups = groupBySource(points);
    const unit = points.find(point => point.Unit)?.Unit || METRICS[key]?.unit || "";
    const heading = metricLabel(key);

    document.getElementById("chart-title").textContent = heading;
    document.getElementById("chart-unit").textContent = unit || "—";
    document.getElementById("chart-subtitle").textContent = `Readings ${range.label.toLowerCase()}`;
    document.getElementById("sample-count").textContent = `${formatNumber(points.length, 0)} ${points.length === 1 ? "reading" : "readings"}`;

    if (chart) {
        chart.destroy();
        chart = null;
    }

    if (points.length === 0) {
        emptyState.hidden = false;
        return;
    }

    emptyState.hidden = true;
    const datasets = [...groups.entries()].map(([source, series], index) => {
        const color = colorForSource(source, index);
        return {
            label: source,
            data: series.map(point => ({ x: new Date(point.TS).getTime(), y: Number(point.Value) })),
            borderColor: color,
            backgroundColor: withAlpha(color, 0.11),
            borderWidth: 2,
            pointRadius: 0,
            pointHoverRadius: 4,
            pointHitRadius: 8,
            fill: "start",
            tension: 0.2,
            spanGaps: false
        };
    });

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
                legend: {
                    display: groups.size > 1,
                    position: "bottom",
                    align: "start",
                    labels: { usePointStyle: true, pointStyle: "circle", boxWidth: 7, boxHeight: 7, padding: 17, color: "#788681", font: { family: "DM Sans", size: 10 } }
                },
                tooltip: {
                    backgroundColor: "#173632",
                    titleColor: "#d9eee8",
                    bodyColor: "#fff",
                    borderColor: "rgba(255,255,255,.12)",
                    borderWidth: 1,
                    padding: 10,
                    displayColors: true,
                    usePointStyle: true,
                    callbacks: {
                        title: items => items.length ? formatTimestamp(items[0].parsed.x) : "",
                        label: context => ` ${context.dataset.label}: ${formatValue(context.parsed.y, unit)}`
                    }
                }
            },
            scales: {
                x: {
                    type: "linear",
                    min: range.from,
                    max: range.to,
                    grid: { display: false },
                    border: { display: false },
                    ticks: {
                        maxTicksLimit: 8,
                        maxRotation: 0,
                        color: "#93a09a",
                        font: { family: "DM Sans", size: 9 },
                        callback: value => formatAxisTimestamp(Number(value), duration)
                    }
                },
                y: {
                    beginAtZero: false,
                    border: { display: false },
                    grid: { color: "#edf1ef", drawTicks: false },
                    ticks: {
                        maxTicksLimit: 6,
                        padding: 9,
                        color: "#93a09a",
                        font: { family: "DM Sans", size: 9 },
                        callback: value => `${formatNumber(Number(value), 1)}${unit ? ` ${unit}` : ""}`
                    }
                }
            }
        }
    });
}

function updateLatest(points) {
    const latest = points.reduce((newest, point) => !newest || new Date(point.TS) > new Date(newest.TS) ? point : newest, null);
    const value = document.getElementById("latest-value");
    const caption = document.getElementById("latest-caption");
    if (!latest) {
        value.textContent = "—";
        caption.textContent = "No measurements in this range";
        return;
    }
    value.textContent = formatValue(Number(latest.Value), latest.Unit || METRICS[currentMetricKey()]?.unit || "");
    caption.textContent = `${latest.Source || "Unknown source"} · ${formatTimestamp(latest.TS)}`;
}

function powerToWatts(point) {
    const value = Number(point.Value);
    const unit = String(point.Unit || "").trim().toLowerCase();
    if (unit === "w" || unit === "watt" || unit === "watts") return value;
    if (unit === "kw" || unit === "kilowatt" || unit === "kilowatts") return value * 1000;
    return null;
}

function interpolate(startValue, endValue, fraction) {
    return startValue + (endValue - startValue) * fraction;
}

function calculateEnergy(points, from, to) {
    const groups = groupBySource(points);
    let wattHours = 0;
    let segments = 0;

    for (const series of groups.values()) {
        for (let index = 1; index < series.length; index += 1) {
            const left = series[index - 1];
            const right = series[index];
            const leftTime = new Date(left.TS).getTime();
            const rightTime = new Date(right.TS).getTime();
            const gap = rightTime - leftTime;
            if (gap <= 0 || gap > MAX_INTERPOLATION_GAP_MS) continue;
            const start = Math.max(leftTime, from);
            const end = Math.min(rightTime, to);
            if (end <= start) continue;

            const leftWatts = powerToWatts(left);
            const rightWatts = powerToWatts(right);
            if (leftWatts === null || rightWatts === null) continue;

            const startFraction = (start - leftTime) / gap;
            const endFraction = (end - leftTime) / gap;
            const startPower = Math.max(0, interpolate(leftWatts, rightWatts, startFraction));
            const endPower = Math.max(0, interpolate(leftWatts, rightWatts, endFraction));
            wattHours += ((startPower + endPower) / 2) * ((end - start) / 3600000);
            segments += 1;
        }
    }
    return { kWh: wattHours / 1000, segments };
}

function stockUnitIsPercent(unit) {
    const normalized = String(unit || "").trim().toLowerCase();
    return normalized === "%" || normalized === "percent" || normalized === "percentage";
}

function calculatePelletUse(points, from, to) {
    const groups = groupBySource(points);
    const units = points.map(point => point.Unit).filter(Boolean);
    const unit = units[0] || "";
    let used = 0;
    let segments = 0;

    for (const series of groups.values()) {
        for (let index = 1; index < series.length; index += 1) {
            const left = series[index - 1];
            const right = series[index];
            const leftTime = new Date(left.TS).getTime();
            const rightTime = new Date(right.TS).getTime();
            const elapsed = rightTime - leftTime;
            if (elapsed <= 0 || elapsed > MAX_INTERPOLATION_GAP_MS) continue;
            const start = Math.max(leftTime, from);
            const end = Math.min(rightTime, to);
            if (end <= start) continue;

            const decrease = Number(left.Value) - Number(right.Value);
            if (decrease <= 0) continue;
            used += decrease * ((end - start) / elapsed);
            segments += 1;
        }
    }

    return { used, unit, segments };
}

function updateEnergyCard(points, range) {
    const result = calculateEnergy(points, range.from, range.to);
    document.getElementById("energy-value").textContent = result.segments ? formatEnergy(result.kWh) : "—";
    document.getElementById("energy-caption").textContent = result.segments
        ? `Estimated from ${formatNumber(result.segments, 0)} measured intervals · gaps over 15 min omitted`
        : "Need consecutive grid power readings in this range";
}

function updatePelletCard(points, range) {
    const result = calculatePelletUse(points, range.from, range.to);
    const calibration = document.getElementById("pellet-calibration");
    const capacityInput = document.getElementById("tank-capacity");
    const savedCapacity = Number(localStorage.getItem(PELLET_CAPACITY_KEY));
    let value = result.used;
    let unit = result.unit;
    let description = "Estimated from stock decreases";

    if (stockUnitIsPercent(result.unit)) {
        calibration.hidden = false;
        if (Number.isFinite(savedCapacity) && savedCapacity > 0) {
            capacityInput.value = String(savedCapacity);
            value = (result.used / 100) * savedCapacity;
            unit = "kg";
        } else {
            unit = "pp";
            description = "Percentage points · set tank capacity to convert to kg";
        }
    } else {
        calibration.hidden = true;
        const normalizedUnit = String(result.unit || "").trim().toLowerCase();
        if (normalizedUnit === "t" || normalizedUnit === "ton" || normalizedUnit === "tonne") {
            value *= 1000;
            unit = "kg";
        }
    }

    document.getElementById("pellet-value").textContent = result.segments ? formatValue(value, unit) : "—";
    document.getElementById("pellet-caption").textContent = result.segments
        ? `${description} · ${range.label.toLowerCase()}`
        : "Need stock readings across this time range";
}

function renderReadings(points) {
    const body = document.getElementById("readings-body");
    const latest = [...points].sort((a, b) => new Date(b.TS) - new Date(a.TS)).slice(0, 10);
    body.replaceChildren();

    if (!latest.length) {
        const row = document.createElement("tr");
        row.className = "empty-row";
        const cell = document.createElement("td");
        cell.colSpan = 3;
        cell.textContent = "No measurements in this range";
        row.appendChild(cell);
        body.appendChild(row);
        return;
    }

    for (const point of latest) {
        const row = document.createElement("tr");
        const timestamp = document.createElement("td");
        timestamp.textContent = formatTimestamp(point.TS);
        const source = document.createElement("td");
        const sourceContent = document.createElement("span");
        sourceContent.className = "source-cell";
        const dot = document.createElement("span");
        dot.className = "source-dot";
        dot.setAttribute("aria-hidden", "true");
        const sourceName = document.createElement("span");
        sourceName.textContent = point.Source || "Unknown source";
        sourceContent.append(dot, sourceName);
        source.appendChild(sourceContent);
        const value = document.createElement("td");
        value.textContent = formatValue(Number(point.Value), point.Unit || "");
        row.append(timestamp, source, value);
        body.appendChild(row);
    }
}

function updateRangeFooter(range) {
    document.getElementById("range-description").textContent = `${range.label} · ${formatTimestamp(range.from)} – ${formatTimestamp(range.to)}`;
}

async function refreshDashboard() {
    const sequence = ++requestSequence;
    const selectedKey = currentMetricKey();
    if (currentRange.live) {
        const duration = currentRange.to - currentRange.from;
        currentRange.to = Date.now();
        currentRange.from = currentRange.to - duration;
    }
    const range = { ...currentRange };
    const metricFrom = range.from;
    const statsFrom = range.from - MAX_INTERPOLATION_GAP_MS;
    const energyKey = "grid.consumption.power";
    const pelletKey = "heating.pellet.stock";

    document.getElementById("load-error").hidden = true;
    document.getElementById("last-updated").textContent = "Updating…";

    try {
        const energyPromise = fetchMetric(energyKey, statsFrom, range.to);
        const pelletPromise = fetchMetric(pelletKey, statsFrom, range.to);
        const selectedPromise = selectedKey === energyKey
            ? energyPromise
            : selectedKey === pelletKey
                ? pelletPromise
                : fetchMetric(selectedKey, metricFrom, range.to);
        const [selectedPoints, energyPoints, pelletPoints] = await Promise.all([selectedPromise, energyPromise, pelletPromise]);
        if (sequence !== requestSequence) return;
        cachedPelletPoints = pelletPoints;

        const displayPoints = selectedPoints.filter(point => {
            const timestamp = new Date(point.TS).getTime();
            return timestamp >= range.from && timestamp <= range.to;
        });
        renderChart(displayPoints, selectedKey, range);
        updateLatest(displayPoints);
        updateEnergyCard(energyPoints, range);
        updatePelletCard(pelletPoints, range);
        renderReadings(displayPoints);
        updateRangeFooter(range);
        document.getElementById("last-updated").textContent = `Updated ${new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(new Date())}`;
    } catch (error) {
        if (sequence !== requestSequence) return;
        document.getElementById("last-updated").textContent = "Update failed";
        const errorElement = document.getElementById("load-error");
        errorElement.textContent = error.message || "Could not load measurements. Please try again.";
        errorElement.hidden = false;
    }
}

function setRange(from, to, label, live = false) {
    currentRange = { from, to, label, live };
    refreshDashboard();
}

function toLocalInputValue(timestamp) {
    const date = new Date(timestamp);
    const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
    return local.toISOString().slice(0, 16);
}

function setupControls() {
    document.querySelectorAll(".range-option").forEach(button => {
        button.addEventListener("click", () => {
            const seconds = Number(button.dataset.range);
            const label = seconds < 3600 ? `Last ${seconds / 60} minutes`
                : seconds < 86400 ? `Last ${seconds / 3600} hour${seconds === 3600 ? "" : "s"}`
                    : `Last ${seconds / 86400} day${seconds === 86400 ? "" : "s"}`;
            document.querySelectorAll(".range-option").forEach(option => {
                const active = option === button;
                option.classList.toggle("active", active);
                option.setAttribute("aria-pressed", String(active));
            });
            const end = Date.now();
            setRange(end - seconds * 1000, end, label, true);
        });
    });

    const customToggle = document.getElementById("custom-range-toggle");
    const customPanel = document.getElementById("custom-range-panel");
    customToggle.addEventListener("click", () => {
        const open = customToggle.getAttribute("aria-expanded") !== "true";
        customToggle.setAttribute("aria-expanded", String(open));
        customPanel.hidden = !open;
        if (open) {
            document.getElementById("range-from").value = toLocalInputValue(currentRange.from);
            document.getElementById("range-to").value = toLocalInputValue(currentRange.to);
        }
    });

    customPanel.addEventListener("submit", event => {
        event.preventDefault();
        const from = new Date(document.getElementById("range-from").value).getTime();
        const to = new Date(document.getElementById("range-to").value).getTime();
        if (!Number.isFinite(from) || !Number.isFinite(to) || from >= to) {
            const errorElement = document.getElementById("load-error");
            errorElement.textContent = "Choose a start time before the end time.";
            errorElement.hidden = false;
            return;
        }
        document.querySelectorAll(".range-option").forEach(option => {
            option.classList.remove("active");
            option.setAttribute("aria-pressed", "false");
        });
        const customLabel = `Custom · ${formatTimestamp(from)} – ${formatTimestamp(to)}`;
        customToggle.setAttribute("aria-expanded", "false");
        customPanel.hidden = true;
        setRange(from, to, customLabel, false);
    });

    document.getElementById("metric").addEventListener("change", refreshDashboard);
    document.getElementById("refresh-button").addEventListener("click", refreshDashboard);
    document.getElementById("auto-refresh").addEventListener("change", setupAutoRefresh);
    document.getElementById("save-capacity").addEventListener("click", () => {
        const input = document.getElementById("tank-capacity");
        const capacity = Number(input.value);
        if (!Number.isFinite(capacity) || capacity <= 0) {
            input.setCustomValidity("Enter a tank capacity greater than zero.");
            input.reportValidity();
            return;
        }
        input.setCustomValidity("");
        localStorage.setItem(PELLET_CAPACITY_KEY, String(capacity));
        updatePelletCard(cachedPelletPoints, currentRange);
    });

    const savedCapacity = localStorage.getItem(PELLET_CAPACITY_KEY);
    if (savedCapacity) document.getElementById("tank-capacity").value = savedCapacity;
}

function setupAutoRefresh() {
    if (reloadTimer) {
        clearInterval(reloadTimer);
        reloadTimer = null;
    }
    if (document.getElementById("auto-refresh").checked) {
        reloadTimer = setInterval(() => {
            refreshDashboard();
        }, REFRESH_INTERVAL_MS);
    }
}

document.addEventListener("DOMContentLoaded", () => {
    setupControls();
    setupAutoRefresh();
    refreshDashboard();
    document.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "visible" && document.getElementById("auto-refresh").checked) refreshDashboard();
    });
});
