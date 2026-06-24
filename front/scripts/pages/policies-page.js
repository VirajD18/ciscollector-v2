import { getPolicies } from '../api/services/policies.js';

let POLICY_CHECKS = [];
let POLICY_TEMPLATES = [];
let POLICY_DEFINITIONS = [];
let POLICY_GROUPS = [];
let POLICY_HOST_MAP = [];
let policySelectedChecks = new Set();
let policyActiveTemplate = 'prod_strict';
let policiesBound = false;

function renderPolicyChecks() {
  const grid = document.getElementById('policy-check-grid');
  if (!grid) return;
  grid.innerHTML = POLICY_CHECKS.map(c => {
    const on = policySelectedChecks.has(c.id);
    return '<label class="policy-check-item"><input type="checkbox" data-check="' + c.id + '" ' +
      (on ? 'checked' : '') + ' /><span><strong>' + c.label + '</strong><br><code>' + c.cmd +
      '</code> · menu ' + c.menu + '</span></label>';
  }).join('');
  grid.querySelectorAll('input[data-check]').forEach(cb => {
    cb.addEventListener('change', () => {
      if (cb.checked) policySelectedChecks.add(cb.dataset.check);
      else policySelectedChecks.delete(cb.dataset.check);
    });
  });
}

function renderPolicyTemplates() {
  const grid = document.getElementById('policy-template-grid');
  if (!grid) return;
  grid.innerHTML = POLICY_TEMPLATES.map(t => {
    const sel = t.id === policyActiveTemplate ? ' selected' : '';
    return '<div class="policy-template-card' + sel + '" data-template="' + t.id + '"><h4>' + t.name +
      '</h4><p>' + t.desc + '</p><p style="margin-top:8px;font-size:11px;color:var(--muted);">' +
      t.checks.length + ' checks</p></div>';
  }).join('');
  grid.querySelectorAll('[data-template]').forEach(card => {
    card.addEventListener('click', () => {
      policyActiveTemplate = card.dataset.template;
      const t = POLICY_TEMPLATES.find(x => x.id === policyActiveTemplate);
      if (t) {
        policySelectedChecks = new Set(t.checks);
        const nameEl = document.getElementById('policy-selected-name');
        if (nameEl) nameEl.textContent = t.name;
        renderPolicyChecks();
        renderPolicyTemplates();
      }
    });
  });
}

function fillPolicySelects() {
  const hosts = window.__SHIELD_BOOT__?.hosts || [];
  const opts = POLICY_DEFINITIONS.map(p => '<option value="' + p.id + '">' + p.name + '</option>').join('');
  ['policy-new-group-policy', 'policy-host-policy', 'email-policy-bundle'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.innerHTML = opts;
  });
  const sched = document.getElementById('policy-sched-target');
  if (sched) {
    sched.innerHTML = POLICY_GROUPS.map(g => '<option value="group:' + g.name + '">Group: ' + g.name + '</option>').join('') +
      POLICY_DEFINITIONS.map(p => '<option value="policy:' + p.id + '">Policy: ' + p.name + '</option>').join('');
  }
  const hostPick = document.getElementById('policy-host-pick');
  if (hostPick) {
    hostPick.innerHTML = hosts.map(h => '<option value="' + h[0] + '">' + h[0] + '</option>').join('');
  }
}

function renderPolicyGroups() {
  const tbody = document.getElementById('policy-groups-tbody');
  if (!tbody) return;
  tbody.innerHTML = POLICY_GROUPS.map(g =>
    '<tr><td><strong>' + g.name + '</strong></td><td>' + (g.hosts?.length || g.hosts || 0) +
    '</td><td><code>' + (g.policy || 'prod_critical_policy') + '</code></td><td><code>' +
    (g.schedule || 'from kshieldconfig.toml') + '</code></td><td>—</td></tr>'
  ).join('');
}

function renderPolicyHosts() {
  const tbody = document.getElementById('policy-hosts-tbody');
  if (!tbody) return;
  tbody.innerHTML = POLICY_HOST_MAP.map(h =>
    '<tr><td><strong>' + h.host + '</strong></td><td>' + (h.group || 'production') + '</td><td><code>' +
    h.policy + '</code></td><td><span class="badge badge-muted">' + h.checks + '</span></td><td>—</td></tr>'
  ).join('');
}

function mapPoliciesApi(data) {
  POLICY_CHECKS = (data.checks || []).map(c => ({
    id: c.id, label: c.label, cmd: c.cmd, menu: c.menu,
  }));
  POLICY_TEMPLATES = (data.templates || []).map(t => ({
    id: t.id, name: t.name, desc: t.desc, checks: t.checks || [],
  }));
  POLICY_DEFINITIONS = (data.definitions || []).map(p => ({
    id: p.id, name: p.name, checks: p.checks || [],
  }));
  POLICY_GROUPS = (data.groups || []).map(g => ({
    name: g.name, hosts: g.hosts || [], policy: 'prod_critical_policy', schedule: '',
  }));
  POLICY_HOST_MAP = (data.host_map || []).map(h => ({
    host: h.host, group: 'production', policy: h.policy, checks: h.checks, override: false,
  }));
  if (POLICY_TEMPLATES.length) {
    policyActiveTemplate = POLICY_TEMPLATES[0].id;
    policySelectedChecks = new Set(POLICY_TEMPLATES[0].checks);
  }
}

export async function initPoliciesPage() {
  const root = document.getElementById('page-policies');
  if (!root) return;
  try {
    mapPoliciesApi(await getPolicies());
  } catch (err) {
    const callout = root.querySelector('.callout');
    if (callout) {
      callout.innerHTML += '<p style="color:var(--danger);margin-top:8px;">Failed to load policies: ' + err.message + '</p>';
    }
    return;
  }
  renderPolicyTemplates();
  renderPolicyChecks();
  fillPolicySelects();
  renderPolicyGroups();
  renderPolicyHosts();

  if (!policiesBound) {
    policiesBound = true;
    document.querySelectorAll('.policy-tab').forEach(tab => {
      tab.addEventListener('click', () => {
        document.querySelectorAll('.policy-tab').forEach(t => t.classList.remove('active'));
        document.querySelectorAll('.policy-section').forEach(s => s.classList.remove('active'));
        tab.classList.add('active');
        const sec = document.getElementById('policy-sec-' + tab.dataset.policyTab);
        if (sec) sec.classList.add('active');
      });
    });
  }
}
