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

export function initMapsChrome({ shell, map, getMapPadding }) {
  if (!shell) return;

  const filtersToggle = document.getElementById('maps-filters-toggle');
  const mapOnlyBtn = document.getElementById('maps-map-only');

  function invalidate() {
    onChromeTransitionEnd(() => {
      map.invalidateSize();
      if (typeof getMapPadding === 'function') {
        map.setFitBoundsPadding(getMapPadding());
      }
    });
  }

  filtersToggle?.addEventListener('click', () => {
    shell.classList.remove('maps-shell--filters-collapsed');
    filtersToggle.setAttribute('aria-expanded', 'true');
    if (!urlHasChromeFlags()) {
      const stored = readStoredCollapse() || {};
      writeStoredCollapse({ ...stored, filtersCollapsed: false });
    }
    invalidate();
  });

  document.getElementById('maps-filters-collapse')?.addEventListener('click', () => {
    shell.classList.add('maps-shell--filters-collapsed');
    filtersToggle?.setAttribute('aria-expanded', 'false');
    if (!urlHasChromeFlags()) {
      const stored = readStoredCollapse() || {};
      writeStoredCollapse({ ...stored, filtersCollapsed: true });
    }
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
      shell.classList.add('maps-shell--filters-collapsed');
      filtersToggle?.setAttribute('aria-expanded', 'false');
    }
    if (stored?.mapOnly) {
      shell.classList.add('maps-shell--map-only');
      mapOnlyBtn?.setAttribute('aria-pressed', 'true');
      applyBodyHeaderHidden(shell);
    }
  }

  applyBodyHeaderHidden(shell);
}
