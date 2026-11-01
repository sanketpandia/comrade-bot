const schedulesTarget = document.getElementById("live-schedules");
const rowMenu = document.getElementById("live-schedules-row-menu");
const listPath = "/live/schedules/list";
const menuPad = 8;

let activeRow = null;
let copyResetTimer = null;

function pageParams() {
  return new URLSearchParams(window.location.search);
}

function readAircraftFilter() {
  return pageParams().get("aircraftId")?.trim() ?? "";
}

function setAircraftFilter(aircraftId) {
  const params = pageParams();
  if (aircraftId) {
    params.set("aircraftId", aircraftId);
  } else {
    params.delete("aircraftId");
  }
  const query = params.toString();
  const url = query ? `${window.location.pathname}?${query}` : window.location.pathname;
  history.replaceState(null, "", url);
}

function schedulesURL(organizationId, aircraftId) {
  const params = new URLSearchParams();
  params.set("organizationId", organizationId);
  if (aircraftId) {
    params.set("aircraftId", aircraftId);
  }
  return `${listPath}?${params.toString()}`;
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
  return {
    x: Math.max(menuPad, x),
    y: Math.max(menuPad, y),
  };
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
    copyResetTimer = window.setTimeout(() => {
      resetCopyButton(copyBtn, defaultLabel);
    }, 1200);
  } catch {
    copyBtn.textContent = "Copy failed";
    copyBtn.classList.add("row-menu__item--failed");
    copyBtn.classList.remove("row-menu__item--copied");
  }
}

function scheduleRowFromEvent(event) {
  const row = event.target.closest(".data-table__row--selectable[data-schedule-id]");
  if (!row || !schedulesTarget?.contains(row)) {
    return null;
  }
  return row;
}

async function loadSchedules(organizationId) {
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
    return;
  }
  const aircraftId = readAircraftFilter();
  schedulesTarget.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "panel-lead";
  loading.textContent = "Loading schedules…";
  schedulesTarget.appendChild(loading);
  try {
    const response = await fetch(schedulesURL(organizationId, aircraftId), { credentials: "same-origin" });
    if (response.status === 403) {
      schedulesTarget.innerHTML = '<p class="panel-message panel-message--error">Forbidden.</p>';
      return;
    }
    if (!response.ok) {
      throw new Error(`schedules request failed: ${response.status}`);
    }
    schedulesTarget.innerHTML = await response.text();
  } catch (error) {
    console.error("load live schedules failed", error);
    schedulesTarget.innerHTML =
      '<p class="panel-message panel-message--error">Could not load schedules.</p>';
  }
}

function onOrgChanged(event) {
  const organizationId = event.detail?.organizationId ?? "";
  if (event.detail?.clearAircraftFilter) {
    setAircraftFilter("");
  }
  loadSchedules(organizationId);
}

if (schedulesTarget) {
  document.addEventListener("live-org-changed", onOrgChanged);

  schedulesTarget.addEventListener("click", (event) => {
    const row = scheduleRowFromEvent(event);
    if (!row) {
      return;
    }
    openRowMenu(event, row);
  });

  schedulesTarget.addEventListener("change", (event) => {
    if (event.target.id !== "live-schedules-aircraft-filter") {
      return;
    }
    const select = document.getElementById("live-org-select");
    setAircraftFilter(event.target.value);
    loadSchedules(select?.value ?? "");
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
}
