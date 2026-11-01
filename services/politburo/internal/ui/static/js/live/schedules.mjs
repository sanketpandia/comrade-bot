import {
  apiErrorNotice,
  countLabel,
  displayOrDash,
  formatRoute,
  formatUtcDateTime,
  scheduleStatusBadge,
  vaLiveAPIPath,
} from "./format.mjs";

const schedulesTarget = document.getElementById("live-schedules");
const filterWrap = document.getElementById("live-schedules-aircraft-filter-wrap");
const filterSelect = document.getElementById("live-schedules-aircraft-filter");
const rowMenu = document.getElementById("live-schedules-row-menu");
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

function aircraftByID(id) {
  return catalog.find((row) => row.id === id);
}

function populateFilterOptions(organizationId, aircraft) {
  if (!filterSelect || !filterWrap) {
    return;
  }
  const selected = readAircraftFilter();
  filterSelect.replaceChildren();
  const allOption = document.createElement("option");
  allOption.value = "";
  allOption.textContent = "All aircraft";
  allOption.selected = selected === "";
  filterSelect.appendChild(allOption);
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
  rowMenu.dataset.scheduleId = "";
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
  schedulesTarget?.querySelectorAll(".data-table__row--active").forEach((el) => {
    el.classList.remove("data-table__row--active");
  });
  row.classList.add("data-table__row--active");
  activeRow = row;
  rowMenu.dataset.aircraftId = row.dataset.aircraftId ?? "";
  rowMenu.dataset.scheduleId = row.dataset.scheduleId ?? "";
  rowMenu.hidden = false;
  rowMenu.style.left = `${event.clientX}px`;
  rowMenu.style.top = `${event.clientY}px`;
  resetCopyButton(rowMenu.querySelector('[data-action="copy-aircraft-id"]'), "Copy Aircraft ID");
  resetCopyButton(rowMenu.querySelector('[data-action="copy-schedule-id"]'), "Copy Schedule ID");
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
  nav.setAttribute("aria-label", "Schedule pagination");
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

function renderSchedules(payload) {
  const rows = Array.isArray(payload?.data) ? payload.data : [];
  const section = document.createElement("section");
  section.className = "panel stack";
  section.setAttribute("aria-labelledby", "live-schedules-heading");

  const header = document.createElement("header");
  header.className = "panel__header panel__header--wrap";
  const heading = document.createElement("h2");
  heading.className = "panel-heading";
  heading.id = "live-schedules-heading";
  heading.textContent = countLabel(payload.totalCount ?? rows.length, "schedule", "schedules");
  header.appendChild(heading);
  section.appendChild(header);

  if (rows.length === 0) {
    const empty = document.createElement("p");
    empty.className = "panel-lead";
    empty.textContent = "No schedules returned for this organisation.";
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
        <th scope="col">Callsign</th>
        <th scope="col">Route</th>
        <th scope="col">Status</th>
        <th scope="col">Departure</th>
      </tr>
    </thead>
  `;
  const tbody = document.createElement("tbody");
  for (const sched of rows) {
    const ac = aircraftByID(sched.aircraftId);
    const statusBadge = scheduleStatusBadge(sched.status);
    const tr = document.createElement("tr");
    tr.className = "data-table__row data-table__row--selectable";
    tr.dataset.scheduleId = sched.id;
    tr.dataset.aircraftId = sched.aircraftId;
    const depCell = document.createElement("td");
    if (sched.isTimeless) {
      depCell.textContent = "Timeless";
    } else {
      const dep = formatUtcDateTime(sched.scheduledDepartureUtc);
      if (dep.iso) {
        depCell.innerHTML = `<time datetime="${dep.iso}" title="${dep.iso}">${dep.display}</time><span class="text-subtle">${dep.relative}</span>`;
      } else {
        depCell.textContent = "—";
      }
    }
    tr.innerHTML = `
      <th scope="row">${displayOrDash(ac?.registration)}</th>
      <td>${displayOrDash(ac?.aircraftName)}</td>
      <td>${displayOrDash(sched.callsign)}</td>
      <td>${formatRoute(sched.originIcao, sched.destinationIcao)}</td>
      <td><span class="badge ${statusBadge.className}">${statusBadge.label}</span></td>
    `;
    tr.appendChild(depCell);
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);
  wrap.appendChild(table);
  section.appendChild(wrap);
  return section;
}

async function loadSchedules(organizationId, { page = readPage(), aircraftId = readAircraftFilter() } = {}) {
  closeRowMenu();
  if (!schedulesTarget) {
    return;
  }
  if (!organizationId) {
    schedulesTarget.replaceChildren();
    const lead = document.createElement("p");
    lead.className = "panel-lead";
    lead.textContent = "Select an organisation to load schedules.";
    schedulesTarget.appendChild(lead);
    if (filterWrap) {
      filterWrap.hidden = true;
    }
    return;
  }
  currentOrgId = organizationId;
  setQueryState({ organizationId, aircraftId, page });
  schedulesTarget.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "panel-lead";
  loading.textContent = "Loading schedules…";
  schedulesTarget.appendChild(loading);
  try {
    await fetchAircraftCatalog(organizationId);
    const params = new URLSearchParams();
    params.set("page", String(page));
    params.set("pageSize", String(pageSize));
    if (aircraftId) {
      params.set("aircraftId", aircraftId);
    }
    const response = await fetch(`${vaLiveAPIPath(organizationId, "/schedules")}?${params.toString()}`, {
      credentials: "same-origin",
    });
    const payload = await response.json().catch(() => null);
    if (!response.ok) {
      schedulesTarget.innerHTML = `<p class="notice notice--error">${apiErrorNotice(payload, response.status)}</p>`;
      return;
    }
    const section = renderSchedules(payload);
    const pager = renderPager(payload.page ?? page, payload.totalPages ?? 1, (nextPage) => {
      loadSchedules(organizationId, { page: nextPage, aircraftId: readAircraftFilter() });
    });
    schedulesTarget.replaceChildren(section);
    if (pager) {
      schedulesTarget.appendChild(pager);
    }
  } catch (error) {
    console.error("load live schedules failed", error);
    schedulesTarget.innerHTML = `<p class="notice notice--error">${String(error.message ?? error)}</p>`;
  }
}

function onOrgChanged(event) {
  const organizationId = event.detail?.organizationId ?? "";
  if (event.detail?.clearAircraftFilter && filterSelect) {
    filterSelect.value = "";
  }
  loadSchedules(organizationId, { page: 1, aircraftId: event.detail?.clearAircraftFilter ? "" : readAircraftFilter() });
}

if (schedulesTarget) {
  document.addEventListener("live-org-changed", onOrgChanged);
  document.addEventListener("live-aircraft-catalog", (event) => {
    if (event.detail?.organizationId !== currentOrgId) {
      return;
    }
    catalog = Array.isArray(event.detail?.aircraft) ? event.detail.aircraft : [];
    populateFilterOptions(currentOrgId, catalog);
  });

  schedulesTarget.addEventListener("click", (event) => {
    const row = event.target.closest(".data-table__row--selectable[data-schedule-id]");
    if (!row || !schedulesTarget.contains(row)) {
      return;
    }
    openRowMenu(event, row);
  });

  filterSelect?.addEventListener("change", () => {
    const orgSelect = document.getElementById("live-org-select");
    loadSchedules(orgSelect?.value ?? "", { page: 1, aircraftId: filterSelect.value });
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
    if (action === "copy-schedule-id") {
      copyMenuValue('[data-action="copy-schedule-id"]', rowMenu.dataset.scheduleId ?? "", "Copy Schedule ID");
    }
  });

  document.addEventListener("click", (event) => {
    if (rowMenu?.hidden) {
      return;
    }
    if (event.target.closest("#live-schedules-row-menu")) {
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
    loadSchedules(initialOrg);
  }
}
