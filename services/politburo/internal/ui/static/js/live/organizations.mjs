import {
  aircraftSlotBadge,
  aircraftStatusBadge,
  aircraftVisibilityBadge,
  apiErrorNotice,
  countLabel,
  displayOrDash,
  formatUtcDateTime,
  vaLiveAPIPath,
} from "./format.mjs";

const select = document.getElementById("live-org-select");
const status = document.getElementById("live-org-status");
const fleetTarget = document.getElementById("live-aircraft-fleet");
const rowMenu = document.getElementById("live-aircraft-row-menu");
const menuPad = 8;

function pageParams() {
  return new URLSearchParams(window.location.search);
}

function announceOrgChange(organizationId, { clearAircraftFilter = false } = {}) {
  document.dispatchEvent(
    new CustomEvent("live-org-changed", {
      detail: { organizationId, clearAircraftFilter },
    }),
  );
}

let activeRow = null;
let copyResetTimer = null;

function roleLabel(role) {
  if (role === 1) {
    return "Admin";
  }
  if (role === 2) {
    return "Owner";
  }
  return String(role);
}

function showStatus(message, isError = false) {
  if (!status) {
    return;
  }
  status.hidden = false;
  status.textContent = message;
  status.classList.toggle("panel-message--error", isError);
}

function clearOptions() {
  select?.replaceChildren();
}

function addOption(value, label, selected = false) {
  if (!select) {
    return;
  }
  const option = document.createElement("option");
  option.value = value;
  option.textContent = label;
  option.selected = selected;
  select.appendChild(option);
}

function closeRowMenu() {
  if (!rowMenu) {
    return;
  }
  rowMenu.hidden = true;
  rowMenu.dataset.aircraftId = "";
  if (activeRow) {
    activeRow.classList.remove("data-table__row--active");
    activeRow = null;
  }
}

function clampMenuPosition(clientX, clientY) {
  if (!rowMenu) {
    return { x: clientX, y: clientY };
  }
  rowMenu.hidden = false;
  const rect = rowMenu.getBoundingClientRect();
  let x = clientX;
  let y = clientY;
  if (x + rect.width > window.innerWidth - menuPad) {
    x = window.innerWidth - rect.width - menuPad;
  }
  if (y + rect.height > window.innerHeight - menuPad) {
    y = window.innerHeight - rect.height - menuPad;
  }
  return {
    x: Math.max(menuPad, x),
    y: Math.max(menuPad, y),
  };
}

function openRowMenu(event, row) {
  if (!rowMenu || !row) {
    return;
  }
  event.preventDefault();
  event.stopPropagation();

  fleetTarget?.querySelectorAll(".data-table__row--active").forEach((el) => {
    el.classList.remove("data-table__row--active");
  });
  row.classList.add("data-table__row--active");
  activeRow = row;

  rowMenu.dataset.aircraftId = row.dataset.aircraftId ?? "";
  rowMenu.hidden = false;
  rowMenu.style.left = `${event.clientX}px`;
  rowMenu.style.top = `${event.clientY}px`;

  const copyBtn = rowMenu.querySelector('[data-action="copy-id"]');
  if (copyBtn) {
    copyBtn.textContent = "Copy Aircraft ID";
    copyBtn.classList.remove("row-menu__item--copied", "row-menu__item--failed");
  }

  requestAnimationFrame(() => {
    const { x, y } = clampMenuPosition(event.clientX, event.clientY);
    rowMenu.style.left = `${x}px`;
    rowMenu.style.top = `${y}px`;
  });
}

async function copyAircraftId() {
  const id = rowMenu?.dataset.aircraftId ?? "";
  const copyBtn = rowMenu?.querySelector('[data-action="copy-id"]');
  if (!id || !copyBtn) {
    return;
  }
  try {
    await navigator.clipboard.writeText(id);
    copyBtn.textContent = "Copied";
    copyBtn.classList.add("row-menu__item--copied");
    copyBtn.classList.remove("row-menu__item--failed");
    window.clearTimeout(copyResetTimer);
    copyResetTimer = window.setTimeout(() => {
      copyBtn.textContent = "Copy Aircraft ID";
      copyBtn.classList.remove("row-menu__item--copied");
    }, 1200);
  } catch {
    copyBtn.textContent = "Copy failed";
    copyBtn.classList.add("row-menu__item--failed");
    copyBtn.classList.remove("row-menu__item--copied");
  }
}

