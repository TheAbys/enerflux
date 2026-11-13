const STOCK_TYPE = "heating.pellet.stock";
const PURCHASE_QUANTITY_TYPE = "heating.pellet.purchase.quantity";
const PURCHASE_COST_TYPE = "heating.pellet.purchase.cost";
const SOURCE_ENDPOINT = "/api/v1/measurements";
const LIVE_SOURCE = "eta";
const MANUAL_SOURCE = "manual";
const FORECAST_WINDOW_MS = 6 * 60 * 60 * 1000;
const FORECAST_MAX_AGE_MS = 20 * 60 * 1000;
const HOUR_MS = 60 * 60 * 1000;
const DAY_MS = 24 * HOUR_MS;

let stockChart = null;
let usageChart = null;
let selectedDays = 30;
let refreshTimer = null;
let activeRequest = 0;

function formatNumber(value, digits = 1) {
    if (!Number.isFinite(value)) return "—";
    return new Intl.NumberFormat(undefined, { maximumFractionDigits: digits, minimumFractionDigits: 0 }).format(value);
}

function formatKg(value) {
    return `${formatNumber(value, 1)} kg`;
}

function formatDate(timestamp, includeTime = false) {
    const options = includeTime
        ? { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }
        : { day: "2-digit", month: "short", year: "numeric" };
    return new Intl.DateTimeFormat(undefined, options).format(new Date(timestamp));
}

function localInputValue(timestamp) {
    const date = new Date(timestamp);
    const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
    return local.toISOString().slice(0, 16);
}

function parseTimestamp(point) {
    return new Date(point.ts).getTime();
}

function makeQuery(types, from, to, limit, source = "") {
    const query = new URLSearchParams();
    for (const type of types) query.append("type", type);
    if (from !== null && from !== undefined) query.set("from", new Date(from).toISOString());
    if (to !== null && to !== undefined) query.set("to", new Date(to).toISOString());
    if (limit) query.set("limit", String(limit));
    if (source) query.append("source", source);
    return query;
}

async function fetchMeasurements(types, from, to, limit = 200000, source = "") {
    const query = makeQuery(types, from, to, limit, source);
    const response = await fetch(`${SOURCE_ENDPOINT}?${query}`, { headers: { Accept: "application/json" } });
    if (!response.ok) {
        let message = "Could not load pellet measurements.";
        try {
            const body = await response.json();
            if (body.error) message = body.error;
        } catch (_) {
            // Keep the fallback message for non-JSON responses.
        }
        throw new Error(message);
    }
    const data = await response.json();
    return Array.isArray(data) ? data.sort((a, b) => parseTimestamp(a) - parseTimestamp(b)) : [];
}

async function fetchLatest(type, to, source = "") {
    const query = makeQuery([type], null, to, 1, source);
    const response = await fetch(`${SOURCE_ENDPOINT}/latest?${query}`, { headers: { Accept: "application/json" } });
    if (!response.ok) throw new Error("Could not load the previous stock reading.");
    const data = await response.json();
    return data && typeof data === "object" ? data : null;
}

function rangeForSelection(now) {
    if (selectedDays === "all") return { from: null, to: now, label: "All recorded history" };
    const from = now - Number(selectedDays) * DAY_MS;
    return { from, to: now, label: `Last ${selectedDays} days` };
}

function normalizeStockPoints(points) {
    const sorted = [...points]
        .filter(point => Number.isFinite(Number(point.value)) && Number.isFinite(parseTimestamp(point)))
        .sort((a, b) => parseTimestamp(a) - parseTimestamp(b));
    const normalized = [];

    for (const point of sorted) {
        const previous = normalized[normalized.length - 1];
        if (previous && previous.source !== point.source && Math.abs(parseTimestamp(point) - parseTimestamp(previous)) <= 15 * 60 * 1000) {
            if (point.source === LIVE_SOURCE && previous.source !== LIVE_SOURCE) normalized[normalized.length - 1] = point;
            continue;
        }
        normalized.push(point);
    }
    return normalized;
}

