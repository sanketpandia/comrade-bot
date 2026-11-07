import {
  apiErrorNotice,
  countLabel,
  displayOrDash,
  formatCrew,
  formatFlightTimeMinutes,
  formatRoute,
  formatUtcDateTime,
  flightStatusBadge,
  vaLiveAPIPath,
} from "./format.mjs";

const flightsTarget = document.getElementById("live-flights");
const filterWrap = document.getElementById("live-flights-aircraft-filter-wrap");
const filterSelect = document.getElementById("live-flights-aircraft-filter");
const rowMenu = document.getElementById("live-flights-row-menu");
const menuPad = 8;
const pageSize = 100;

let activeRow = null;
let copyResetTimer = null;
let catalog = [];
let currentOrgId = "";

function pageParams() {
  return new URLSearchParams(window.location.search);
}

function readAircraftFilter() {
  return pageParams().get("aircraftId")?.trim() ?? filterSelect?.value?.trim() ?? "";
}

function readPage() {
  const page = Number.parseInt(pageParams().get("page") ?? "1", 10);
  return Number.isFinite(page) && page > 0 ? page : 1;
}

function setQueryState({ organizationId, aircraftId, page }) {
  const params = pageParams();
  if (organizationId) {
    params.set("organizationId", organizationId);
  }
  if (aircraftId) {
    params.set("aircraftId", aircraftId);
  } else {
    params.delete("aircraftId");
  }
  if (page && page > 1) {
    params.set("page", String(page));
  } else {
    params.delete("page");
  }
  const query = params.toString();
  const url = query ? `${window.location.pathname}?${query}` : window.location.pathname;
  history.replaceState(null, "", url);
  if (filterSelect && filterSelect.value !== aircraftId) {
    filterSelect.value = aircraftId ?? "";
  }
}

function populateFilterOptions(organizationId, aircraft) {
  if (!filterSelect || !filterWrap) {
    return;
  }
  const selected = readAircraftFilter();
  filterSelect.replaceChildren();
  const prompt = document.createElement("option");
  prompt.value = "";
  prompt.textContent = "Select an aircraft";
  prompt.selected = selected === "";
  filterSelect.appendChild(prompt);
  for (const row of aircraft) {
    const option = document.createElement("option");
    option.value = row.id;
    const reg = displayOrDash(row.registration);
    const name = displayOrDash(row.aircraftName);
    option.textContent = `${reg} · ${name}`;
    option.selected = selected === row.id;
    filterSelect.appendChild(option);
  }
  filterWrap.hidden = !organizationId;
}

async function fetchAircraftCatalog(organizationId) {
  if (!organizationId) {
    catalog = [];
    populateFilterOptions("", []);
    return;
  }
  const response = await fetch(vaLiveAPIPath(organizationId, "/aircraft"), { credentials: "same-origin" });
  const payload = await response.json().catch(() => null);
  if (!response.ok) {
    throw new Error(apiErrorNotice(payload, response.status));
  }
  catalog = Array.isArray(payload?.data) ? payload.data : [];
  populateFilterOptions(organizationId, catalog);
}

function closeRowMenu() {
  if (!rowMenu) {
    return;
  }
  rowMenu.hidden = true;
  rowMenu.dataset.aircraftId = "";
  rowMenu.dataset.flightId = "";
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
  return { x: Math.max(menuPad, x), y: Math.max(menuPad, y) };
}

function resetCopyButton(copyBtn, defaultLabel) {
  if (!copyBtn) {
    return;
  }
  copyBtn.textContent = defaultLabel;
  copyBtn.classList.remove("row-menu__item--copied", "row-menu__item--failed");
}

function openRowMenu(event, row) {
  if (!rowMenu || !row) {
    return;
  }
  event.preventDefault();
  event.stopPropagation();
  flightsTarget?.querySelectorAll(".data-table__row--active").forEach((el) => {
    el.classList.remove("data-table__row--active");
  });
  row.classList.add("data-table__row--active");
  activeRow = row;
  rowMenu.dataset.aircraftId = row.dataset.aircraftId ?? "";
  rowMenu.dataset.flightId = row.dataset.flightId ?? "";
  rowMenu.hidden = false;
  rowMenu.style.left = `${event.clientX}px`;
  rowMenu.style.top = `${event.clientY}px`;
  resetCopyButton(rowMenu.querySelector('[data-action="copy-aircraft-id"]'), "Copy Aircraft ID");
  resetCopyButton(rowMenu.querySelector('[data-action="copy-flight-id"]'), "Copy Flight ID");
  requestAnimationFrame(() => {
    const { x, y } = clampMenuPosition(event.clientX, event.clientY);
    rowMenu.style.left = `${x}px`;
    rowMenu.style.top = `${y}px`;
  });
}

