/** Shared formatters for Live Mode JSON tables (mirrors server badge labels). */

export function displayOrDash(value) {
  const s = String(value ?? "").trim();
  return s === "" ? "—" : s;
}

export function aircraftStatusBadge(status) {
  switch (status) {
    case 0:
      return { label: "Active", className: "badge--success" };
    case 1:
      return { label: "Deleted", className: "badge--error" };
    default:
      return { label: `Status ${status}`, className: "badge--muted" };
  }
}

export function aircraftVisibilityBadge(visibility) {
  switch (visibility) {
    case 1:
      return { label: "Visible", className: "badge--success" };
    case 2:
      return { label: "Hangared", className: "badge--muted" };
    case 0:
      return { label: "Unknown", className: "badge--muted" };
    default:
      return { label: `Visibility ${visibility}`, className: "badge--muted" };
  }
}

export function aircraftSlotBadge(active) {
  return active
    ? { label: "Active slot", className: "badge--success" }
    : { label: "Storage", className: "badge--muted" };
}

export function liveEnumLabel(name) {
  const s = String(name ?? "").trim();
  if (s === "") {
    return "—";
  }
  return s.replace(/([a-z])([A-Z])/g, "$1 $2");
}

export function scheduleStatusBadge(status) {
  const key = String(status ?? "").trim();
  switch (key) {
    case "Unknown":
      return { label: liveEnumLabel(key), className: "badge--muted" };
    case "Scheduled":
    case "Boarding":
    case "Boarded":
    case "InFlight":
      return { label: liveEnumLabel(key), className: "badge--success" };
    case "TaxiingToRunway":
    case "TaxiingToParking":
    case "Arrived":
      return { label: liveEnumLabel(key), className: "badge--muted" };
    case "Diverted":
    case "Delayed":
    case "Cancelled":
      return { label: liveEnumLabel(key), className: "badge--error" };
    default:
      return { label: liveEnumLabel(key) || key, className: "badge--muted" };
  }
}

export function scheduleFlightTypeBadge(flightType) {
  const key = String(flightType ?? "").trim();
  if (key === "" || key === "None") {
    return { label: liveEnumLabel(key) || "—", className: "badge--muted" };
  }
  return { label: liveEnumLabel(key), className: "badge--muted" };
}

export function flightStatusBadge(status) {
  switch (status) {
    case 0:
      return { label: "Active", className: "badge--success" };
    case 1:
      return { label: "Completed", className: "badge--muted" };
    case 2:
      return { label: "Cancelled", className: "badge--error" };
    default:
      return { label: `Status ${status}`, className: "badge--muted" };
  }
}

export function formatRoute(origin, dest) {
  const o = displayOrDash(origin);
  const d = displayOrDash(dest);
  return `${o} → ${d}`;
}

export function formatFlightTimeMinutes(minutes) {
  const total = Math.max(0, Math.round(Number(minutes) || 0));
  const h = Math.floor(total / 60);
  const m = total % 60;
  if (h > 0 && m > 0) {
    return `${h}h ${m}m`;
  }
  if (h > 0) {
    return `${h}h`;
  }
  return `${m}m`;
}

export function formatCrew(crew) {
  if (!Array.isArray(crew) || crew.length === 0) {
    return "—";
  }
  const names = crew
    .map((member) => {
      const username = member?.username?.trim?.() ?? "";
      if (username) {
        return username;
      }
      return String(member?.userId ?? "").trim();
    })
    .filter(Boolean);
  return names.length === 0 ? "—" : names.join(", ");
}

export function formatUtcDateTime(iso) {
  if (!iso) {
    return { display: "—", iso: "", relative: "" };
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return { display: "—", iso: "", relative: "" };
  }
  const display = `${date.toISOString().slice(0, 19).replace("T", " ")} UTC`;
  return {
    display,
    iso: date.toISOString(),
    relative: relativeTime(date),
  };
}

export function relativeTime(date) {
  const seconds = Math.round((date.getTime() - Date.now()) / 1000);
  const abs = Math.abs(seconds);
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" });
  if (abs < 60) {
    return rtf.format(seconds, "second");
  }
  const minutes = Math.round(seconds / 60);
  if (Math.abs(minutes) < 60) {
    return rtf.format(minutes, "minute");
  }
  const hours = Math.round(minutes / 60);
  if (Math.abs(hours) < 48) {
    return rtf.format(hours, "hour");
  }
  const days = Math.round(hours / 24);
  return rtf.format(days, "day");
}

export function countLabel(count, singular, plural) {
  if (count === 0) {
    return `No ${plural}`;
  }
  if (count === 1) {
    return `1 ${singular}`;
  }
  return `${count} ${plural}`;
}

export function apiErrorNotice(payload, status) {
  const code = payload?.error?.code ?? "";
  if (status === 403) {
    return "Forbidden.";
  }
  if (code === "IF_OAUTH_REQUIRED") {
    return "Connect Infinite Flight under Admin → Live Mode, then reload this page.";
  }
  if (code === "NOT_FOUND") {
    return "Organisation not linked to this virtual airline.";
  }
  if (code === "IF_LIVE_UPSTREAM") {
    return "Could not load data from Infinite Flight.";
  }
  return payload?.error?.message ?? "Could not load data.";
}

export function vaLiveAPIPath(orgId, suffix) {
  return `/api/v1/va/live/organizations/${encodeURIComponent(orgId)}${suffix}`;
}
