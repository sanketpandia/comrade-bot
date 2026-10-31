document.addEventListener('click', async (event) => {
  const key = event.target.closest('.copy-key[data-copy]');
  if (!key) return;
  try {
    await navigator.clipboard.writeText(key.dataset.copy);
    key.classList.add('copy-key--copied');
    key.classList.remove('copy-key--failed');
    window.setTimeout(() => key.classList.remove('copy-key--copied'), 1200);
  } catch {
    key.classList.add('copy-key--failed');
    key.classList.remove('copy-key--copied');
    window.setTimeout(() => key.classList.remove('copy-key--failed'), 1200);
  }
});