function consumptionInRange(points, from, to) {
    let kilograms = 0;
    let intervals = 0;
    let coveredMs = 0;

    for (let index = 1; index < points.length; index += 1) {
        const left = points[index - 1];
        const right = points[index];
        const leftTime = parseTimestamp(left);
        const rightTime = parseTimestamp(right);
        const span = rightTime - leftTime;
        if (span <= 0) continue;

        const start = from === null ? leftTime : Math.max(leftTime, from);
        const end = Math.min(rightTime, to);
        if (end <= start) continue;

        const decrease = Number(left.value) - Number(right.value);
        if (decrease <= 0) continue;

        const overlap = end - start;
        kilograms += decrease * (overlap / span);
        coveredMs += overlap;
        intervals += 1;
    }

    return { kilograms, intervals, coveredMs };
}

function estimatedDailyUse(points, from, to) {
    const buckets = new Map();

    for (let index = 1; index < points.length; index += 1) {
        const left = points[index - 1];
        const right = points[index];
        const leftTime = parseTimestamp(left);
        const rightTime = parseTimestamp(right);
        const span = rightTime - leftTime;
        const decrease = Number(left.value) - Number(right.value);
        if (span <= 0 || decrease <= 0) continue;

        const start = from === null ? leftTime : Math.max(leftTime, from);
        const end = Math.min(rightTime, to);
        if (end <= start) continue;

        const firstDay = new Date(start);
        firstDay.setHours(0, 0, 0, 0);
        for (let dayStart = firstDay.getTime(); dayStart < end;) {
            const nextDay = new Date(dayStart);
            nextDay.setDate(nextDay.getDate() + 1);
            const dayEnd = nextDay.getTime();
            const overlapStart = Math.max(start, dayStart);
            const overlapEnd = Math.min(end, dayEnd);
            if (overlapEnd > overlapStart) {
                const day = new Date(dayStart);
                const key = `${day.getFullYear()}-${String(day.getMonth() + 1).padStart(2, "0")}-${String(day.getDate()).padStart(2, "0")}`;
                const share = decrease * ((overlapEnd - overlapStart) / span);
                buckets.set(key, (buckets.get(key) || 0) + share);
            }
            dayStart = dayEnd;
        }
    }

    return [...buckets.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([date, kg]) => ({ date, kg }));
}

function colorForSource(source) {
    return source === MANUAL_SOURCE ? "#d3973f" : source === LIVE_SOURCE ? "#159b83" : "#6884d6";
}

function renderStockChart(points, range) {
    if (stockChart) {
        stockChart.destroy();
        stockChart = null;
    }

    const grouped = new Map();
    for (const point of points) {
        const source = point.source || "unknown";
        if (!grouped.has(source)) grouped.set(source, []);
        grouped.get(source).push(point);
    }

    document.getElementById("stock-count").textContent = `${formatNumber(points.length, 0)} ${points.length === 1 ? "reading" : "readings"}`;
    document.getElementById("stock-empty").hidden = points.length > 0;
    if (!points.length) return;

    const datasets = [...grouped.entries()].map(([source, series]) => {
        const color = colorForSource(source);
        return {
            label: source === MANUAL_SOURCE ? "Manual history" : source === LIVE_SOURCE ? "ETA live" : source,
            data: series.map(point => ({ x: parseTimestamp(point), y: Number(point.value), note: point.note || "", source })),
            borderColor: color,
            backgroundColor: source === MANUAL_SOURCE ? "rgba(211, 151, 63, .10)" : "rgba(21, 155, 131, .09)",
            borderWidth: 2,
            pointRadius: series.length > 350 ? 0 : 2,
            pointHoverRadius: 4,
            fill: "start",
            tension: 0.12,
            spanGaps: false
        };
    });

    const min = range.from === null ? points.reduce((earliest, point) => Math.min(earliest, parseTimestamp(point)), Number.POSITIVE_INFINITY) : range.from;
    stockChart = new Chart(document.getElementById("stock-chart"), {
        type: "line",
        data: { datasets },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            parsing: false,
            plugins: {
                decimation: { enabled: true, algorithm: "lttb", samples: 1000, threshold: 1500 },
                legend: { display: grouped.size > 1, position: "bottom", align: "start", labels: { usePointStyle: true, pointStyle: "circle", boxWidth: 7, boxHeight: 7, padding: 17, color: "#788681", font: { family: "DM Sans", size: 12 } } },
                tooltip: {
                    backgroundColor: "#173632",
                    titleColor: "#d9eee8",
                    bodyColor: "#fff",
                    padding: 10,
                    usePointStyle: true,
                    callbacks: {
                        title: items => items.length ? formatDate(items[0].parsed.x, items[0].raw.source !== MANUAL_SOURCE) : "",
                        label: context => ` ${context.dataset.label}: ${formatKg(context.parsed.y)}`,
                        afterBody: items => items.length && items[0].raw.note ? [items[0].raw.note] : []
                    }
                }
            },
            scales: {
                x: {
                    type: "linear",
                    min,
                    max: range.to,
                    grid: { display: false },
                    border: { display: false },
                    ticks: { maxTicksLimit: 8, maxRotation: 0, color: "#93a09a", font: { family: "DM Sans", size: 11 }, callback: value => new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short" }).format(new Date(Number(value))) }
                },
                y: {
                    beginAtZero: false,
                    border: { display: false },
                    grid: { color: "#edf1ef", drawTicks: false },
                    ticks: { maxTicksLimit: 6, padding: 9, color: "#93a09a", font: { family: "DM Sans", size: 11 }, callback: value => `${formatNumber(Number(value), 0)} kg` }
                }
            }
        }
    });
}

