import { apiErrorNotice, vaLiveAPIPath } from "./format.mjs";

const dialog = document.getElementById("live-schedule-flightplan-modal");
const form = document.getElementById("live-schedule-flightplan-form");
const textArea = document.getElementById("live-schedule-flightplan-text");
const statusEl = document.getElementById("live-schedule-flightplan-modal-status");

let scheduleId = "";
let initialFlightPlan = "";
let getOrganizationId = () => "";
let onSaved = async () => {};

function showStatus(message, isError = false) {
  if (!statusEl) {
    return;
  }
  statusEl.hidden = message === "";
  statusEl.textContent = message;
  statusEl.classList.toggle("panel-message--error", isError);
}

function closeDialog() {
  if (!dialog) {
    return;
  }
  dialog.close();
  scheduleId = "";
  initialFlightPlan = "";
  if (textArea) {
    textArea.value = "";
  }
  showStatus("");
}

export function initFlightPlanEditor(options) {
  if (!dialog || !form) {
    return;
  }
  getOrganizationId = options.getOrganizationId ?? getOrganizationId;
  onSaved = options.onSaved ?? onSaved;

  document.getElementById("live-schedule-flightplan-modal-close")?.addEventListener("click", closeDialog);
  document.getElementById("live-schedule-flightplan-cancel")?.addEventListener("click", closeDialog);
  dialog.addEventListener("cancel", (event) => {
    event.preventDefault();
    closeDialog();
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    const orgId = getOrganizationId();
    if (!orgId || !scheduleId) {
      showStatus("Missing organisation or schedule.", true);
      return;
    }
    const flightPlan = textArea?.value ?? "";
    if (flightPlan === initialFlightPlan) {
      showStatus("No changes to save.", true);
      return;
    }
    const saveBtn = document.getElementById("live-schedule-flightplan-save");
    saveBtn?.setAttribute("disabled", "true");
    showStatus("Saving…");
    try {
      const response = await fetch(
        vaLiveAPIPath(orgId, `/schedules/${encodeURIComponent(scheduleId)}/flightplan`),
        {
          method: "PUT",
          credentials: "same-origin",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ flightPlan: flightPlan.trim() === "" ? null : flightPlan }),
        },
      );
      if (!response.ok) {
        const payload = await response.json().catch(() => null);
        throw new Error(apiErrorNotice(payload, response.status));
      }
      showStatus("Saved.");
      await onSaved();
      closeDialog();
    } catch (error) {
      showStatus(String(error.message ?? error), true);
    } finally {
      saveBtn?.removeAttribute("disabled");
    }
  });
}

export function openFlightPlanEditor(schedule) {
  if (!dialog || !schedule?.id) {
    return;
  }
  scheduleId = schedule.id;
  initialFlightPlan = schedule.flightPlan ?? "";
  if (textArea) {
    textArea.value = initialFlightPlan;
  }
  showStatus("");
  dialog.showModal();
}
