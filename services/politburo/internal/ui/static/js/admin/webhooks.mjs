import { initMultiselect } from '/static/js/ui/multiselect.mjs';

const FEATURE = 'active_flights';
const FEATURE_ASSIGNMENT = 'live_mode_assignment';
const CONFIG_URL = '/api/v1/va/config?configType=WEBHOOKS';
const TRIGGER_URL = '/api/v1/va/webhook/trigger';
const FREQUENCIES = new Set(['30m', '60m', '120m']);

const form = document.getElementById('admin-webhooks-form');
const assignmentForm = document.getElementById('admin-webhooks-assignment-form');
const statusEl = document.getElementById('admin-webhooks-status');
const urlInput = document.getElementById('webhook-url');
const frequencyInput = document.getElementById('webhook-frequency');
const vaCodeInput = document.getElementById('webhook-va-code');
const saveButton = document.getElementById('webhook-save');
const sendButton = document.getElementById('webhook-send');
const assignmentUrlInput = document.getElementById('webhook-assignment-url');
const assignmentSaveButton = document.getElementById('webhook-assignment-save');
const pilotState = document.getElementById('webhook-pilot-state');
const pilotInputs = pilotState
  ? [...pilotState.querySelectorAll('input[type="checkbox"]')]
  : [];

let stored = {};
let savedSnapshot = null;
let assignmentSavedSnapshot = null;
let busy = false;
let loaded = false;

initMultiselect(pilotState);

function showStatus(message, isError) {
  if (!statusEl) return;
  statusEl.textContent = message;
  statusEl.hidden = !message;
  statusEl.classList.toggle('panel-message--error', Boolean(isError));
}

function errorMessage(body, fallback) {
  return body?.error?.message || fallback;
}

function controls() {
  return [
    urlInput,
    frequencyInput,
    vaCodeInput,
    saveButton,
    sendButton,
    assignmentUrlInput,
    assignmentSaveButton,
    ...pilotInputs,
    pilotState?.querySelector('.multiselect__trigger'),
  ].filter(Boolean);
}

function setBusy(next) {
  busy = next;
  for (const control of controls()) {
    if (control === sendButton) continue;
    control.disabled = next;
  }
  syncSend();
}

function selectedPilotStates() {
  return pilotInputs.filter((input) => input.checked).map((input) => input.value);
}

function validWebhookURL(value) {
  let parsed;
  try {
    parsed = new URL(value);
  } catch {
    return false;
  }
  if (parsed.protocol !== 'https:' || parsed.username || parsed.password || parsed.port) return false;
  const host = parsed.hostname.toLowerCase();
  if (host !== 'discord.com' && host !== 'discordapp.com') return false;
  const parts = parsed.pathname.replace(/\/$/, '').split('/');
  return parts.length === 5 && parts[1] === 'api' && parts[2] === 'webhooks' && parts[3] !== '' && parts[4] !== '';
}

function formEntry() {
  const existing = stored[FEATURE];
  const previous = existing && typeof existing.filters === 'object' && existing.filters && !Array.isArray(existing.filters)
    ? existing.filters
    : {};
  const filters = { ...previous };
  const states = selectedPilotStates();
  if (states.length > 0) {
    filters.pilotState = states;
  } else {
    delete filters.pilotState;
  }
  const vaCode = vaCodeInput?.value.trim() ?? '';
  if (vaCode) {
    filters.vaCode = vaCode;
  } else {
    delete filters.vaCode;
  }
  return {
    url: urlInput?.value.trim() ?? '',
    frequency: frequencyInput?.value ?? '',
    filters,
  };
}

function assignmentEntry() {
  return { url: assignmentUrlInput?.value.trim() ?? '' };
}

function snapshot(entry) {
  const filters = { ...entry.filters };
  if (Array.isArray(filters.pilotState)) {
    filters.pilotState = [...filters.pilotState].sort();
  }
  return JSON.stringify({
    url: entry.url,
    frequency: entry.frequency,
    filters,
  });
}

function assignmentSnapshot(entry) {
  return JSON.stringify({ url: entry.url });
}

function applyFeature(feature) {
  const url = typeof feature?.url === 'string' ? feature.url : '';
  const frequency = typeof feature?.frequency === 'string' ? feature.frequency : '';
  const filters = feature?.filters && typeof feature.filters === 'object' ? feature.filters : {};
  const states = new Set(Array.isArray(filters.pilotState) ? filters.pilotState : []);
  if (urlInput) {
    urlInput.value = url;
    urlInput.placeholder = 'https://discord.com/api/webhooks/…';
  }
  if (frequencyInput) {
    frequencyInput.value = FREQUENCIES.has(frequency) ? frequency : '';
  }
  for (const input of pilotInputs) {
    input.checked = states.has(input.value);
    input.dispatchEvent(new Event('change'));
  }
  if (vaCodeInput) {
    vaCodeInput.value = typeof filters.vaCode === 'string' ? filters.vaCode : '';
  }
}

function applyAssignmentFeature(feature) {
  const url = typeof feature?.url === 'string' ? feature.url : '';
  if (assignmentUrlInput) {
    assignmentUrlInput.value = url;
    assignmentUrlInput.placeholder = 'https://discord.com/api/webhooks/…';
  }
}

function rememberSaved() {
  savedSnapshot = stored[FEATURE] ? snapshot(formEntry()) : null;
  assignmentSavedSnapshot = stored[FEATURE_ASSIGNMENT]
    ? assignmentSnapshot(assignmentEntry())
    : null;
  syncSend();
}

