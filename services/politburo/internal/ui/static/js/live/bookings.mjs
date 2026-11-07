import {
  apiErrorNotice,
  displayOrDash,
  formatRoute,
  formatUtcDateTime,
} from "./format.mjs";

const target = document.getElementById("live-bookings");

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
    lead.textContent = "No crew self-assignments recorded yet.";
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

async function loadBookings() {
  if (!target) {
    return;
  }
  try {
    const response = await fetch("/api/v1/va/live/schedule-bookings", { credentials: "include" });
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

loadBookings();
