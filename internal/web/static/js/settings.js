const SETTINGS_ENDPOINT = "/api/v1/configuration/sources";

const SETTINGS_FIELDS = {
    solarLogURL: "solarlog-url",
    solarLogPollInterval: "solarlog-interval",
    etaPelletstockURL: "pellet-url",
    etaPelletstockPollInterval: "pellet-interval",
    etaOutsidetempURL: "outside-url",
    etaOutsidetempPollInterval: "outside-interval"
};

function setMessage(id, message) {
    const element = document.getElementById(id);
    element.textContent = message;
    element.hidden = !message;
}

async function readError(response, fallback) {
    try {
        const payload = await response.json();
        return payload.error || fallback;
    } catch (_) {
        return fallback;
    }
}

async function loadSettings() {
    setMessage("settings-error", "");
    setMessage("settings-status", "Loading source configuration…");

    try {
        const response = await fetch(SETTINGS_ENDPOINT, { headers: { Accept: "application/json" } });
        if (!response.ok) throw new Error(await readError(response, "Could not load source configuration."));
        const settings = await response.json();
        for (const [name, id] of Object.entries(SETTINGS_FIELDS)) {
            document.getElementById(id).value = settings[name] || "";
        }
        setMessage("settings-status", "Current source configuration loaded.");
    } catch (error) {
        setMessage("settings-status", "");
        setMessage("settings-error", error.message || "Could not load source configuration.");
    }
}

async function saveSettings(event) {
    event.preventDefault();
    const form = event.currentTarget;
    if (!form.reportValidity()) return;

    const button = document.getElementById("save-settings");
    const originalContent = button.innerHTML;
    const settings = {};
    for (const [name, id] of Object.entries(SETTINGS_FIELDS)) {
        settings[name] = document.getElementById(id).value.trim();
    }

    button.disabled = true;
    button.textContent = "Saving…";
    setMessage("settings-error", "");
    setMessage("settings-status", "Saving configuration…");

    try {
        const response = await fetch(SETTINGS_ENDPOINT, {
            method: "PUT",
            headers: { "Content-Type": "application/json", Accept: "application/json" },
            body: JSON.stringify(settings)
        });
        if (!response.ok) throw new Error(await readError(response, "Could not save source configuration."));
        setMessage("settings-status", "Configuration saved in PostgreSQL. The agent will reload it within 5 seconds; changed endpoints are polled immediately.");
    } catch (error) {
        setMessage("settings-status", "");
        setMessage("settings-error", error.message || "Could not save source configuration.");
    } finally {
        button.disabled = false;
        button.innerHTML = originalContent;
    }
}

document.addEventListener("DOMContentLoaded", () => {
    document.getElementById("source-settings-form").addEventListener("submit", saveSettings);
    loadSettings();
});
