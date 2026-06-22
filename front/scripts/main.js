import { initApp } from './app/init.js';

initApp().catch((err) => {
  console.error('Dashboard failed to start:', err);
  const root = document.getElementById('app-root');
  if (root) {
    root.innerHTML =
      '<div style="padding:2rem;color:#e85d75;font-family:sans-serif">' +
      '<h1>KloudDB Shield</h1><p>Failed to load dashboard. Serve via main-server (port 8081), not file://</p>' +
      `<pre style="color:#a8b0c4;margin-top:1rem">${err.message}</pre></div>`;
  }
});
