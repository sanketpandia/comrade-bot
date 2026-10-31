const STORAGE_KEY = 'maps-chrome-collapse';

function urlHasChromeFlags() {
  const params = new URLSearchParams(window.location.search);
  if (params.has('mode')) return true;
  return ['header', 'filters', 'detail'].some((key) => params.has(key));
}

function readStoredCollapse() {
  try {
    return JSON.parse(localStorage.getItem(STORAGE_KEY) || 'null');
  } catch {
    return null;
  }
}

function writeStoredCollapse(state) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    /* ignore */
  }
}

function applyBodyHeaderHidden(shell) {
  const hide = shell.classList.contains('maps-shell--map-only');
  document.body.classList.toggle('maps-page--chrome-hidden', hide);
}

export function onChromeTransitionEnd(callback) {
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  if (reduced) {
    callback();
    return;
  }
  window.setTimeout(callback, 220);
}

function setFiltersExpanded(shell, filtersToggle, expanded) {
  shell.classList.toggle('maps-shell--filters-collapsed', !expanded);
  if (!filtersToggle) return;
  filtersToggle.setAttribute('aria-expanded', expanded ? 'true' : 'false');
  filtersToggle.setAttribute('aria-label', 'Expand filters');
}

export function initMapsChrome({ shell, map, getMapPadding }) {
  if (!shell) return;

  const filtersToggle = document.getElementById('maps-filters-toggle');
  const filtersCollapse = document.getElementById('maps-filters-collapse');
  const mapOnlyBtn = document.getElementById('maps-map-only');

  function invalidate() {
    onChromeTransitionEnd(() => {
      map.invalidateSize();
      if (typeof getMapPadding === 'function') {
        map.setFitBoundsPadding(getMapPadding());
      }
    });
  }

  function persistFiltersCollapsed(collapsed) {
    if (!urlHasChromeFlags()) {
      const stored = readStoredCollapse() || {};
      writeStoredCollapse({ ...stored, filtersCollapsed: collapsed });
    }
  }

  filtersToggle?.addEventListener('click', () => {
    if (!shell.classList.contains('maps-shell--filters-collapsed')) return;
    setFiltersExpanded(shell, filtersToggle, true);
    persistFiltersCollapsed(false);
    invalidate();
  });

  filtersCollapse?.addEventListener('click', () => {
    setFiltersExpanded(shell, filtersToggle, false);
    persistFiltersCollapsed(true);
    invalidate();
  });

  mapOnlyBtn?.addEventListener('click', () => {
    const hidden = shell.classList.toggle('maps-shell--map-only');
    mapOnlyBtn.setAttribute('aria-pressed', hidden ? 'true' : 'false');
    applyBodyHeaderHidden(shell);
    if (!urlHasChromeFlags()) {
      writeStoredCollapse({ mapOnly: hidden });
    }
    invalidate();
  });

  if (!urlHasChromeFlags()) {
    const stored = readStoredCollapse();
    if (stored?.filtersCollapsed) {
      setFiltersExpanded(shell, filtersToggle, false);
    }
    if (stored?.mapOnly) {
      shell.classList.add('maps-shell--map-only');
      mapOnlyBtn?.setAttribute('aria-pressed', 'true');
      applyBodyHeaderHidden(shell);
    }
  }

  applyBodyHeaderHidden(shell);
}
