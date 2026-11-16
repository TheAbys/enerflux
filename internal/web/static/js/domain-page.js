const DAY_MS = 24 * 60 * 60 * 1000;
const COLORS = ["#168d78", "#637fd0", "#d39a35", "#8a6cda", "#d96e72", "#2d9cb3"];
let selectedDays = 7;
let readingChart = null;
let dailyChart = null;
let requestID = 0;

function number(value, digits = 2) {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? new Intl.NumberFormat(undefined, { maximumFractionDigits: digits }).format(parsed) : "—";
}

function valueWithUnit(value, unit) {
    return value === null || value === undefined ? "—" : `${number(value)}${unit ? ` ${unit}` : ""}`;
}

function dateTime(value) {
    return new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }).format(new Date(value));
}

function localDateTimeValue(timestamp) {
    const date = new Date(timestamp);
    const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
    return local.toISOString().slice(0, 16);
}

function rgb(hex, alpha) {
    const value = hex.replace("#", "");
    return `rgba(${parseInt(value.slice(0, 2), 16)}, ${parseInt(value.slice(2, 4), 16)}, ${parseInt(value.slice(4, 6), 16)}, ${alpha})`;
}

function setSummary(data, domain) {
    const consumed = data.consumed;
    const unit = domain === "water" ? "m³" : "kWh";
    document.getElementById("metric-value").textContent = valueWithUnit(consumed.value, unit);
    document.getElementById("coverage-note").textContent = consumed.value === null
        ? consumed.status
        : `${number(consumed.intervals, 0)} intervals · ${number(consumed.coveredHours, 1)}h covered · estimated`;
    document.getElementById("cost-value").textContent = valueWithUnit(consumed.costEur, "EUR");
    document.getElementById("cost-caption").textContent = consumed.pricePerUnit === null
        ? (domain === "water" ? "Water and wastewater rates are configured in EnerFlux." : "Electricity tariff is configured in EnerFlux.")
        : `${valueWithUnit(consumed.pricePerUnit, consumed.priceUnit)} · ${consumed.costEur === null ? "no usage estimate for this period" : "estimated for selected period"}`;

    const latest = domain === "water" ? data.currentReading : data.currentPower;
    document.getElementById("latest-value").textContent = latest ? valueWithUnit(latest.value, latest.unit) : "—";
    document.getElementById("latest-caption").textContent = latest ? `${latest.source || "Unknown source"} · ${dateTime(latest.ts)}` : "No recent source reading";
}

