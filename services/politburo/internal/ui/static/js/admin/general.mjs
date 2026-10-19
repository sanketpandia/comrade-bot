const codeInput = document.getElementById('va-code');
const statusEl = document.getElementById('admin-general-status');

function showStatus(message, isError) {
  if (!statusEl) {
    return;
  }
  statusEl.textContent = message;
  statusEl.hidden = !message;
  statusEl.classList.toggle('admin-panel__status--error', Boolean(isError));
}

async function loadVADetails() {
  try {
    const response = await fetch('/api/v1/va', { credentials: 'same-origin' });
    const body = await response.json().catch(() => ({}));
    if (!response.ok) {
      const message = body?.error?.message || `Request failed (${response.status})`;
      showStatus(message, true);
      if (codeInput) {
        codeInput.value = '';
        codeInput.placeholder = '—';
      }
      return;
    }
    const code = body?.data?.code ?? '';
    if (codeInput) {
      codeInput.value = code;
      codeInput.placeholder = '';
    }
    showStatus('', false);
  } catch {
    showStatus('Could not load VA details.', true);
    if (codeInput) {
      codeInput.value = '';
      codeInput.placeholder = '—';
    }
  }
}

loadVADetails();
