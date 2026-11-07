import { apiErrorNotice, liveEnumLabel, vaLiveAPIPath } from "./format.mjs";

const dialog = document.getElementById("live-schedule-modal");
const form = document.getElementById("live-schedule-form");
const titleEl = document.getElementById("live-schedule-modal-title");
const statusEl = document.getElementById("live-schedule-modal-status");
const addBtn = document.getElementById("live-schedule-add");
const aircraftField = document.getElementById("live-schedule-field-aircraft");
const aircraftSelect = document.getElementById("live-schedule-aircraft");

const fieldIds = {
  callsign: "live-schedule-callsign",
  originIcao: "live-schedule-origin",
  destinationIcao: "live-schedule-destination",
  status: "live-schedule-status",
  flightType: "live-schedule-flight-type",
  isTimeless: "live-schedule-timeless",
  scheduledDepartureUtc: "live-schedule-departure",
  scheduledArrivalUtc: "live-schedule-arrival",
  briefing: "live-schedule-briefing",
  debriefing: "live-schedule-debriefing",
};

let mode = "create";
let scheduleId = "";
let initialSnapshot = null;
let getOrganizationId = () => "";
let getCatalog = () => [];
let onSaved = async () => {};
let enumOptions = { status: [], flightType: [] };
let enumsLoaded = false;

function showModalStatus(message, isError = false) {
  if (!statusEl) {
    return;
  }
  statusEl.hidden = message === "";
  statusEl.textContent = message;
  statusEl.classList.toggle("panel-message--error", isError);
}

function readValue(key) {
  const el = document.getElementById(fieldIds[key]);
  if (!el) {
    return undefined;
  }
  if (key === "isTimeless") {
    return el.checked;
  }
  if (key === "status" || key === "flightType") {
    const raw = el.value.trim();
    return raw === "" ? undefined : raw;
  }
  if (key === "scheduledDepartureUtc" || key === "scheduledArrivalUtc") {
    const raw = el.value.trim();
    if (raw === "") {
      return undefined;
    }
    const date = new Date(raw);
    return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
  }
  const raw = el.value;
  if (typeof raw !== "string") {
    return undefined;
  }
  const trimmed = raw.trim();
  if (trimmed === "" && key !== "briefing" && key !== "debriefing") {
    return undefined;
  }
  return raw;
}

function writeValue(key, value) {
  const el = document.getElementById(fieldIds[key]);
  if (!el) {
    return;
  }
  if (key === "isTimeless") {
    el.checked = Boolean(value);
    return;
  }
  if (key === "scheduledDepartureUtc" || key === "scheduledArrivalUtc") {
    el.value = isoToDatetimeLocal(value);
    return;
  }
  if (key === "status" || key === "flightType") {
    el.value = value === undefined || value === null || value === "" ? "" : String(value);
    return;
  }
  el.value = value ?? "";
}

