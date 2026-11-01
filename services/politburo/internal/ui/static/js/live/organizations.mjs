const select = document.getElementById("live-org-select");
const status = document.getElementById("live-org-status");

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
  if (!select) {
    return;
  }
  select.replaceChildren();
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

async function loadOrganizations() {
  if (!select) {
    return;
  }
  try {
    const response = await fetch("/api/v1/live/organizations", { credentials: "same-origin" });
    const payload = await response.json().catch(() => null);
    if (response.status === 409 && payload?.error?.code === "IF_OAUTH_REQUIRED") {
      clearOptions();
      addOption("", "Not connected");
      select.disabled = true;
      showStatus("Connect Infinite Flight under Admin → Live Mode, then reload this page.");
      return;
    }
    if (!response.ok) {
      throw new Error(`organizations request failed: ${response.status}`);
    }
    const orgs = Array.isArray(payload?.data) ? payload.data : [];
    clearOptions();
    if (orgs.length === 0) {
      addOption("", "No organisations");
      select.disabled = true;
      showStatus("No organisations where you are Admin or Owner.");
      return;
    }
    orgs.forEach((org, index) => {
      const name = org?.name ?? "Organisation";
      const label = `${name} · ${roleLabel(org?.role)}`;
      addOption(org?.id ?? "", label, index === 0);
    });
    select.disabled = false;
    if (status) {
      status.hidden = true;
    }
  } catch (error) {
    console.error("load live organizations failed", error);
    clearOptions();
    addOption("", "Unavailable");
    select.disabled = true;
    showStatus("Could not load organisations.", true);
  }
}

loadOrganizations();
