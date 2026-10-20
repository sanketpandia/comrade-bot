const PANEL = () => document.getElementById('maps-detail-panel');
const SHELL = () => document.getElementById('maps-shell');

function el(id) {
  return document.getElementById(id);
}

function formatUTC(iso) {
  const at = new Date(iso);
  if (Number.isNaN(at.getTime())) return '—';
  const pad = (n) => String(n).padStart(2, '0');
  return `${at.getUTCFullYear()}-${pad(at.getUTCMonth() + 1)}-${pad(at.getUTCDate())} `
    + `${pad(at.getUTCHours())}:${pad(at.getUTCMinutes())}:${pad(at.getUTCSeconds())} UTC`;
}

function verticalTrend(speed) {
  if (speed > 50) return { label: 'CLIMBING', className: 'maps-detail-badge--climb' };
  if (speed < -50) return { label: 'DESCENDING', className: 'maps-detail-badge--descend' };
  return { label: 'LEVEL', className: 'maps-detail-badge--level' };
}

function pilotLabel(state) {
  if (!state) return 'UNKNOWN';
  return String(state).replaceAll('_', ' ').toUpperCase();
}

function connectionLabel(status) {
  if (status === 'connected') return { text: 'CONNECTED', className: 'maps-detail-badge--connected' };
  return { text: 'DISCONNECTED', className: 'maps-detail-badge--disconnected' };
}

function setText(id, value) {
  const node = el(id);
  if (node) node.textContent = value || '—';
}

function showDetailState({ empty, loading, error, content }) {
  el('maps-detail-empty').hidden = !empty;
  el('maps-detail-loading').hidden = !loading;
  const errorEl = el('maps-detail-error');
  errorEl.hidden = !error;
  if (!error) errorEl.textContent = '';
  el('maps-detail-content').hidden = !content;
}

let detailPage = 1;
let selectedFlightId = '';

function setDetailPage(page) {
  detailPage = page === 2 ? 2 : 1;
  el('maps-detail-page-1').hidden = detailPage !== 1;
  el('maps-detail-page-2').hidden = detailPage !== 2;
  el('maps-detail-page-label').textContent = `Page ${detailPage === 1 ? '01' : '02'}/02`;
  el('maps-detail-page-prev').disabled = detailPage === 1;
  el('maps-detail-page-next').disabled = detailPage === 2;
}

function renderFlightRecord(record) {
  setText('maps-detail-callsign', record.callsign);
  setText('maps-detail-username', record.username);
  setText('maps-detail-vo', record.virtualOrganization);
  setText('maps-detail-aircraft', record.aircraftName);
  setText('maps-detail-livery', record.liveryName);
  setText('maps-detail-session', record.normalizedName);
  setText('maps-detail-speed', record.normalized?.speed ?? `${record.speed} kts`);
  setText('maps-detail-altitude', `${record.altitude} ft`);
  setText('maps-detail-vspeed', record.normalized?.verticalSpeed ?? `${record.verticalSpeed} ft/min`);
  const trend = verticalTrend(record.verticalSpeed);
  const vsBadge = el('maps-detail-vspeed-badge');
  vsBadge.textContent = trend.label;
  vsBadge.className = `maps-detail-badge ${trend.className}`;
  vsBadge.hidden = false;
  setText('maps-detail-track', `${record.track}°`);
  const pilotBadge = el('maps-detail-pilot-badge');
  pilotBadge.textContent = pilotLabel(record.normalized?.pilotState);
  const conn = connectionLabel(record.normalized?.isConnected);
  const connBadge = el('maps-detail-connection-badge');
  connBadge.textContent = conn.text;
  connBadge.className = `maps-detail-badge maps-detail-badge--connection ${conn.className}`;
  const fpl = el('maps-detail-fpl');
  const fplRequired = record.pathSync?.fplSyncRequired === true;
  fpl.textContent = fplRequired ? 'YES' : 'NO';
  fpl.className = `maps-detail-badge ${fplRequired ? 'maps-detail-badge--warn' : 'maps-detail-badge--muted'}`;
  setText('maps-detail-last-report', formatUTC(record.lastReport));
  setText('maps-detail-flight-id', record.flightId);
  setText('maps-detail-user-id', record.userId);
  setText('maps-detail-position', `${record.latitude}, ${record.longitude}`);
  setText('maps-detail-pilot-code', String(record.pilotState));
  setText('maps-detail-session-id', record.sessionId);
  setDetailPage(1);
  showDetailState({ empty: false, loading: false, error: false, content: true });
}

function openPanel() {
  const panel = PANEL();
  const shell = SHELL();
  if (!panel || !shell) return;
  panel.hidden = false;
  shell.classList.add('maps-shell--detail-open');
}

function bindTabs() {
  const panel = PANEL();
  if (!panel) return;
  for (const tab of panel.querySelectorAll('.maps-detail-panel__tab')) {
    tab.addEventListener('click', () => {
      const name = tab.getAttribute('data-tab');
      for (const t of panel.querySelectorAll('.maps-detail-panel__tab')) {
        const active = t.getAttribute('data-tab') === name;
        t.classList.toggle('maps-detail-panel__tab--active', active);
        t.setAttribute('aria-selected', active ? 'true' : 'false');
      }
      for (const pane of panel.querySelectorAll('.maps-detail-panel__pane')) {
        pane.hidden = pane.getAttribute('data-pane') !== name;
      }
    });
  }
  el('maps-detail-page-prev')?.addEventListener('click', () => setDetailPage(1));
  el('maps-detail-page-next')?.addEventListener('click', () => setDetailPage(2));
}

export function initFlightDetailPanel() {
  bindTabs();
  showDetailState({ empty: true, loading: false, error: false, content: false });
}

export function getSelectedFlightId() {
  return selectedFlightId;
}

export async function loadFlightDetail(flightId, fetchJSON) {
  if (!flightId) return;
  selectedFlightId = flightId;
  openPanel();
  const panel = PANEL();
  if (panel) {
    for (const tab of panel.querySelectorAll('.maps-detail-panel__tab')) {
      const active = tab.getAttribute('data-tab') === 'info';
      tab.classList.toggle('maps-detail-panel__tab--active', active);
      tab.setAttribute('aria-selected', active ? 'true' : 'false');
    }
    for (const pane of panel.querySelectorAll('.maps-detail-panel__pane')) {
      pane.hidden = pane.getAttribute('data-pane') !== 'info';
    }
  }
  showDetailState({ empty: false, loading: true, error: false, content: false });
  try {
    const body = await fetchJSON(`/api/v1/game/flight/active/${encodeURIComponent(flightId)}`);
    renderFlightRecord(body?.data ?? {});
  } catch (err) {
    showDetailState({ empty: false, loading: false, error: true, content: false });
    const errorEl = el('maps-detail-error');
    if (errorEl) {
      errorEl.textContent = err.status === 404
        ? 'This flight is no longer active.'
        : (err.message || 'Failed to load flight details.');
    }
  }
}
