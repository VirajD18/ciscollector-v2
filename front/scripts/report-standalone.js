import { mountHtmlReportViewer } from './pages/html-report-viewer.js';

const params = new URLSearchParams(location.search);
const host = params.get('host') || '';
const root = document.getElementById('html-report-root');

if (!root) {
  throw new Error('#html-report-root not found');
}

if (!host) {
  root.innerHTML =
    '<div class="kshield-html-report"><div class="kshield-loading">' +
    'No host specified. Open from the dashboard or use <code>?host=localhost:5432</code></div></div>';
} else {
  document.title = host + ' — KloudDBShield Report';
  void mountHtmlReportViewer(root, host);
}