function syncSend() {
  if (!sendButton) return;
  const current = formEntry();
  const unchanged = savedSnapshot !== null && snapshot(current) === savedSnapshot;
  sendButton.disabled = busy || !unchanged || !validWebhookURL(current.url);
  if (savedSnapshot === null) {
    sendButton.title = 'Save a webhook before sending.';
  } else if (!unchanged) {
    sendButton.title = 'Save changes before sending.';
  } else {
    sendButton.title = '';
  }
}

function validate(entry) {
  if (!validWebhookURL(entry.url)) {
    return 'Use an HTTPS Discord webhook URL (discord.com or discordapp.com).';
  }
  if (!FREQUENCIES.has(entry.frequency)) {
    return 'Choose a frequency.';
  }
  if (typeof entry.filters.vaCode === 'string' && entry.filters.vaCode.trim() === '') {
    return 'VA code must not be empty.';
  }
  return '';
}

function validateAssignment(entry) {
  if (!validWebhookURL(entry.url)) {
    return 'Use an HTTPS Discord webhook URL (discord.com or discordapp.com).';
  }
  return '';
}

async function loadConfig() {
  try {
    const response = await fetch(CONFIG_URL, { credentials: 'same-origin' });
    const body = await response.json().catch(() => ({}));
    if (response.status === 404) {
      stored = {};
      applyFeature(null);
      applyAssignmentFeature(null);
      showStatus('No webhook saved yet.', false);
      loaded = true;
      return;
    }
    if (!response.ok) {
      showStatus(errorMessage(body, `Could not load webhooks (${response.status}).`), true);
      return;
    }
    stored = body?.data?.config && typeof body.data.config === 'object' ? body.data.config : {};
    applyFeature(stored[FEATURE] ?? null);
    applyAssignmentFeature(stored[FEATURE_ASSIGNMENT] ?? null);
    const any = stored[FEATURE] || stored[FEATURE_ASSIGNMENT];
    showStatus(any ? '' : 'No webhooks saved yet.', false);
    loaded = true;
  } catch {
    showStatus('Could not load webhooks.', true);
  } finally {
    if (loaded) {
      setBusy(false);
      rememberSaved();
    }
  }
}

async function saveConfig(event) {
  event.preventDefault();
  if (busy) return;
  const entry = formEntry();
  const problem = validate(entry);
  if (problem) {
    showStatus(problem, true);
    return;
  }
  const next = { ...stored, [FEATURE]: entry };
  setBusy(true);
  try {
    const response = await fetch(CONFIG_URL, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(next),
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      showStatus(errorMessage(body, `Could not save webhook (${response.status}).`), true);
      return;
    }
    stored = body?.data?.config && typeof body.data.config === 'object' ? body.data.config : next;
    applyFeature(stored[FEATURE] ?? entry);
    applyAssignmentFeature(stored[FEATURE_ASSIGNMENT] ?? null);
    showStatus('Saved.', false);
  } catch {
    showStatus('Could not save webhook.', true);
  } finally {
    setBusy(false);
    rememberSaved();
  }
}

async function saveAssignmentConfig(event) {
  event.preventDefault();
  if (busy) return;
  const entry = assignmentEntry();
  const problem = validateAssignment(entry);
  if (problem) {
    showStatus(problem, true);
    return;
  }
  const next = { ...stored, [FEATURE_ASSIGNMENT]: entry };
  setBusy(true);
  try {
    const response = await fetch(CONFIG_URL, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(next),
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      showStatus(errorMessage(body, `Could not save webhook (${response.status}).`), true);
      return;
    }
    stored = body?.data?.config && typeof body.data.config === 'object' ? body.data.config : next;
    applyFeature(stored[FEATURE] ?? null);
    applyAssignmentFeature(stored[FEATURE_ASSIGNMENT] ?? entry);
    showStatus('Saved.', false);
  } catch {
    showStatus('Could not save webhook.', true);
  } finally {
    setBusy(false);
    rememberSaved();
  }
}

function triggerMessage(data) {
  const outcome = data?.outcome;
  if (outcome === 'success') {
    const count = Number(data.discordMessages) || 0;
    if (count === 1) return 'Sent 1 Discord message.';
    if (count > 1) return `Sent ${count} Discord messages.`;
    return 'Sent.';
  }
  if (outcome === 'skipped' && data.skipReason === 'no_preferred_servers') {
    return 'Nothing sent. This airline has no preferred servers.';
  }
  if (outcome === 'skipped') {
    return 'Nothing sent.';
  }
  return data?.errorMessage || 'Discord rejected the webhook.';
}

async function sendNow() {
  if (busy || sendButton?.disabled) return;
  setBusy(true);
  try {
    const response = await fetch(TRIGGER_URL, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ webhookType: FEATURE }),
    });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      showStatus(errorMessage(body, `Could not send webhook (${response.status}).`), true);
      return;
    }
    const failed = body?.data?.outcome === 'error';
    showStatus(triggerMessage(body?.data), failed);
  } catch {
    showStatus('Could not send webhook.', true);
  } finally {
    setBusy(false);
  }
}

form?.addEventListener('submit', saveConfig);
form?.addEventListener('input', syncSend);
form?.addEventListener('change', syncSend);
assignmentForm?.addEventListener('submit', saveAssignmentConfig);
sendButton?.addEventListener('click', sendNow);

loadConfig();
