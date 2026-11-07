import {
  apiErrorNotice,
  displayOrDash,
  formatRoute,
  formatUtcDateTime,
} from "./format.mjs";

const target = document.getElementById("live-bookings");
let currentOrgId = "";

function pageParams() {
  return new URLSearchParams(window.location.search);
}

function setQueryState({ organizationId }) {
  const params = new URLSearchParams(window.location.search);
  if (organizationId) {
    params.set("organizationId", organizationId);
  } else {
    params.delete("organizationId");
  }
  const qs = params.toString();
  const url = qs ? `${window.location.pathname}?${qs}` : window.location.pathname;
  window.history.replaceState(null, "", url);
}

function outcomeBadge(outcome) {
  if (outcome === "success") {
    return { label: "Success", className: "badge--success" };
  }
  if (outcome === "error") {
    return { label: "Error", className: "badge--error" };
  }
  return { label: displayOrDash(outcome), className: "badge--muted" };
}

function renderTable(items) {
  if (!target) {
    return;
  }
  if (!items.length) {
    target.replaceChildren();
    const lead = document.createElement("p");
    lead.className = "panel-lead";
    lead.textContent = "No crew self-assignments recorded yet for this organisation.";
    target.appendChild(lead);
    return;
  }

  const wrap = document.createElement("div");
  wrap.className = "data-table-wrap";
  const table = document.createElement("table");
  table.className = "data-table";
  const thead = document.createElement("thead");
  thead.innerHTML = `
    <tr>
      <th scope="col">Time (UTC)</th>
      <th scope="col">Organisation ID</th>
      <th scope="col">Pilot</th>
      <th scope="col">Registration</th>
      <th scope="col">Route</th>
      <th scope="col">Callsign</th>
      <th scope="col">Outcome</th>
      <th scope="col">HTTP</th>
    </tr>`;
  table.appendChild(thead);
  const tbody = document.createElement("tbody");
  for (const row of items) {
    const tr = document.createElement("tr");
    tr.className = "data-table__row";
    const pilot = row.actorIfcUsername || row.actorUsername || "—";
    const route = row.route?.trim()
      ? row.route
      : formatRoute(row.originIcao, row.destinationIcao);
    const badge = outcomeBadge(row.outcome);
    const started = formatUtcDateTime(row.startedAt);
    const timeCell = started.iso
      ? `<time datetime="${started.iso}" title="${started.iso}">${started.display}</time>`
      : "—";
    tr.innerHTML = `
      <td>${timeCell}</td>
      <td><code>${displayOrDash(row.organizationId)}</code></td>
      <td>${displayOrDash(pilot)}</td>
      <td>${displayOrDash(row.registration)}</td>
      <td>${displayOrDash(route)}</td>
      <td>${displayOrDash(row.callsign)}</td>
      <td><span class="badge ${badge.className}">${badge.label}</span></td>
      <td>${row.upstreamStatus ?? "—"}</td>`;
    if (row.errorMessage) {
      tr.title = String(row.errorMessage);
    }
    tbody.appendChild(tr);
  }
  table.appendChild(tbody);
  wrap.appendChild(table);
  target.replaceChildren(wrap);
}

async function loadBookings(organizationId) {
  if (!target) {
    return;
  }
  if (!organizationId) {
    currentOrgId = "";
    setQueryState({ organizationId: "" });
    target.replaceChildren();
    const lead = document.createElement("p");
    lead.className = "panel-lead";
    lead.textContent = "Select an organisation to load bookings.";
    target.appendChild(lead);
    return;
  }

  currentOrgId = organizationId;
  setQueryState({ organizationId });

  target.replaceChildren();
  const loading = document.createElement("p");
  loading.className = "panel-lead";
  loading.textContent = "Loading bookings…";
  target.appendChild(loading);

  try {
    const params = new URLSearchParams({ organizationId });
    const response = await fetch(`/api/v1/va/live/schedule-bookings?${params.toString()}`, {
      credentials: "include",
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      target.replaceChildren(apiErrorNotice(payload, response.status));
      return;
    }
    const items = Array.isArray(payload?.data) ? payload.data : [];
    renderTable(items);
  } catch (err) {
    target.replaceChildren(apiErrorNotice({ error: { message: String(err) } }, 0));
  }
}

function onOrgChanged(event) {
  const organizationId = event.detail?.organizationId ?? "";
  if (organizationId === currentOrgId) {
    return;
  }
  loadBookings(organizationId);
}

if (target) {
  document.addEventListener("live-org-changed", onOrgChanged);
  const orgSelect = document.getElementById("live-org-select");
  const initialOrg = pageParams().get("organizationId")?.trim() ?? orgSelect?.value ?? "";
  if (initialOrg) {
    loadBookings(initialOrg);
  }
}
