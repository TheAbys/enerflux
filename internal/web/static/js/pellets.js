let stockChart = null;
let usageChart = null;
let selectedDays = 30;
let activeRequest = 0;

function formatNumber(value, digits = 2) {
    const number = Number(value);
    return Number.isFinite(number) ? new Intl.NumberFormat(undefined, { maximumFractionDigits: digits }).format(number) : "—";
}

function valueWithUnit(value, unit = "") {
    return value === null || value === undefined ? "—" : `${formatNumber(value)}${unit ? ` ${unit}` : ""}`;
}

function formatDate(value, includeTime = false) {
    return new Intl.DateTimeFormat(undefined, includeTime
        ? { day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit" }
        : { day: "2-digit", month: "short", year: "numeric" }).format(new Date(value));
}

function localInputValue(timestamp) {
    const date = new Date(timestamp);
    const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000);
    return local.toISOString().slice(0, 16);
}

function selectedWindow() {
    const to = Date.now();
    const from = selectedDays === "all" ? Date.parse("2000-01-01T00:00:00Z") : to - Number(selectedDays) * 24 * 60 * 60 * 1000;
    return { from, to };
}

function sourceColor(source, index) {
    if (source === "manual") return "#d3973f";
    if (source === "eta") return "#159b83";
    return ["#637fd0", "#8a6cda", "#d96e72", "#2d9cb3"][index % 4];
}

function setSummary(data) {
    const stock = data.currentStock;
    document.getElementById("stock-value").textContent = stock ? valueWithUnit(stock.value, stock.unit) : "—";
    document.getElementById("stock-caption").textContent = stock ? `${stock.source || "Unknown source"} · ${formatDate(stock.ts, stock.source !== "manual")}` : "No stock readings stored";

    const consumed = data.consumed;
    document.getElementById("used-value").textContent = valueWithUnit(consumed.value, consumed.unit);
    document.getElementById("used-caption").textContent = consumed.value === null
        ? consumed.status
        : `${formatNumber(consumed.intervals, 0)} declining intervals · ${formatNumber(consumed.coveredHours, 1)}h covered · estimated`;

    document.getElementById("total-value").textContent = valueWithUnit(data.lifetimeConsumedKg, "kg");
    document.getElementById("total-caption").textContent = data.lifetimeConsumedKg === null ? "Need at least two kilogram stock readings" : "Lifetime decreases; refill increases excluded";
    document.getElementById("daily30-value").textContent = valueWithUnit(data.averageDaily30DaysKg, "kg/day");
    document.getElementById("daily30-caption").textContent = data.consumedLast30DaysKg === null
        ? "Need readings covering part of the past 30 days"
        : `${valueWithUnit(data.consumedLast30DaysKg, "kg")} estimated · ${formatNumber(data.last30DaysIntervals, 0)} intervals across ${formatNumber(data.last30DaysCoveredHours / 24, 1)} covered days`;

    document.getElementById("forecast-value").textContent = valueWithUnit(data.forecastNextHourKg, "kg");
    document.getElementById("forecast-caption").textContent = data.forecastStatus;
    document.getElementById("cost-value").textContent = valueWithUnit(consumed.costEur, "EUR");
    document.getElementById("cost-caption").textContent = data.averagePurchasePriceEurPerKg === null
        ? "Record purchase quantity and cost to estimate fuel cost"
        : `${valueWithUnit(data.averagePurchasePriceEurPerKg, "EUR/kg")} weighted purchase average · ${consumed.costEur === null ? "no usage estimate" : "applied to selected-period use"}`;
}