function renderUsageChart(dailyUse) {
    if (usageChart) {
        usageChart.destroy();
        usageChart = null;
    }
    document.getElementById("usage-empty").hidden = dailyUse.length > 0;
    if (!dailyUse.length) return;

    usageChart = new Chart(document.getElementById("usage-chart"), {
        type: "bar",
        data: {
            labels: dailyUse.map(item => item.date),
            datasets: [{
                label: "Pellets used",
                data: dailyUse.map(item => item.kg),
                backgroundColor: "rgba(21, 155, 131, .72)",
                hoverBackgroundColor: "#159b83",
                borderRadius: 3,
                maxBarThickness: 22
            }]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            plugins: {
                legend: { display: false },
                tooltip: {
                    backgroundColor: "#173632",
                    titleColor: "#d9eee8",
                    bodyColor: "#fff",
                    padding: 9,
                    callbacks: {
                        title: items => items.length ? formatDate(new Date(`${items[0].label}T12:00:00`).getTime()) : "",
                        label: context => ` Estimated use: ${formatKg(context.parsed.y)}`
                    }
                }
            },
            scales: {
                x: { grid: { display: false }, border: { display: false }, ticks: { maxTicksLimit: 10, maxRotation: 0, color: "#93a09a", font: { family: "DM Sans", size: 11 }, callback: function (value) { const label = this.getLabelForValue(value); const date = new Date(`${label}T12:00:00`); return new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short" }).format(date); } } },
                y: { beginAtZero: true, border: { display: false }, grid: { color: "#edf1ef", drawTicks: false }, ticks: { maxTicksLimit: 5, padding: 9, color: "#93a09a", font: { family: "DM Sans", size: 11 }, callback: value => `${formatNumber(Number(value), 1)} kg` } }
            }
        }
    });
}

function groupPurchases(rows) {
    const groups = new Map();
    for (const row of rows) {
        const source = row.source || "unknown";
        if (!groups.has(source)) groups.set(source, { source, ts: row.ts, quantity: null, cost: null, note: row.note || "" });
        const group = groups.get(source);
        if (parseTimestamp(row) < parseTimestamp(group)) group.ts = row.ts;
        if (row.note && !group.note) group.note = row.note;
        if (row.type === PURCHASE_QUANTITY_TYPE) group.quantity = Number(row.value);
        if (row.type === PURCHASE_COST_TYPE) group.cost = Number(row.value);
    }
    return [...groups.values()].sort((a, b) => parseTimestamp(b) - parseTimestamp(a));
}

function weightedPrice(purchases, to) {
    let totalQuantity = 0;
    let totalCost = 0;
    for (const purchase of purchases) {
        if (parseTimestamp(purchase) > to || !(purchase.quantity > 0) || !(purchase.cost >= 0)) continue;
        totalQuantity += purchase.quantity;
        totalCost += purchase.cost;
    }
    return totalQuantity > 0 ? totalCost / totalQuantity : null;
}

function renderPurchases(purchases) {
    const body = document.getElementById("purchases-body");
    const count = document.getElementById("purchase-count");
    const rows = purchases.slice(0, 10);
    count.textContent = `${purchases.length} ${purchases.length === 1 ? "PURCHASE" : "PURCHASES"}`;
    body.replaceChildren();

    if (!rows.length) {
        const row = document.createElement("tr");
        row.className = "empty-row";
        const cell = document.createElement("td");
        cell.colSpan = 5;
        cell.textContent = "No purchases recorded yet";
        row.appendChild(cell);
        body.appendChild(row);
        return;
    }

    for (const purchase of rows) {
        const row = document.createElement("tr");
        const date = document.createElement("td");
        date.textContent = formatDate(purchase.ts, true);
        const quantity = document.createElement("td");
        quantity.textContent = purchase.quantity === null ? "—" : formatKg(purchase.quantity);
        const cost = document.createElement("td");
        cost.textContent = purchase.cost === null ? "—" : `€${formatNumber(purchase.cost, 2)}`;
        const price = document.createElement("td");
        price.textContent = purchase.quantity > 0 && purchase.cost !== null ? `€${formatNumber(purchase.cost / purchase.quantity, 3)}` : "—";
        const note = document.createElement("td");
        note.textContent = purchase.note || "";
        row.append(date, quantity, cost, price, note);
        body.appendChild(row);
    }
}

function renderStockEvents(points) {
    const body = document.getElementById("stock-events-body");
    const chronological = [...points].sort((a, b) => parseTimestamp(a) - parseTimestamp(b));
    const refillOrAdjustment = new WeakSet();
    for (let index = 1; index < chronological.length; index += 1) {
        if (Number(chronological[index].value) > Number(chronological[index - 1].value)) refillOrAdjustment.add(chronological[index]);
    }
    const rows = [...chronological].reverse().slice(0, 12);
    body.replaceChildren();
    if (!rows.length) {
        const row = document.createElement("tr");
        row.className = "empty-row";
        const cell = document.createElement("td");
        cell.colSpan = 3;
        cell.textContent = "No stock readings in this period";
        row.appendChild(cell);
        body.appendChild(row);
        return;
    }

    for (const point of rows) {
        const row = document.createElement("tr");
        const date = document.createElement("td");
        date.textContent = point.source === MANUAL_SOURCE ? formatDate(point.ts) : formatDate(point.ts, true);
        const stock = document.createElement("td");
        stock.textContent = formatKg(Number(point.value));
        const detail = document.createElement("td");
        const event = refillOrAdjustment.has(point) ? "Stock increased — refill or adjustment boundary" : "";
        detail.textContent = [point.source || "unknown", event, point.note || ""].filter(Boolean).join(" · ");
        row.append(date, stock, detail);
        body.appendChild(row);
    }
}

function updateSummary(points, baseline, range, purchases, forecastPoints, allHistoryPoints) {
    const history = normalizeStockPoints([...(baseline ? [baseline] : []), ...points]);
    const current = history.reduce((latest, point) => !latest || parseTimestamp(point) > parseTimestamp(latest) ? point : latest, null);
    const min = range.from === null ? (history.length ? history.reduce((earliest, point) => Math.min(earliest, parseTimestamp(point)), Number.POSITIVE_INFINITY) : null) : range.from;
    const used = consumptionInRange(history, min, range.to);
    const durationDays = min === null ? 0 : Math.max((range.to - min) / DAY_MS, 1);
    const lifetimeHistory = normalizeStockPoints(allHistoryPoints);
    const lifetimeStart = lifetimeHistory.length ? parseTimestamp(lifetimeHistory[0]) : null;
    const lifetimeUse = consumptionInRange(lifetimeHistory, lifetimeStart, range.to);
    const last30Start = range.to - 30 * DAY_MS;
    const last30Use = consumptionInRange(lifetimeHistory, last30Start, range.to);

    document.getElementById("stock-value").textContent = current ? formatKg(Number(current.value)) : "—";
    document.getElementById("stock-caption").textContent = current ? `${current.source || "unknown"} · ${formatDate(current.ts, current.source !== MANUAL_SOURCE)}` : "No stock readings available";
    document.getElementById("used-value").textContent = used.intervals ? formatKg(used.kilograms) : "—";
    document.getElementById("used-caption").textContent = used.intervals
        ? `${range.label} · avg ${formatNumber(used.kilograms / durationDays, 2)} kg/day from ${used.intervals} declining intervals`
        : "Need consecutive stock readings in this period";

    document.getElementById("total-value").textContent = lifetimeUse.intervals ? formatKg(lifetimeUse.kilograms) : "—";
    document.getElementById("total-caption").textContent = lifetimeUse.intervals && lifetimeStart !== null
        ? `Since ${formatDate(lifetimeStart)} · refill increases excluded`
        : "Need at least two recorded stock readings";

    document.getElementById("daily30-value").textContent = last30Use.intervals ? `${formatNumber(last30Use.kilograms / 30, 2)} kg/day` : "—";
    document.getElementById("daily30-caption").textContent = last30Use.intervals
        ? `${formatKg(last30Use.kilograms)} estimated over 30 days · ${last30Use.intervals} declining intervals`
        : "Need stock readings spanning the past 30 days";

    const recent = normalizeStockPoints(forecastPoints.filter(point => point.source === LIVE_SOURCE));
    const now = Date.now();
    const latestLive = recent[recent.length - 1];
    const forecastFrom = now - FORECAST_WINDOW_MS;
    const forecastUse = consumptionInRange(recent, forecastFrom, now);
    const firstInWindow = recent.find(point => parseTimestamp(point) >= forecastFrom) || recent[0];
    const observedMs = latestLive && firstInWindow ? parseTimestamp(latestLive) - Math.max(parseTimestamp(firstInWindow), forecastFrom) : 0;
    const fresh = latestLive && now - parseTimestamp(latestLive) <= FORECAST_MAX_AGE_MS;

    if (fresh && recent.length >= 2 && observedMs >= 2 * HOUR_MS) {
        const nextHourKg = forecastUse.kilograms / (observedMs / HOUR_MS);
        document.getElementById("forecast-value").textContent = formatKg(nextHourKg);
        document.getElementById("forecast-caption").textContent = `Based on ${formatNumber(observedMs / HOUR_MS, 1)}h of fresh ETA stock readings`;
    } else {
        document.getElementById("forecast-value").textContent = "—";
        document.getElementById("forecast-caption").textContent = "Need at least 2h of fresh ETA readings (latest within 20m)";
    }

    const averagePrice = weightedPrice(purchases, range.to);
    if (averagePrice === null || used.intervals === 0) {
        document.getElementById("cost-value").textContent = "—";
        document.getElementById("cost-caption").textContent = averagePrice === null ? "Record purchase quantities and prices to estimate cost" : "Need stock readings across this period";
    } else {
        document.getElementById("cost-value").textContent = `€${formatNumber(used.kilograms * averagePrice, 2)}`;
        document.getElementById("cost-caption").textContent = `${formatKg(used.kilograms)} × weighted purchase average €${formatNumber(averagePrice, 3)}/kg`;
    }
}

async function refreshPelletPage() {
    const request = ++activeRequest;
    const now = Date.now();
    const range = rangeForSelection(now);
    document.getElementById("pellet-error").hidden = true;
    document.getElementById("pellet-updated").textContent = "Loading…";

    try {
        const stockPromise = fetchMeasurements([STOCK_TYPE], range.from, range.to);
        const allHistoryPromise = range.from === null ? stockPromise : fetchMeasurements([STOCK_TYPE], null, range.to, 200000);
        const [stock, baseline, purchases, forecastRecent, forecastBaseline, allHistoryStock] = await Promise.all([
            stockPromise,
            range.from === null ? Promise.resolve(null) : fetchLatest(STOCK_TYPE, range.from),
            fetchMeasurements([PURCHASE_QUANTITY_TYPE, PURCHASE_COST_TYPE], null, range.to, 10000),
            fetchMeasurements([STOCK_TYPE], now - FORECAST_WINDOW_MS, now, 200000, LIVE_SOURCE),
            fetchLatest(STOCK_TYPE, now - FORECAST_WINDOW_MS, LIVE_SOURCE),
            allHistoryPromise
        ]);
        if (request !== activeRequest) return;

        const history = normalizeStockPoints([...(baseline ? [baseline] : []), ...stock]);
        const chartPoints = range.from === null ? stock : stock.filter(point => parseTimestamp(point) >= range.from);
        const lifetimeHistory = normalizeStockPoints(allHistoryStock);
        const daily30Use = estimatedDailyUse(lifetimeHistory, now - 30 * DAY_MS, now);
        const purchasesGrouped = groupPurchases(purchases);
        const forecastPoints = normalizeStockPoints([...(forecastBaseline ? [forecastBaseline] : []), ...forecastRecent]);

        renderStockChart(chartPoints, range);
        renderUsageChart(daily30Use);
        renderStockEvents([...(baseline ? [baseline] : []), ...chartPoints]);
        renderPurchases(purchasesGrouped);
        updateSummary(chartPoints, baseline, range, purchasesGrouped, forecastPoints, lifetimeHistory);

        document.getElementById("period-caption").textContent = range.label;
        document.getElementById("stock-chart-subtitle").textContent = `${formatNumber(chartPoints.length, 0)} manual and live stock observations`;
        document.getElementById("period-caption").setAttribute("aria-live", "polite");
        document.getElementById("pellet-range-description").textContent = range.from === null
            ? `From ${chartPoints.length ? formatDate(chartPoints.reduce((oldest, point) => parseTimestamp(point) < parseTimestamp(oldest) ? point : oldest).ts) : "no recorded date"} to now`
            : `${formatDate(range.from)} – ${formatDate(range.to)}`;
        document.getElementById("pellet-updated").textContent = `Updated ${new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" }).format(new Date())}`;
    } catch (error) {
        if (request !== activeRequest) return;
        document.getElementById("pellet-updated").textContent = "Update failed";
        const message = document.getElementById("pellet-error");
        message.textContent = error.message || "Could not load pellet data.";
        message.hidden = false;
    }
}

function setupRangeControls() {
    document.querySelectorAll(".pellet-controls .range-option").forEach(button => {
        button.addEventListener("click", () => {
            document.querySelectorAll(".pellet-controls .range-option").forEach(item => {
                const active = item === button;
                item.classList.toggle("active", active);
                item.setAttribute("aria-pressed", String(active));
            });
            selectedDays = button.dataset.days === "all" ? "all" : Number(button.dataset.days);
            refreshPelletPage();
        });
    });
}

function setupPurchaseForm() {
    const form = document.getElementById("purchase-form");
    const dateInput = document.getElementById("purchase-at");
    dateInput.value = localInputValue(Date.now());

    form.addEventListener("submit", async event => {
        event.preventDefault();
        if (!form.reportValidity()) return;

        const quantity = Number(document.getElementById("purchase-quantity").value);
        const totalPaid = Number(document.getElementById("purchase-cost").value);
        if (!(quantity > 0) || !(totalPaid > 0)) {
            document.getElementById("purchase-status").textContent = "Enter a quantity and a total amount greater than zero.";
            document.getElementById("purchase-status").classList.add("is-error");
            return;
        }

        const date = new Date(dateInput.value);
        if (!Number.isFinite(date.getTime())) return;
        const timestamp = date.toISOString();
        const purchaseID = globalThis.crypto?.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
        const source = `manual-purchase:${purchaseID}`;
        const note = document.getElementById("purchase-note").value.trim();
        const payload = [
            { ts: timestamp, type: PURCHASE_QUANTITY_TYPE, value: quantity, unit: "kg", source, note },
            { ts: timestamp, type: PURCHASE_COST_TYPE, value: totalPaid, unit: "EUR", source, note }
        ];

        const button = document.getElementById("save-purchase");
        const status = document.getElementById("purchase-status");
        button.disabled = true;
        status.classList.remove("is-error");
        status.textContent = "Saving purchase measurements…";

        try {
            const response = await fetch(SOURCE_ENDPOINT, {
                method: "POST",
                headers: { "Content-Type": "application/json", Accept: "application/json" },
                body: JSON.stringify(payload)
            });
            if (!response.ok) {
                let error = "Could not save purchase.";
                try {
                    const body = await response.json();
                    if (body.error) error = body.error;
                } catch (_) {
                    // Use the fallback message for an empty or non-JSON response.
                }
                throw new Error(error);
            }
            form.reset();
            dateInput.value = localInputValue(Date.now());
            status.textContent = "Purchase saved in measurements.";
            await refreshPelletPage();
        } catch (error) {
            status.textContent = error.message || "Could not save purchase.";
            status.classList.add("is-error");
        } finally {
            button.disabled = false;
        }
    });
}

document.addEventListener("DOMContentLoaded", () => {
    setupRangeControls();
    setupPurchaseForm();
    document.getElementById("pellet-refresh").addEventListener("click", refreshPelletPage);
    refreshPelletPage();
    refreshTimer = setInterval(refreshPelletPage, 60000);
    document.addEventListener("visibilitychange", () => {
        if (document.visibilityState === "visible") refreshPelletPage();
    });
});