function fleetRowFromEvent(event) {
  const row = event.target.closest(".data-table__row--selectable[data-aircraft-id]");
  if (!row || !fleetTarget?.contains(row)) {
    return null;
  }
  return row;
}

function renderFleetTable(rows) {
  const section = document.createElement("section");
  section.className = "panel stack";
  section.setAttribute("aria-labelledby", "live-aircraft-fleet-heading");

  const header = document.createElement("header");
  header.className = "panel__header";
  const heading = document.createElement("h2");
  heading.className = "panel-heading";
  heading.id = "live-aircraft-fleet-heading";
  heading.textContent = countLabel(rows.length, "aircraft", "aircraft");
  header.appendChild(heading);
  section.appendChild(header);

  if (rows.length === 0) {
    const empty = document.createElement("p");
    empty.className = "panel-lead";
    empty.textContent = "No aircraft returned for this organisation.";
    section.appendChild(empty);
    return section;
  }

  const wrap = document.createElement("div");
  wrap.className = "data-table-wrap";
  const table = document.createElement("table");
  table.className = "data-table";
  table.innerHTML = `
    <thead>
      <tr>
        <th scope="col">Registration</th>
        <th scope="col">Aircraft</th>
        <th scope="col">Status</th>
        <th scope="col">Visibility</th>
        <th scope="col">Slot</th>
        <th scope="col">Rank</th>
        <th scope="col">Created</th>
      </tr>
    </thead>
  `;
  const tbody = document.createElement("tbody");
  for (const row of rows) {
    const statusBadge = aircraftStatusBadge(row.status);
    const visBadge = aircraftVisibilityBadge(row.visibility);
    const slotBadge = aircraftSlotBadge(row.isFleetActiveSlot);
    const created = formatUtcDateTime(row.createdAt);
    const tr = document.createElement("tr");
    tr.className = "data-table__row data-table__row--selectable";
    tr.dataset.aircraftId = row.id;
    tr.innerHTML = `
      <th scope="row">${displayOrDash(row.registration)}</th>
      <td>${displayOrDash(row.aircraftName)}</td>
      <td><span class="badge ${statusBadge.className}">${statusBadge.label}</span></td>
      <td><span class="badge ${visBadge.className}">${visBadge.label}</span></td>
      <td><span class="badge ${slotBadge.className}">${slotBadge.label}</span></td>
      <td>${row.fleetRank ?? "—"}</td>
      <td></td>
    `;
    const createdCell = tr.lastElementChild;
    if (created.iso) {
      createdCell.innerHTML = `<time datetime="${created.iso}" title="${created.iso}">${created.display}</time><span class="text-subtle">${created.relative}</span>`;
    } else {
      createdCell.textContent = "—";
    }
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);
  wrap.appendChild(table);
  section.appendChild(wrap);
  return section;
}

async function loadFleet(organizationId) {
  closeRowMenu();
  if (!fleetTarget) {
    return;
  }
  if (!organizationId) {
    fleetTarget.replaceChildren();
    const lead = document.createElement("p");
    lead.className = "panel-lead";
    lead.textContent = "Select an organisation to load the fleet.";
    fleetTarget.appendChild(lead);
    return;
  }
  fleetTarget.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "panel-lead";
  loading.textContent = "Loading fleet…";
  fleetTarget.appendChild(loading);
  try {
    const response = await fetch(vaLiveAPIPath(organizationId, "/aircraft"), { credentials: "same-origin" });
    const payload = await response.json().catch(() => null);
    if (!response.ok) {
      fleetTarget.innerHTML = `<p class="notice notice--error">${apiErrorNotice(payload, response.status)}</p>`;
      return;
    }
    const rows = Array.isArray(payload?.data) ? payload.data : [];
    fleetTarget.replaceChildren(renderFleetTable(rows));
    document.dispatchEvent(
      new CustomEvent("live-aircraft-catalog", {
        detail: { organizationId, aircraft: rows },
      }),
    );
  } catch (error) {
    console.error("load live aircraft fleet failed", error);
    fleetTarget.innerHTML = '<p class="notice notice--error">Could not load aircraft.</p>';
  }
}