async function copyMenuValue(buttonSelector, value, defaultLabel) {
  const copyBtn = rowMenu?.querySelector(buttonSelector);
  if (!value || !copyBtn) {
    return;
  }
  try {
    await navigator.clipboard.writeText(value);
    copyBtn.textContent = "Copied";
    copyBtn.classList.add("row-menu__item--copied");
    copyBtn.classList.remove("row-menu__item--failed");
    window.clearTimeout(copyResetTimer);
    copyResetTimer = window.setTimeout(() => resetCopyButton(copyBtn, defaultLabel), 1200);
  } catch {
    copyBtn.textContent = "Copy failed";
    copyBtn.classList.add("row-menu__item--failed");
    copyBtn.classList.remove("row-menu__item--copied");
  }
}

function renderPager(page, totalPages, onNavigate) {
  if (totalPages <= 1) {
    return null;
  }
  const nav = document.createElement("nav");
  nav.className = "stack";
  nav.setAttribute("aria-label", "Flight history pagination");
  const prev = document.createElement("button");
  prev.type = "button";
  prev.className = "btn btn--ghost";
  prev.textContent = "Previous";
  prev.disabled = page <= 1;
  prev.addEventListener("click", () => onNavigate(page - 1));
  const label = document.createElement("p");
  label.className = "panel-lead";
  label.textContent = `Page ${page} of ${totalPages}`;
  const next = document.createElement("button");
  next.type = "button";
  next.className = "btn btn--ghost";
  next.textContent = "Next";
  next.disabled = page >= totalPages;
  next.addEventListener("click", () => onNavigate(page + 1));
  nav.append(prev, label, next);
  return nav;
}