function renderStockChart(points, from, to) {
    if (stockChart) stockChart.destroy();
    const units = [...new Set(points.map(point => point.unit || "unknown"))];
    document.getElementById("stock-unit").textContent = units.length === 1 ? units[0] : units.length ? "mixed units" : "—";
    document.getElementById("stock-count").textContent = `${formatNumber(points.length, 0)} ${points.length === 1 ? "reading" : "readings"}`;
    document.getElementById("stock-empty").hidden = points.length > 0;
    if (!points.length) return;

    const groups = new Map();
    for (const point of points) {
        const key = `${point.source || "unknown"} · ${point.unit || "unit"}`;
        if (!groups.has(key)) groups.set(key, []);
        groups.get(key).push(point);
    }
    const datasets = [...groups.entries()].map(([label, values], index) => {
        const color = sourceColor(values[0].source, index);
        return {
            label,
            data: values.map(point => ({ x: new Date(point.ts).getTime(), y: Number(point.value), note: point.note || "", eventType: point.eventType || "" })),
            borderColor: color,
            backgroundColor: `${color}18`,
            borderWidth: 2,
            pointRadius: values.length > 350 ? 0 : 2,
            pointHoverRadius: 4,
            fill: "start",
            tension: .12,
            spanGaps: false
        };
    });
    stockChart = new Chart(document.getElementById("stock-chart"), {
        type: "line",
        data: { datasets },
        options: {
            responsive: true, maintainAspectRatio: false, animation: false, parsing: false,
            plugins: {
                decimation: { enabled: true, algorithm: "lttb", samples: 1000, threshold: 1500 },
                legend: { display: groups.size > 1, position: "bottom", align: "start" },
                tooltip: { callbacks: { title: items => items.length ? formatDate(items[0].parsed.x, true) : "", label: context => ` ${context.dataset.label}: ${valueWithUnit(context.parsed.y, context.dataset.label.split(" · ").at(-1))}`, afterBody: items => items.length ? [items[0].raw.note, items[0].raw.eventType === "stock_increase" ? "Refill or adjustment boundary" : ""].filter(Boolean) : [] } }
            },
            scales: { x: { type: "linear", min: from, max: to, grid: { display: false }, ticks: { maxTicksLimit: 8, callback: value => new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short" }).format(new Date(Number(value))) } }, y: { beginAtZero: false, ticks: { maxTicksLimit: 6, callback: value => `${formatNumber(Number(value), 0)}` } } }
        }
    });
}

function renderUsageChart(points) {
    if (usageChart) usageChart.destroy();
    document.getElementById("usage-empty").hidden = points.length > 0;
    if (!points.length) return;
    usageChart = new Chart(document.getElementById("usage-chart"), {
        type: "bar",
        data: { labels: points.map(point => point.date), datasets: [{ label: "Estimated pellets used", data: points.map(point => point.value), backgroundColor: "rgba(21, 155, 131, .72)", hoverBackgroundColor: "#159b83", borderRadius: 3, maxBarThickness: 22 }] },
        options: { responsive: true, maintainAspectRatio: false, animation: false, plugins: { legend: { display: false }, tooltip: { callbacks: { title: items => items.length ? items[0].label : "", label: context => ` Estimated use: ${valueWithUnit(context.parsed.y, "kg")}` } } }, scales: { x: { grid: { display: false }, ticks: { maxTicksLimit: 10, callback: function (value) { const date = new Date(`${this.getLabelForValue(value)}T12:00:00`); return new Intl.DateTimeFormat(undefined, { day: "2-digit", month: "short" }).format(date); } } }, y: { beginAtZero: true, ticks: { callback: value => `${formatNumber(Number(value), 1)} kg` } } } }
    });
}

function renderStockEvents(points) {
    const body = document.getElementById("stock-events-body");
    body.replaceChildren();
    const rows = [...points].sort((a, b) => new Date(b.ts) - new Date(a.ts)).slice(0, 12);
    if (!rows.length) {
        const row = document.createElement("tr"); row.className = "empty-row";
        const cell = document.createElement("td"); cell.colSpan = 3; cell.textContent = "No stock readings in this period";
        row.appendChild(cell); body.appendChild(row); return;
    }
    for (const point of rows) {
        const row = document.createElement("tr");
        const date = document.createElement("td"); date.textContent = formatDate(point.ts, point.source !== "manual");
        const stock = document.createElement("td"); stock.textContent = valueWithUnit(point.value, point.unit);
        const detail = document.createElement("td");
        detail.textContent = [point.source || "unknown", point.eventType === "stock_increase" ? "Refill or adjustment" : "", point.note || ""].filter(Boolean).join(" · ");
        row.append(date, stock, detail); body.appendChild(row);
    }
}

function renderPurchases(purchases) {
    const body = document.getElementById("purchases-body");
    body.replaceChildren();
    document.getElementById("purchase-count").textContent = `${purchases.length} ${purchases.length === 1 ? "PURCHASE" : "PURCHASES"}`;
    if (!purchases.length) {
        const row = document.createElement("tr"); row.className = "empty-row";
        const cell = document.createElement("td"); cell.colSpan = 5; cell.textContent = "No purchases recorded yet";
        row.appendChild(cell); body.appendChild(row); return;
    }
    for (const purchase of purchases.slice(0, 50)) {
        const row = document.createElement("tr");
        const date = document.createElement("td"); date.textContent = formatDate(purchase.ts, true);
        const quantity = document.createElement("td"); quantity.textContent = valueWithUnit(purchase.quantityKg, "kg");
        const cost = document.createElement("td"); cost.textContent = valueWithUnit(purchase.totalCostEur, "EUR");
        const unitPrice = document.createElement("td"); unitPrice.textContent = valueWithUnit(purchase.costPerKgEur, "EUR/kg");
        const note = document.createElement("td"); note.textContent = purchase.note || "";
        row.append(date, quantity, cost, unitPrice, note); body.appendChild(row);
    }
}

async function refreshPelletPage() {
    const request = ++activeRequest;
    const { from, to } = selectedWindow();
    const query = new URLSearchParams({ from: new Date(from).toISOString(), to: new Date(to).toISOString() });
    document.getElementById("pellet-error").hidden = true;
    document.getElementById("pellet-updated").textContent = "Updating…";
    try {
        const response = await fetch(`/api/v1/analytics/heating?${query}`, { headers: { Accept: "application/json" } });
        if (!response.ok) {
            const payload = await response.json().catch(() => ({}));
            throw new Error(payload.error || "Could not load heating insights.");
        }
        const data = await response.json();
        if (request !== activeRequest) return;
        setSummary(data);
        renderStockChart(data.stockReadings || [], from, to);
        renderUsageChart(data.daily || []);
        renderStockEvents(data.stockReadings || []);
        renderPurchases(data.purchases || []);
        document.getElementById("period-caption").textContent = selectedDays === "all" ? "All recorded history" : `Last ${selectedDays} days`;
        document.getElementById("stock-chart-subtitle").textContent = `${formatNumber((data.stockReadings || []).length, 0)} stored stock observations · kg when configured`;
        document.getElementById("pellet-range-description").textContent = selectedDays === "all" ? "All recorded history" : `Last ${selectedDays} days · ${formatDate(from)} – now`;
        document.getElementById("pellet-updated").textContent = `Updated ${new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" }).format(new Date())}`;
    } catch (error) {
        if (request !== activeRequest) return;
        document.getElementById("pellet-updated").textContent = "Update failed";
        const message = document.getElementById("pellet-error");
        message.textContent = error.message || "Could not load heating insights.";
        message.hidden = false;
    }
}

function setupRangeControls() {
    document.querySelectorAll(".pellet-controls .range-option").forEach(button => button.addEventListener("click", () => {
        selectedDays = button.dataset.days === "all" ? "all" : Number(button.dataset.days);
        document.querySelectorAll(".pellet-controls .range-option").forEach(item => {
            const active = item === button;
            item.classList.toggle("active", active);
            item.setAttribute("aria-pressed", String(active));
        });
        refreshPelletPage();
    }));
}

async function submitMeasurement(payload, status, successMessage) {
    const response = await fetch("/api/v1/measurements", { method: "POST", headers: { "Content-Type": "application/json", Accept: "application/json" }, body: JSON.stringify(payload) });
    if (!response.ok) {
        const body = await response.json().catch(() => ({}));
        throw new Error(body.error || "Could not save this heating record.");
    }
    status.dataset.state = "success";
    status.textContent = successMessage;
    status.hidden = false;
}

function setupStockForm() {
    const form = document.getElementById("stock-entry-form");
    const timeInput = document.getElementById("stock-entry-time");
    const status = document.getElementById("stock-entry-status");
    timeInput.value = localInputValue(Date.now());
    timeInput.max = localInputValue(Date.now());
    form.addEventListener("submit", async event => {
        event.preventDefault();
        if (!form.reportValidity()) return;
        const button = form.querySelector("button[type='submit']");
        const value = Number(document.getElementById("stock-entry-value").value);
        button.disabled = true;
        status.hidden = true;
        try {
            await submitMeasurement([{ ts: new Date(timeInput.value).toISOString(), type: "heating.pellet.stock", value, unit: "kg", source: "manual", note: document.getElementById("stock-entry-note").value.trim() }], status, "Stock reading saved in measurements.");
            form.reset();
            timeInput.value = localInputValue(Date.now());
            await refreshPelletPage();
        } catch (error) {
            status.dataset.state = "error"; status.textContent = error.message || "Could not save stock reading."; status.hidden = false;
        } finally { button.disabled = false; }
    });
}

function setupPurchaseForm() {
    const form = document.getElementById("purchase-form");
    const dateInput = document.getElementById("purchase-at");
    const status = document.getElementById("purchase-status");
    dateInput.value = localInputValue(Date.now());
    form.addEventListener("submit", async event => {
        event.preventDefault();
        if (!form.reportValidity()) return;
        const quantity = Number(document.getElementById("purchase-quantity").value);
        const totalPaid = Number(document.getElementById("purchase-cost").value);
        if (!(quantity > 0) || !(totalPaid > 0)) return;
        const button = document.getElementById("save-purchase");
        button.disabled = true;
        status.textContent = "Saving purchase measurements…";
        try {
            const id = globalThis.crypto?.randomUUID ? crypto.randomUUID() : `${Date.now()}-${Math.random().toString(16).slice(2)}`;
            const source = `manual-purchase:${id}`;
            const timestamp = new Date(dateInput.value).toISOString();
            const note = document.getElementById("purchase-note").value.trim();
            await submitMeasurement([
                { ts: timestamp, type: "heating.pellet.purchase.quantity", value: quantity, unit: "kg", source, note },
                { ts: timestamp, type: "heating.pellet.purchase.cost", value: totalPaid, unit: "EUR", source, note }
            ], status, "Purchase saved in measurements.");
            form.reset();
            dateInput.value = localInputValue(Date.now());
            await refreshPelletPage();
        } catch (error) {
            status.dataset.state = "error"; status.textContent = error.message || "Could not save purchase.";
        } finally { button.disabled = false; }
    });
}

document.addEventListener("DOMContentLoaded", () => {
    setupRangeControls();
    setupStockForm();
    setupPurchaseForm();
    document.getElementById("pellet-refresh").addEventListener("click", refreshPelletPage);
    refreshPelletPage();
    setInterval(refreshPelletPage, 60000);
    document.addEventListener("visibilitychange", () => { if (document.visibilityState === "visible") refreshPelletPage(); });
});