fleetTarget?.addEventListener("click", (event) => {
  const row = fleetRowFromEvent(event);
  if (!row) {
    return;
  }
  openRowMenu(event, row);
});

rowMenu?.addEventListener("click", (event) => {
  const item = event.target.closest(".row-menu__item");
  if (!item) {
    return;
  }
  event.stopPropagation();
  const action = item.dataset.action;
  if (action === "copy-id") {
    copyAircraftId();
    return;
  }
  if (action === "schedules") {
    const orgId = select?.value ?? "";
    const aircraftId = rowMenu?.dataset.aircraftId ?? "";
    if (!orgId || !aircraftId) {
      closeRowMenu();
      return;
    }
    const params = new URLSearchParams();
    params.set("organizationId", orgId);
    params.set("aircraftId", aircraftId);
    window.location.href = `/live/schedules?${params.toString()}`;
    return;
  }
  if (action === "flight-history") {
    const orgId = select?.value ?? "";
    const aircraftId = rowMenu?.dataset.aircraftId ?? "";
    if (!orgId || !aircraftId) {
      closeRowMenu();
      return;
    }
    const params = new URLSearchParams();
    params.set("organizationId", orgId);
    params.set("aircraftId", aircraftId);
    window.location.href = `/live/flights?${params.toString()}`;
  }
});

document.addEventListener("click", (event) => {
  if (rowMenu?.hidden) {
    return;
  }
  if (event.target.closest("#live-aircraft-row-menu")) {
    return;
  }
  if (event.target.closest(".data-table__row--selectable")) {
    return;
  }
  closeRowMenu();
});

document.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    closeRowMenu();
  }
});

async function loadOrganizations() {
  if (!select) {
    return;
  }
  try {
    const response = await fetch("/api/v1/va/live/connections", { credentials: "same-origin" });
    const payload = await response.json().catch(() => null);
    if (response.status === 404) {
      clearOptions();
      addOption("", "Unavailable");
      select.disabled = true;
      showStatus("Virtual airline context missing. Open Live Mode from your Discord signed link.", true);
      await loadFleet("");
      announceOrgChange("");
      return;
    }
    if (response.status === 409 && payload?.error?.code === "IF_OAUTH_REQUIRED") {
      clearOptions();
      addOption("", "Not connected");
      select.disabled = true;
      showStatus("Connect Infinite Flight under Admin → Live Mode, then reload this page.");
      await loadFleet("");
      announceOrgChange("");
      return;
    }
    if (!response.ok) {
      throw new Error(`connections request failed: ${response.status}`);
    }
    const orgs = Array.isArray(payload?.data) ? payload.data : [];
    clearOptions();
    if (orgs.length === 0) {
      addOption("", "No organisations");
      select.disabled = true;
      showStatus("No linked organisations. Connect under Admin → Live Mode and link in-game orgs you administer.");
      await loadFleet("");
      announceOrgChange("");
      return;
    }
    const queryOrgID = pageParams().get("organizationId")?.trim() ?? "";
    orgs.forEach((org, index) => {
      const name = org?.name ?? "Organisation";
      const label = `${name} · ${roleLabel(org?.role)}`;
      const selected = queryOrgID ? org?.id === queryOrgID : index === 0;
      addOption(org?.id ?? "", label, selected);
    });
    select.disabled = false;
    if (status) {
      status.hidden = true;
    }
    await loadFleet(select.value);
    announceOrgChange(select.value);
  } catch (error) {
    console.error("load live organizations failed", error);
    clearOptions();
    addOption("", "Unavailable");
    select.disabled = true;
    showStatus("Could not load organisations.", true);
    await loadFleet("");
    announceOrgChange("");
  }
}

select?.addEventListener("change", () => {
  loadFleet(select.value);
  announceOrgChange(select.value, { clearAircraftFilter: true });
});

if (select) {
  loadOrganizations();
}