function renderFlights(payload, aircraftId) {
  const rows = Array.isArray(payload?.data) ? payload.data : [];
  const section = document.createElement("section");
  section.className = "panel stack";
  section.setAttribute("aria-labelledby", "live-flights-heading");

  const header = document.createElement("header");
  header.className = "panel__header panel__header--wrap";
  const heading = document.createElement("h2");
  heading.className = "panel-heading";
  heading.id = "live-flights-heading";
  heading.textContent = aircraftId
    ? countLabel(payload.totalCount ?? rows.length, "flight", "flights")
    : "Flight history";
  header.appendChild(heading);
  section.appendChild(header);

  if (!aircraftId) {
    const prompt = document.createElement("p");
    prompt.className = "panel-lead";
    prompt.textContent = "Select an aircraft to load flight history.";
    section.appendChild(prompt);
    return section;
  }

  if (rows.length === 0) {
    const empty = document.createElement("p");
    empty.className = "panel-lead";
    empty.textContent = "No flights in the log for this aircraft.";
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
        <th scope="col">Callsign</th>
        <th scope="col">Route</th>
        <th scope="col">Status</th>
        <th scope="col">Departure</th>
        <th scope="col">Arrival</th>
        <th scope="col">Flight time</th>
        <th scope="col">Crew</th>
      </tr>
    </thead>
  `;
  const tbody = document.createElement("tbody");
  for (const flight of rows) {
    const statusBadge = flightStatusBadge(flight.status);
    const dep = formatUtcDateTime(flight.actualDepartureUtc);
    const arr = formatUtcDateTime(flight.actualArrivalUtc);
    const tr = document.createElement("tr");
    tr.className = "data-table__row data-table__row--selectable";
    tr.dataset.flightId = flight.id;
    tr.dataset.aircraftId = flight.aircraftId;
    const depCell = document.createElement("td");
    const arrCell = document.createElement("td");
    if (dep.iso) {
      depCell.innerHTML = `<time datetime="${dep.iso}" title="${dep.iso}">${dep.display}</time><span class="text-subtle">${dep.relative}</span>`;
    } else {
      depCell.textContent = "—";
    }
    if (arr.iso) {
      arrCell.innerHTML = `<time datetime="${arr.iso}" title="${arr.iso}">${arr.display}</time><span class="text-subtle">${arr.relative}</span>`;
    } else {
      arrCell.textContent = "—";
    }
    tr.innerHTML = `
      <th scope="row">${displayOrDash(flight.callsign)}</th>
      <td>${formatRoute(flight.originIcao, flight.destinationIcao)}</td>
      <td><span class="badge ${statusBadge.className}">${statusBadge.label}</span></td>
    `;
    tr.append(depCell, arrCell);
    const timeCell = document.createElement("td");
    timeCell.textContent = formatFlightTimeMinutes(flight.flightTimeMinutes);
    const crewCell = document.createElement("td");
    crewCell.textContent = formatCrew(flight.crew);
    tr.append(timeCell, crewCell);
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);
  wrap.appendChild(table);
  section.appendChild(wrap);
  return section;
}

async function loadFlights(organizationId, { page = readPage(), aircraftId = readAircraftFilter() } = {}) {
  closeRowMenu();
  if (!flightsTarget) {
    return;
  }
  if (!organizationId) {
    flightsTarget.replaceChildren();
    const lead = document.createElement("p");
    lead.className = "panel-lead";
    lead.textContent = "Select an organisation to load flight history.";
    flightsTarget.appendChild(lead);
    if (filterWrap) {
      filterWrap.hidden = true;
    }
    return;
  }
  currentOrgId = organizationId;
  setQueryState({ organizationId, aircraftId, page });
  flightsTarget.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "panel-lead";
  loading.textContent = aircraftId ? "Loading flights…" : "Loading…";
  flightsTarget.appendChild(loading);
  try {
    await fetchAircraftCatalog(organizationId);
    if (!aircraftId) {
      flightsTarget.replaceChildren(renderFlights({ data: [] }, ""));
      return;
    }
    const params = new URLSearchParams();
    params.set("page", String(page));
    params.set("pageSize", String(pageSize));
    const response = await fetch(
      `${vaLiveAPIPath(organizationId, `/aircraft/${encodeURIComponent(aircraftId)}/flights`)}?${params.toString()}`,
      { credentials: "same-origin" },
    );
    const payload = await response.json().catch(() => null);
    if (!response.ok) {
      flightsTarget.innerHTML = `<p class="notice notice--error">${apiErrorNotice(payload, response.status)}</p>`;
      return;
    }
    const section = renderFlights(payload, aircraftId);
    const pager = renderPager(payload.page ?? page, payload.totalPages ?? 1, (nextPage) => {
      loadFlights(organizationId, { page: nextPage, aircraftId: readAircraftFilter() });
    });
    flightsTarget.replaceChildren(section);
    if (pager) {
      flightsTarget.appendChild(pager);
    }
  } catch (error) {
    console.error("load live flights failed", error);
    flightsTarget.innerHTML = `<p class="notice notice--error">${String(error.message ?? error)}</p>`;
  }
}

function onOrgChanged(event) {
  const organizationId = event.detail?.organizationId ?? "";
  const aircraftId = event.detail?.clearAircraftFilter ? "" : readAircraftFilter();
  if (event.detail?.clearAircraftFilter && filterSelect) {
    filterSelect.value = "";
  }
  loadFlights(organizationId, { page: 1, aircraftId });
}

if (flightsTarget) {
  document.addEventListener("live-org-changed", onOrgChanged);
  document.addEventListener("live-aircraft-catalog", (event) => {
    if (event.detail?.organizationId !== currentOrgId) {
      return;
    }
    catalog = Array.isArray(event.detail?.aircraft) ? event.detail.aircraft : [];
    populateFilterOptions(currentOrgId, catalog);
  });

  flightsTarget.addEventListener("click", (event) => {
    const row = event.target.closest(".data-table__row--selectable[data-flight-id]");
    if (!row || !flightsTarget.contains(row)) {
      return;
    }
    openRowMenu(event, row);
  });

  filterSelect?.addEventListener("change", () => {
    const orgSelect = document.getElementById("live-org-select");
    loadFlights(orgSelect?.value ?? "", { page: 1, aircraftId: filterSelect.value });
  });

  rowMenu?.addEventListener("click", (event) => {
    const item = event.target.closest(".row-menu__item");
    if (!item) {
      return;
    }
    event.stopPropagation();
    const action = item.dataset.action;
    if (action === "copy-aircraft-id") {
      copyMenuValue('[data-action="copy-aircraft-id"]', rowMenu.dataset.aircraftId ?? "", "Copy Aircraft ID");
      return;
    }
    if (action === "copy-flight-id") {
      copyMenuValue('[data-action="copy-flight-id"]', rowMenu.dataset.flightId ?? "", "Copy Flight ID");
    }
  });

  document.addEventListener("click", (event) => {
    if (rowMenu?.hidden) {
      return;
    }
    if (event.target.closest("#live-flights-row-menu")) {
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

  const orgSelect = document.getElementById("live-org-select");
  const initialOrg = pageParams().get("organizationId")?.trim() ?? orgSelect?.value ?? "";
  if (initialOrg) {
    loadFlights(initialOrg);
  }
}
