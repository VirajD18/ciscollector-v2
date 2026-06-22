/**
 * Splits dba-console-prototype.html into modular front/ assets.
 * Run from repo root: node front/scripts/extract-prototype.mjs
 */
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, '../..');
const PROTO = path.join(ROOT, 'dba-console-prototype.html');
const FRONT = path.join(ROOT, 'front');

const html = fs.readFileSync(PROTO, 'utf8');

function extractBetween(source, startMarker, endMarker) {
  const start = source.indexOf(startMarker);
  if (start === -1) throw new Error(`Missing: ${startMarker}`);
  const end = source.indexOf(endMarker, start + startMarker.length);
  if (end === -1) throw new Error(`Missing end after: ${startMarker}`);
  return source.slice(start + startMarker.length, end).trim();
}

function stripIife(scriptBlock) {
  let appJs = scriptBlock
    .replace(/^\(function\s*\(\)\s*\{/, '')
    .replace(/'use strict';/, '')
    .trim();
  if (appJs.endsWith('})();')) appJs = appJs.slice(0, -5).trim();
  if (appJs.endsWith('})()')) appJs = appJs.slice(0, -4).trim();
  return appJs;
}

function patchAppJs(appJs) {
  appJs = appJs.replace(
    /^function gid\(id\)[\s\S]*?function gq\(sel\)[\s\S]*?\n\n/m,
    '',
  );

  const pagesMatch = appJs.match(/const pages = (\{[\s\S]*?\n    \});/);
  const pagesObj = pagesMatch ? pagesMatch[1] : null;
  if (pagesMatch) {
    appJs = appJs.replace(/const pages = \{[\s\S]*?\n    \};\n\n/, '');
    writeRoutesFromPages(pagesObj);
  }

  appJs = appJs.replace(/\bconst hosts = \[/, 'let hosts = [');
  appJs = appJs.replace(/\bconst FLEET_CATEGORIES = \[/, 'let FLEET_CATEGORIES = [');

  const boot =
    '    const boot = window.__SHIELD_BOOT__;\n' +
    '    if (boot?.hosts?.length) hosts = boot.hosts;\n' +
    '    if (boot?.fleetCategories?.length) FLEET_CATEGORIES = boot.fleetCategories;\n\n';

  if (!appJs.includes('window.__SHIELD_BOOT__')) {
    appJs = appJs.replace(
      /(\n    document\.querySelectorAll\('\.nav-item'\)\.forEach)/,
      `\n${boot}$1`,
    );
  }

  const header =
    "import { gid, gval, gon, gq } from '../utils/dom.js';\n" +
    "import { PAGE_META as pages } from '../router/routes.js';\n\n";

  return header + appJs;
}

function writeRoutesFromPages(pagesObjLiteral) {
  const ids = [...html.matchAll(/<section\s+id="page-([^"]+)"/g)].map((m) => m[1]);
  const uniqueIds = [...new Set(ids)];

  const meta = pagesObjLiteral
    .split('\n')
    .map((line) => line.replace(/^ {4}/, '  '))
    .join('\n')
    .replace(/\n\s*\};?\s*$/, '\n}');

  const routesPath = path.join(FRONT, 'scripts/router/routes.js');
  const content =
    '/** Auto-synced from dba-console-prototype.html — re-run extract-prototype.mjs after edits */\n' +
    "export const DEFAULT_PAGE = 'strategic-dashboard';\n\n" +
    `export const PAGE_IDS = ${JSON.stringify(uniqueIds, null, 2)};\n\n` +
    `export const PAGE_META = ${meta};\n`;

  fs.writeFileSync(routesPath, content);
}

const styleBlock = extractBetween(html, '<style>', '</style>');
const bodyInner = extractBetween(html, '<body>', '</body>');
const scriptBlock = extractBetween(html, '<script>', '</script>');

const sidebar = extractBetween(bodyInner, '<aside class="sidebar">', '</aside>');
const topbar = extractBetween(bodyInner, '<header class="topbar">', '</header>');
const flowBanner = extractBetween(bodyInner, '<footer class="flow-banner">', '</footer>');
const contentBlock = extractBetween(bodyInner, '<main class="content">', '</main>');

const pageRegex = /<section\s+id="page-([^"]+)"[^>]*>[\s\S]*?<\/section>/g;
const pages = [];
let m;
while ((m = pageRegex.exec(contentBlock)) !== null) {
  pages.push({ id: m[1], html: m[0].trim() });
}

const dirs = [
  'components',
  'pages',
  'styles',
  'scripts/app',
  'scripts/router',
  'scripts/utils',
  'scripts/charts',
  'scripts/api/services',
  'mock-data',
  'layouts',
  'assets',
  'vendor',
  'utilities',
];

for (const d of dirs) {
  fs.mkdirSync(path.join(FRONT, d), { recursive: true });
}

const tokensEnd = styleBlock.indexOf('* { box-sizing');
const tokensCss = tokensEnd > 0 ? styleBlock.slice(0, tokensEnd).trim() : ':root {}';
const mainCss = tokensEnd > 0 ? styleBlock.slice(tokensEnd).trim() : styleBlock;

fs.writeFileSync(path.join(FRONT, 'styles/tokens.css'), tokensCss + '\n');
fs.writeFileSync(path.join(FRONT, 'styles/main.css'), mainCss + '\n');

fs.writeFileSync(
  path.join(FRONT, 'components/sidebar.html'),
  `<aside class="sidebar">\n${sidebar}\n</aside>\n`,
);
fs.writeFileSync(
  path.join(FRONT, 'components/topbar.html'),
  `<header class="topbar">\n${topbar}\n</header>\n`,
);
fs.writeFileSync(
  path.join(FRONT, 'components/flow-banner.html'),
  `<footer class="flow-banner">\n${flowBanner}\n</footer>\n`,
);

const pageIds = new Set(pages.map((p) => p.id));
for (const file of fs.readdirSync(path.join(FRONT, 'pages'))) {
  if (!file.endsWith('.html')) continue;
  const id = file.replace(/\.html$/, '');
  if (!pageIds.has(id)) {
    fs.unlinkSync(path.join(FRONT, 'pages', file));
    console.log(`Removed obsolete page: ${file}`);
  }
}

for (const p of pages) {
  fs.writeFileSync(path.join(FRONT, 'pages', `${p.id}.html`), p.html + '\n');
}

const patchedApp = patchAppJs(stripIife(scriptBlock));
fs.writeFileSync(path.join(FRONT, 'scripts/app/prototype-app.js'), patchedApp + '\n');

console.log(`Extracted ${pages.length} pages, CSS, components, routes, and app script.`);