function renderReadingChart(points, domain) {
    if (readingChart) readingChart.destroy();
    document.getElementById("reading-empty").hidden = points.length > 0;
    const canvas = document.getElementById("reading-chart");
    const groups = new Map();
    for (const point of points) {
        const source = point.source || "unknown";
        if (!groups.has(source)) groups.set(source, []);
        groups.get(source).push(point);
    }
    const datasets = [...groups.entries()].map(([source, values], index) => {
        const color = COLORS[index % COLORS.length];
        return { label: source, data: values.map(point => ({ x: new Date(point.ts).getTime(), y: Number(point.value) })), borderColor: color, backgroundColor: rgb(color, .12), borderWidth: 2, pointRadius: values.length > 300 ? 0 : 2, pointHoverRadius: 4, fill: domain === "water", tension: .15, spanGaps: false };
    });
    readingChart = new Chart(canvas, {
        type: "line", data: { datasets }, options: {
            responsive: true, maintainAspectRatio: false, animation: false, parsing: false,
            plugins: { decimation: { enabled: true, algorithm: "lttb", samples: 1000, threshold: 1500 }, legend: { display: groups.size > 1, position: "bottom", align: "start" }, tooltip: { callbacks: { title: items => items.length ? dateTime(items[0].parsed.x) : "", label: context => ` ${context.dataset.label}: ${valueWithUnit(context.parsed.y, domain === "water" ? "m³" : "W")}` } } },
            scales: { x: { type: "linear", grid: { display: false }, ticks: { maxTicksLimit: 8, callback: value => new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short", hour: "2-digit" }).format(new Date(Number(value))) } }, y: { beginAtZero: false, ticks: { maxTicksLimit: 6, callback: value => valueWithUnit(Number(value), domain === "water" ? "m³" : "W") } } }
        }
    });
}

function renderDailyChart(points, domain) {
    if (dailyChart) dailyChart.destroy();
    document.getElementById("daily-empty").hidden = points.length > 0;
    const unit = domain === "water" ? "m³" : "kWh";
    const color = domain === "water" ? "#2b91b2" : "#7565dc";
    dailyChart = new Chart(document.getElementById("daily-chart"), {
        type: "bar",
        data: { labels: points.map(point => point.date), datasets: [{ label: domain === "water" ? "Water used" : "Grid energy", data: points.map(point => point.value), backgroundColor: rgb(color, .75), hoverBackgroundColor: color, borderRadius: 3, maxBarThickness: 24 }] },
        options: { responsive: true, maintainAspectRatio: false, animation: false, plugins: { legend: { display: false }, tooltip: { callbacks: { title: items => items.length ? items[0].label : "", label: context => ` ${number(context.parsed.y)} ${unit}` } } }, scales: { x: { grid: { display: false }, ticks: { maxTicksLimit: 10, callback: function (value) { const date = new Date(`${this.getLabelForValue(value)}T12:00:00`); return new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short" }).format(date); } } }, y: { beginAtZero: true, ticks: { callback: value => `${number(value)} ${unit}` } } } }
    });
}

function renderReadings(points, domain) {
    const body = document.getElementById("readings-body");
    body.replaceChildren();
    const sorted = [...points].sort((a, b) => new Date(b.ts) - new Date(a.ts));
    document.getElementById("reading-count").textContent = `${number(sorted.length, 0)} ${sorted.length === 1 ? "reading" : "readings"}`;
    document.getElementById("table-count").textContent = `${number(sorted.length, 0)} READINGS`;
    if (!sorted.length) {
        const row = document.createElement("tr");
        row.className = "empty-row";
        const cell = document.createElement("td");
        cell.colSpan = 3;
        cell.textContent = `No ${domain} readings in this period`;
        row.appendChild(cell);
        body.appendChild(row);
        return;
    }
    for (const point of sorted.slice(0, 250)) {
        const row = document.createElement("tr");
        const ts = document.createElement("td");
        const value = document.createElement("td");
        const source = document.createElement("td");
        ts.textContent = dateTime(point.ts);
        value.textContent = valueWithUnit(point.value, point.unit);
        source.textContent = point.source || "Unknown source";
        row.append(ts, value, source);
        body.appendChild(row);
    }
}

async function refreshDomainPage() {
    const request = ++requestID;
    const domain = document.body.dataset.domain;
    const to = Date.now();
    const from = to - selectedDays * DAY_MS;
    const query = new URLSearchParams({ from: new Date(from).toISOString(), to: new Date(to).toISOString() });
    document.getElementById("domain-error").hidden = true;
    document.getElementById("page-updated").textContent = "Updating…";
    try {
        const response = await fetch(`/api/v1/analytics/${domain}?${query}`, { headers: { Accept: "application/json" } });
        if (!response.ok) {
            const payload = await response.json().catch(() => ({}));
            throw new Error(payload.error || `Could not load ${domain} insights.`);
        }
        const data = await response.json();
        if (request !== requestID) return;
        setSummary(data, domain);
        const readings = domain === "water" ? data.readings : data.powerReadings;
        renderReadingChart(readings || [], domain);
        renderDailyChart(data.daily || [], domain);
        renderReadings(readings || [], domain);
        document.getElementById("period-caption").textContent = `Last ${selectedDays === 1 ? "24 hours" : `${selectedDays} days`}`;
        document.getElementById("range-description").textContent = `${new Intl.DateTimeFormat(undefined, { dateStyle: "medium" }).format(new Date(from))} – now`;
        document.getElementById("page-updated").textContent = `Updated ${new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" }).format(new Date())}`;
    } catch (error) {
        if (request !== requestID) return;
        const message = document.getElementById("domain-error");
        message.textContent = error.message || `Could not load ${domain} insights.`;
        message.hidden = false;
        document.getElementById("page-updated").textContent = "Update failed";
    }
}

function setLocalTimestamp(input) {
    input.value = localDateTimeValue(Date.now());
}

function setupManualEntry() {
    const form = document.getElementById("manual-entry-form");
    if (!form) return;
    const timestamp = document.getElementById("entry-time");
    const status = document.getElementById("entry-status");
    setLocalTimestamp(timestamp);
    timestamp.max = localDateTimeValue(Date.now());
    form.addEventListener("submit", async event => {
        event.preventDefault();
        if (!form.reportValidity()) return;
        const value = Number(document.getElementById("entry-value").value);
        const date = new Date(timestamp.value);
        const button = form.querySelector("button[type='submit']");
        if (!Number.isFinite(value) || value < 0 || !Number.isFinite(date.getTime())) return;
        button.disabled = true;
        status.dataset.state = "success";
        status.textContent = "Saving meter reading…";
        status.hidden = false;
        try {
            const response = await fetch("/api/v1/measurements", {
                method: "POST",
                headers: { "Content-Type": "application/json", Accept: "application/json" },
                body: JSON.stringify([{ ts: date.toISOString(), type: "water.meter.reading", value, unit: "m³", source: "manual" }])
            });
            if (!response.ok) {
                const payload = await response.json().catch(() => ({}));
                throw new Error(payload.error || "Could not save this water reading.");
            }
            form.reset();
            setLocalTimestamp(timestamp);
            status.textContent = "Water reading saved in EnerFlux.";
            await refreshDomainPage();
        } catch (error) {
            status.dataset.state = "error";
            status.textContent = error.message || "Could not save this water reading.";
        } finally {
            button.disabled = false;
        }
    });
}

document.addEventListener("DOMContentLoaded", () => {
    document.querySelectorAll(".domain-range-control .range-option").forEach(button => button.addEventListener("click", () => {
        selectedDays = Number(button.dataset.days);
        document.querySelectorAll(".domain-range-control .range-option").forEach(item => {
            const active = item === button;
            item.classList.toggle("active", active);
            item.setAttribute("aria-pressed", String(active));
        });
        refreshDomainPage();
    }));
    document.getElementById("page-refresh").addEventListener("click", refreshDomainPage);
    setupManualEntry();
    refreshDomainPage();
    setInterval(refreshDomainPage, 60000);
});