function isoToDatetimeLocal(iso) {
  if (!iso) {
    return "";
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  const pad = (n) => String(n).padStart(2, "0");
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}T${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}`;
}

function buildUpsertBody(fromSnapshot = null) {
  const keys = [
    "callsign",
    "originIcao",
    "destinationIcao",
    "status",
    "flightType",
    "isTimeless",
    "scheduledDepartureUtc",
    "scheduledArrivalUtc",
    "briefing",
    "debriefing",
  ];
  const body = {};
  for (const key of keys) {
    const value = readValue(key);
    if (value === undefined) {
      continue;
    }
    if (fromSnapshot && fromSnapshot[key] === value) {
      continue;
    }
    body[key] = value;
  }
  return body;
}

function snapshotForm() {
  const snap = {};
  for (const key of Object.keys(fieldIds)) {
    const value = readValue(key);
    if (value !== undefined) {
      snap[key] = value;
    }
  }
  snap.aircraftId = aircraftSelect?.value ?? "";
  return snap;
}

function fillEnumSelect(elementId, names) {
  const el = document.getElementById(elementId);
  if (!el) {
    return;
  }
  const selected = el.value;
  el.replaceChildren();
  const empty = document.createElement("option");
  empty.value = "";
  empty.textContent = "Not set";
  empty.selected = selected === "";
  el.appendChild(empty);
  for (const name of names) {
    const option = document.createElement("option");
    option.value = name;
    option.textContent = liveEnumLabel(name);
    option.selected = name === selected;
    el.appendChild(option);
  }
}

function populateEnumSelects() {
  fillEnumSelect(fieldIds.status, enumOptions.status);
  fillEnumSelect(fieldIds.flightType, enumOptions.flightType);
}

async function loadScheduleEnums() {
  if (enumsLoaded) {
    return;
  }
  const response = await fetch("/api/v1/va/live/schedule-enums", { credentials: "same-origin" });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(apiErrorNotice(payload, response.status));
  }
  enumOptions = {
    status: Array.isArray(payload?.data?.status) ? payload.data.status : [],
    flightType: Array.isArray(payload?.data?.flightType) ? payload.data.flightType : [],
  };
  enumsLoaded = true;
  populateEnumSelects();
}

function populateAircraftOptions() {
  if (!aircraftSelect) {
    return;
  }
  const selected = aircraftSelect.value;
  aircraftSelect.replaceChildren();
  const placeholder = document.createElement("option");
  placeholder.value = "";
  placeholder.textContent = "Select an aircraft";
  aircraftSelect.appendChild(placeholder);
  for (const row of getCatalog()) {
    const option = document.createElement("option");
    option.value = row.id;
    const reg = row.registration ?? "—";
    const name = row.aircraftName ?? "";
    option.textContent = name ? `${reg} · ${name}` : reg;
    if (row.id === selected) {
      option.selected = true;
    }
    aircraftSelect.appendChild(option);
  }
}

function resetForm() {
  for (const key of Object.keys(fieldIds)) {
    writeValue(key, key === "isTimeless" ? false : "");
  }
  if (aircraftSelect) {
    aircraftSelect.value = "";
  }
  initialSnapshot = null;
  showModalStatus("");
}

export function initScheduleEditor(options) {
  if (!dialog || !form) {
    return;
  }
  getOrganizationId = options.getOrganizationId ?? getOrganizationId;
  getCatalog = options.getCatalog ?? getCatalog;
  onSaved = options.onSaved ?? onSaved;

  loadScheduleEnums().catch((error) => {
    console.error("load schedule enums failed", error);
  });

  const close = () => {
    dialog.close();
    resetForm();
  };

  document.getElementById("live-schedule-modal-close")?.addEventListener("click", close);
  document.getElementById("live-schedule-cancel")?.addEventListener("click", close);
  dialog.addEventListener("cancel", (event) => {
    event.preventDefault();
    close();
  });

  addBtn?.addEventListener("click", async () => {
    try {
      await loadScheduleEnums();
    } catch (error) {
      showModalStatus(String(error.message ?? error), true);
      return;
    }
    mode = "create";
    scheduleId = "";
    resetForm();
    populateEnumSelects();
    populateAircraftOptions();
    if (aircraftField) {
      aircraftField.hidden = false;
    }
    if (titleEl) {
      titleEl.textContent = "Add schedule";
    }
    showModalStatus("");
    dialog.showModal();
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const orgId = getOrganizationId();
    if (!orgId) {
      showModalStatus("Select an organisation first.", true);
      return;
    }
    const upsertBody = mode === "edit" ? buildUpsertBody(initialSnapshot) : buildUpsertBody();
    if (mode === "create" && Object.keys(upsertBody).length === 0) {
      showModalStatus("Enter at least one field.", true);
      return;
    }
    if (mode === "edit" && Object.keys(upsertBody).length === 0) {
      showModalStatus("No changes to save.", true);
      return;
    }
    const saveBtn = document.getElementById("live-schedule-save");
    saveBtn?.setAttribute("disabled", "true");
    showModalStatus("Saving…");
    try {
      if (mode === "create") {
        const acId = aircraftSelect?.value?.trim() ?? "";
        if (!acId) {
          throw new Error("Select an aircraft.");
        }
        const response = await fetch(vaLiveAPIPath(orgId, `/aircraft/${encodeURIComponent(acId)}/schedules`), {
          method: "POST",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(upsertBody),
        });
        if (!response.ok) {
          const payload = await response.json().catch(() => null);
          throw new Error(apiErrorNotice(payload, response.status));
        }
      } else {
        const response = await fetch(vaLiveAPIPath(orgId, `/schedules/${encodeURIComponent(scheduleId)}`), {
          method: "PUT",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(upsertBody),
        });
        if (!response.ok) {
          const payload = await response.json().catch(() => null);
          throw new Error(apiErrorNotice(payload, response.status));
        }
      }
      showModalStatus("Saved.");
      await onSaved();
      close();
    } catch (error) {
      showModalStatus(String(error.message ?? error), true);
    } finally {
      saveBtn?.removeAttribute("disabled");
    }
  });
}

export async function openScheduleEditor(schedule) {
  if (!dialog || !schedule?.id) {
    return;
  }
  try {
    await loadScheduleEnums();
  } catch (error) {
    const status = document.getElementById("live-org-status");
    if (status) {
      status.hidden = false;
      status.textContent = String(error.message ?? error);
      status.classList.add("panel-message--error");
    }
    return;
  }
  mode = "edit";
  scheduleId = schedule.id;
  resetForm();
  populateEnumSelects();
  populateAircraftOptions();
  if (aircraftField) {
    aircraftField.hidden = true;
  }
  if (aircraftSelect) {
    aircraftSelect.value = schedule.aircraftId ?? "";
  }
  writeValue("callsign", schedule.callsign);
  writeValue("originIcao", schedule.originIcao);
  writeValue("destinationIcao", schedule.destinationIcao);
  writeValue("status", schedule.status);
  writeValue("flightType", schedule.flightType);
  writeValue("isTimeless", schedule.isTimeless);
  writeValue("scheduledDepartureUtc", schedule.scheduledDepartureUtc);
  writeValue("scheduledArrivalUtc", schedule.scheduledArrivalUtc);
  writeValue("briefing", schedule.briefing);
  writeValue("debriefing", schedule.debriefing);
  initialSnapshot = snapshotForm();
  if (titleEl) {
    titleEl.textContent = "Edit schedule";
  }
  showModalStatus("");
  dialog.showModal();
}

export function setScheduleAddEnabled(enabled) {
  if (!addBtn) {
    return;
  }
  addBtn.disabled = !enabled;
}
