package grid

import (
	"bytes"
	"fmt"
	"html/template"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

// MultiHTMLData holds template data for a multi-adapter HTML export.
type MultiHTMLData struct {
	Title           string
	GeneratedAt     string
	Duration        string
	RefreshInterval int
	TotalInterfaces int
	TotalHosts      int
	TotalOnline     int
	TotalFast       int
	TotalSlow       int
	TotalSilent     int
	TotalOffline    int
	Interfaces      []HTMLData
}

const multiHtmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
{{- if gt .RefreshInterval 0 }}
<meta http-equiv="refresh" content="{{ .RefreshInterval }}">
{{- end }}
<title>{{ .Title }}</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background-color: #1a1a1a;
    color: #e0e0e0;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 24px;
    gap: 20px;
  }
  .main-container {
    width: 100%;
    max-width: 960px;
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  .top-card {
    background-color: #222222;
    border: 1px solid #333333;
    border-radius: 8px;
    padding: 20px 24px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 16px;
    border-bottom: 1px solid #333333;
    padding-bottom: 14px;
  }
  h1 {
    font-size: 1.35rem;
    font-weight: 700;
    color: #ffffff;
    letter-spacing: -0.02em;
  }
  .header-right {
    display: flex;
    align-items: center;
    gap: 16px;
    flex-wrap: wrap;
  }
  .header-actions {
    display: flex;
    gap: 8px;
    align-items: center;
    flex-wrap: wrap;
  }
  .action-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: #2b303c;
    border: 1px solid #3e4451;
    color: #abb2bf;
    font-size: 0.78rem;
    font-weight: 600;
    padding: 5px 11px;
    border-radius: 5px;
    cursor: pointer;
    transition: all 0.15s ease;
    user-select: none;
  }
  .action-btn:hover {
    background: #353b45;
    color: #ffffff;
    border-color: #5c6370;
  }
  .action-btn.copied {
    background: rgba(82, 196, 26, 0.2);
    border-color: #52c41a;
    color: #52c41a;
  }
  .meta {
    font-size: 0.76rem;
    color: #888888;
  }
  .tabs-bar {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  .tab-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: #25282f;
    border: 1px solid #383e4a;
    color: #abb2bf;
    padding: 6px 14px;
    border-radius: 20px;
    font-size: 0.82rem;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .tab-btn:hover {
    background: #2e3440;
    color: #eceff4;
    border-color: #4c566a;
  }
  .tab-btn.active {
    background: #38bdf8;
    color: #0f172a;
    border-color: #38bdf8;
    font-weight: 700;
    box-shadow: 0 0 10px rgba(56, 189, 248, 0.35);
  }
  .iface-card {
    background-color: #222222;
    border: 1px solid #333333;
    border-radius: 8px;
    padding: 24px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 18px;
    transition: all 0.2s ease;
  }
  .iface-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    padding-bottom: 12px;
    border-bottom: 1px solid #333333;
  }
  .iface-title {
    font-size: 1.15rem;
    font-weight: 600;
    color: #ffffff;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .primary-badge {
    background: rgba(56, 189, 248, 0.2);
    border: 1px solid #38bdf8;
    color: #38bdf8;
    font-size: 0.7rem;
    font-weight: 600;
    padding: 2px 7px;
    border-radius: 4px;
  }
  .hud-card {
    display: flex;
    flex-direction: column;
    gap: 8px;
    width: 100%;
    background: rgba(255, 255, 255, 0.03);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 6px;
    padding: 10px 14px;
  }
  .hud-adapter-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.88rem;
    font-weight: 600;
    color: #f1f5f9;
  }
  .hud-badges {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }
  .hud-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 3px 8px;
    border-radius: 6px;
    font-size: 0.75rem;
    font-weight: 500;
    border: 1px solid transparent;
  }
  .hud-badge-speed {
    background: rgba(56, 189, 248, 0.15);
    border-color: rgba(56, 189, 248, 0.35);
    color: #38bdf8;
  }
  .hud-badge-mtu {
    background: rgba(168, 85, 247, 0.15);
    border-color: rgba(168, 85, 247, 0.35);
    color: #c084fc;
  }
  .hud-badge-dhcp {
    background: rgba(74, 222, 128, 0.15);
    border-color: rgba(74, 222, 128, 0.35);
    color: #4ade80;
  }
  .hud-badge-static {
    background: rgba(148, 163, 184, 0.15);
    border-color: rgba(148, 163, 184, 0.35);
    color: #94a3b8;
  }
  .hud-badge-wifi {
    background: rgba(251, 191, 36, 0.15);
    border-color: rgba(251, 191, 36, 0.35);
    color: #fbbf24;
  }
  .stats-bar {
    display: flex;
    gap: 8px;
    font-size: 0.82rem;
    flex-wrap: wrap;
    align-items: center;
    width: 100%;
  }
  .filter-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: #25282f;
    border: 1px solid #383e4a;
    color: #abb2bf;
    padding: 3px 10px;
    border-radius: 12px;
    font-size: 0.76rem;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .filter-pill:hover {
    background: #2e3440;
    color: #eceff4;
    border-color: #4c566a;
  }
  .filter-pill.active {
    background: #3b4252;
    color: #ffffff;
    border-color: #61afef;
    box-shadow: 0 0 6px rgba(97, 175, 239, 0.3);
    font-weight: 600;
  }
  .stat-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    display: inline-block;
  }
  .dot-online { background-color: #4d86a2; }
  .dot-fast { background-color: #e4f9d4; }
  .dot-slow { background-color: #ab7550; }
  .dot-offline { background-color: #404e41; }
  .grid-canvas {
    background-color: #2c2c2c;
    border: 1px solid #333333;
    padding: 3px;
    border-radius: 4px;
    display: inline-block;
    transition: transform 0.2s ease;
  }
  .grid-table {
    display: grid;
    gap: 1px;
    background-color: #2c2c2c;
  }
  .cell {
    width: 8px;
    height: 8px;
    border-radius: 1px;
    cursor: pointer;
    transition: transform 0.1s ease, filter 0.15s ease;
  }
  .cell:hover {
    transform: scale(1.6);
    z-index: 10;
    outline: 2px solid #ffffff;
    outline-offset: 1px;
  }
  .cell-me {
    outline: 2px solid #38bdf8 !important;
    outline-offset: 1px;
    z-index: 5;
  }
  .cell-silent {
    border: 1px dashed #f59e0b;
    box-shadow: 0 0 3px #f59e0b;
  }
  .cell-empty {
    visibility: hidden;
    pointer-events: none;
  }
  .cell-dimmed {
    opacity: 0.15;
    filter: grayscale(80%);
  }
  .hud {
    font-size: 0.8rem;
    color: #888888;
    height: 22px;
    line-height: 22px;
    text-align: center;
    width: 100%;
    background: #1e1e1e;
    border-radius: 4px;
    border: 1px solid #2a2a2a;
    padding: 0 8px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .scale-controls {
    display: flex;
    gap: 6px;
    align-items: center;
    font-size: 0.75rem;
    color: #777777;
  }
  .scale-btn {
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    color: #cccccc;
    padding: 2px 8px;
    border-radius: 3px;
    cursor: pointer;
    font-size: 0.72rem;
  }
  .scale-btn:hover {
    background: #3a3a3a;
    color: #ffffff;
  }
  .deltas-card {
    width: 100%;
    background: #1c1d22;
    border: 1px solid #2d3139;
    border-radius: 6px;
    padding: 10px 14px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  .deltas-title {
    font-size: 0.8rem;
    font-weight: 600;
    color: #abb2bf;
    display: flex;
    justify-content: space-between;
  }
  .deltas-list {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 120px;
    overflow-y: auto;
  }
  .delta-row {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 0.75rem;
  }
  .delta-badge-inline {
    padding: 1px 6px;
    border-radius: 3px;
    font-weight: 600;
    font-size: 0.68rem;
    text-transform: uppercase;
  }
  .delta-badge-joined { background: rgba(74, 222, 128, 0.2); color: #4ade80; }
  .delta-badge-dropped { background: rgba(248, 113, 113, 0.2); color: #f87171; }
  .delta-badge-changed { background: rgba(251, 191, 36, 0.2); color: #fbbf24; }

  /* Light Theme */
  body.light-theme {
    background-color: #f3f4f6;
    color: #1f2937;
  }
  body.light-theme .top-card,
  body.light-theme .iface-card {
    background-color: #ffffff;
    border-color: #e5e7eb;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.08);
  }
  body.light-theme h1,
  body.light-theme .iface-title { color: #111827; }
  body.light-theme header,
  body.light-theme .iface-header { border-bottom-color: #e5e7eb; }
  body.light-theme .meta { color: #6b7280; }
  body.light-theme .action-btn,
  body.light-theme .tab-btn {
    background: #f3f4f6;
    border-color: #d1d5db;
    color: #374151;
  }
  body.light-theme .tab-btn.active {
    background: #38bdf8;
    color: #0f172a;
    border-color: #38bdf8;
  }
  body.light-theme .filter-pill {
    background: #f3f4f6;
    border-color: #d1d5db;
    color: #4b5563;
  }
  body.light-theme .filter-pill.active {
    background: #e0e7ff;
    color: #3730a3;
    border-color: #818cf8;
  }
  body.light-theme .hud {
    background: #f9fafb;
    border-color: #e5e7eb;
    color: #374151;
  }
</style>
</head>

<body>
<div class="main-container">
  <div class="top-card">
    <header>
      <h1>PingGrid Multi-Adapter Monitor</h1>
      <div class="header-right">
        <div class="header-actions">
          <button id="refresh-btn" class="action-btn" onclick="location.reload()" title="Reload sweep dashboard">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg>
            <span>Refresh</span>
          </button>
          <button id="copy-btn" class="action-btn" onclick="copyAllActiveIPs()" title="Copy active responding IP addresses across all interfaces">
            <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
            <span id="copy-text">Copy All Active ({{ .TotalOnline }})</span>
          </button>
          <button id="theme-btn" class="action-btn" onclick="toggleTheme()" title="Toggle Dark/Light theme">
            <span>🌓 Theme</span>
          </button>
        </div>
        <div class="meta">{{ .TotalInterfaces }} Interfaces &bull; {{ .TotalOnline }}/{{ .TotalHosts }} Active &bull; {{ .GeneratedAt }} &bull; {{ .Duration }}</div>
      </div>
    </header>

    <div class="tabs-bar">
      <button class="tab-btn active" onclick="switchIfaceTab('all', this)">All Interfaces (Stacked)</button>
      {{- range $i, $iface := .Interfaces }}
      <button class="tab-btn" onclick="switchIfaceTab('iface-{{ $i }}', this)">{{ $iface.InterfaceName }} ({{ $iface.Online }}/{{ $iface.Total }})</button>
      {{- end }}
    </div>
  </div>

  {{- range $i, $iface := .Interfaces }}
  <div id="iface-card-{{ $i }}" class="iface-card">
    <div class="iface-header">
      <div class="iface-title">
        <span>{{ $iface.InterfaceName }}</span>
        {{- if $iface.LinkHealth }}
        {{- if $iface.LinkHealth.IsWireless }}<span class="hud-badge hud-badge-wifi">Wi-Fi</span>{{ end }}
        {{- end }}
        <span style="font-size: 0.8rem; color: #888888;">{{ $iface.Title }}</span>
      </div>
      <div class="meta">{{ $iface.Online }}/{{ $iface.Total }} Active &bull; {{ $iface.Duration }}</div>
    </div>

    {{- if $iface.LinkHealth }}
    <div class="hud-card">
      <div class="hud-adapter-title">
        <span>{{ if $iface.LinkHealth.IsWireless }}📶{{ else }}🔌{{ end }}</span>
        <span>{{ if $iface.LinkHealth.AdapterModel }}{{ $iface.LinkHealth.AdapterModel }}{{ else }}{{ $iface.InterfaceName }}{{ end }}</span>
      </div>
      <div class="hud-badges">
        {{- if $iface.LinkHealth.LinkSpeedStr }}
        <span class="hud-badge hud-badge-speed">⚡ {{ $iface.LinkHealth.LinkSpeedStr }}{{ if $iface.LinkHealth.Duplex }} {{ $iface.LinkHealth.Duplex }}{{ end }}</span>
        {{- end }}
        {{- if gt $iface.LinkHealth.MTU 0 }}
        <span class="hud-badge hud-badge-mtu">📦 MTU {{ $iface.LinkHealth.MTU }}</span>
        {{- end }}
        {{- if $iface.LinkHealth.DHCPEnabled }}
        <span class="hud-badge hud-badge-dhcp">🟢 DHCP Active</span>
        {{- else }}
        <span class="hud-badge hud-badge-static">⚪ Static IP</span>
        {{- end }}
        {{- if $iface.LinkHealth.IsWireless }}
        <span class="hud-badge hud-badge-wifi">📶 {{ $iface.LinkHealth.SSID }}{{ if $iface.LinkHealth.Band }} &bull; {{ $iface.LinkHealth.Band }}{{ end }}{{ if gt $iface.LinkHealth.SignalPercent 0 }} &bull; {{ $iface.LinkHealth.SignalPercent }}%{{ end }}</span>
        {{- end }}
      </div>
    </div>
    {{- end }}

    <div class="stats-bar">
      <button class="filter-pill active" onclick="setIfaceFilter({{ $i }}, 'all', this)">All: {{ $iface.Total }}</button>
      <button class="filter-pill" onclick="setIfaceFilter({{ $i }}, 'active', this)"><span class="stat-dot dot-online"></span> Active: {{ $iface.Online }}</button>
      <button class="filter-pill" onclick="setIfaceFilter({{ $i }}, 'fast', this)"><span class="stat-dot dot-fast"></span> Fast: {{ $iface.Fast }}</button>
      <button class="filter-pill" onclick="setIfaceFilter({{ $i }}, 'slow', this)"><span class="stat-dot dot-slow"></span> Slow: {{ $iface.Slow }}</button>
      {{- if gt $iface.Silent 0 }}
      <button class="filter-pill" onclick="setIfaceFilter({{ $i }}, 'silent', this)"><span class="stat-dot" style="background:#f59e0b;"></span> Silent: {{ $iface.Silent }}</button>
      {{- end }}
      <button class="filter-pill" onclick="setIfaceFilter({{ $i }}, 'offline', this)"><span class="stat-dot dot-offline"></span> Offline: {{ $iface.Offline }}</button>
    </div>

    <div id="canvas-wrapper-{{ $i }}" class="grid-canvas" style="background-color: {{ $iface.ColorBorder }};">
      <div class="grid-table" style="grid-template-columns: repeat({{ $iface.Cols }}, 8px); background-color: {{ $iface.ColorBorder }};">
        {{- range $iface.Cells }}
        <div class="cell iface-cell-{{ $i }}{{ if .Delta }} cell-{{ .Delta }}{{ end }}{{ if .IsMe }} cell-me{{ end }}{{ if .IsSilent }} cell-silent{{ end }}{{ if .IsEmpty }} cell-empty{{ end }}"
             style="background-color: {{ .Color }};"
             data-ip="{{ .IP }}" data-status="{{ .Status }}" data-rtt="{{ .RTT }}" data-delta="{{ .Delta }}" data-role="{{ .Role }}" data-mac="{{ .MAC }}" data-vendor="{{ .Vendor }}"
             title="{{ .Tooltip }}"
             {{ if not .IsEmpty }}onmouseover="updateIfaceHud({{ $i }}, '{{ .IP }}', '{{ .Status }}', '{{ .RTT }}', '{{ .Delta }}', '{{ .Role }}', '{{ .MAC }}', '{{ .Vendor }}')"
             onmouseout="resetIfaceHud({{ $i }})"{{ end }}></div>
        {{- end }}
      </div>
    </div>

    <div id="hud-{{ $i }}" class="hud">Hover over any grid cell to view host details</div>

    <div class="scale-controls">
      <span>Zoom:</span>
      <button class="scale-btn" onclick="setIfaceScale({{ $i }}, 1)">1x</button>
      <button class="scale-btn" onclick="setIfaceScale({{ $i }}, 2)">2x</button>
      <button class="scale-btn" onclick="setIfaceScale({{ $i }}, 3)">3x</button>
    </div>

    {{- if $iface.Deltas }}
    <div class="deltas-card">
      <div class="deltas-title">
        <span>Recent State Changes ({{ len $iface.Deltas }})</span>
      </div>
      <div class="deltas-list">
        {{- range $iface.Deltas }}
        <div class="delta-row">
          <span class="delta-badge-inline delta-badge-{{ .Kind }}">{{ .Kind }}</span>
          <span class="delta-ip">{{ .IP }}</span>
          <span class="delta-detail">
            {{- if eq .Kind "joined" }}came online ({{ .New }}, {{ .RTT }}){{ end }}
            {{- if eq .Kind "dropped" }}went offline (previously {{ .Old }}){{ end }}
            {{- if eq .Kind "changed" }}performance shifted: {{ .Old }} &rarr; {{ .New }} ({{ .RTT }}){{ end }}
          </span>
        </div>
        {{- end }}
      </div>
    </div>
    {{- end }}
  </div>
  {{- end }}
</div>

<script>
  function switchIfaceTab(target, btn) {
    document.querySelectorAll('.tab-btn').forEach(function(b) { b.classList.remove('active'); });
    btn.classList.add('active');
    var cards = document.querySelectorAll('.iface-card');
    cards.forEach(function(card, idx) {
      if (target === 'all' || card.id === 'iface-card-' + target.replace('iface-', '')) {
        card.style.display = 'flex';
      } else {
        card.style.display = 'none';
      }
    });
  }

  function updateIfaceHud(idx, ip, status, rtt, delta, role, mac, vendor) {
    var hud = document.getElementById('hud-' + idx);
    if (!hud) return;
    var roleBadge = '';
    if (role) roleBadge = ' &bull; <span style="color:#38bdf8;font-weight:bold">' + role + '</span>';
    var hwInfo = '';
    if (mac) {
      hwInfo = ' &bull; <span style="color:#94a3b8;font-family:monospace">' + mac + '</span>';
      if (vendor) hwInfo += ' (' + vendor + ')';
    }
    var deltaBadge = '';
    if (delta === 'joined') deltaBadge = ' &bull; <span style="color:#4ade80">[Joined]</span>';
    else if (delta === 'dropped') deltaBadge = ' &bull; <span style="color:#f87171">[Dropped]</span>';
    else if (delta === 'changed') deltaBadge = ' &bull; <span style="color:#fbbf24">[Changed]</span>';
    hud.innerHTML = '<strong>' + ip + '</strong>' + roleBadge + ' &bull; ' + status + ' &bull; ' + rtt + hwInfo + deltaBadge;
  }

  function resetIfaceHud(idx) {
    var hud = document.getElementById('hud-' + idx);
    if (!hud) return;
    hud.innerHTML = 'Hover over any grid cell to view host details';
  }

  function setIfaceScale(idx, factor) {
    var wrapper = document.getElementById('canvas-wrapper-' + idx);
    if (!wrapper) return;
    wrapper.style.transform = 'scale(' + factor + ')';
    wrapper.style.transformOrigin = 'center center';
    wrapper.style.margin = ((factor - 1) * 38) + 'px 0';
  }

  function setIfaceFilter(idx, filterType, btn) {
    var card = document.getElementById('iface-card-' + idx);
    if (!card) return;
    var pills = card.querySelectorAll('.filter-pill');
    pills.forEach(function(p) { p.classList.remove('active'); });
    btn.classList.add('active');

    var cells = card.querySelectorAll('.iface-cell-' + idx);
    cells.forEach(function(cell) {
      if (cell.classList.contains('cell-empty')) return;
      var status = cell.getAttribute('data-status') || '';
      var delta = cell.getAttribute('data-delta') || '';
      var isVisible = true;

      switch (filterType) {
        case 'active':
          isVisible = status !== 'Offline';
          break;
        case 'fast':
          isVisible = status.startsWith('Fast');
          break;
        case 'slow':
          isVisible = status === 'Slow';
          break;
        case 'silent':
          isVisible = status.indexOf('Silent') !== -1;
          break;
        case 'offline':
          isVisible = status === 'Offline';
          break;
        case 'deltas':
          isVisible = delta !== '';
          break;
        case 'all':
        default:
          isVisible = true;
          break;
      }

      if (isVisible) {
        cell.classList.remove('cell-dimmed');
      } else {
        cell.classList.add('cell-dimmed');
      }
    });
  }

  function copyAllActiveIPs() {
    var cells = document.querySelectorAll('.cell:not(.cell-empty)');
    var ips = [];
    cells.forEach(function(c) {
      var status = c.getAttribute('data-status') || '';
      if (status !== 'Offline') {
        var ip = c.getAttribute('data-ip');
        if (ip && ips.indexOf(ip) === -1) ips.push(ip);
      }
    });
    if (ips.length === 0) return;
    navigator.clipboard.writeText(ips.join('\n')).then(function() {
      var btn = document.getElementById('copy-btn');
      var txt = document.getElementById('copy-text');
      if (btn && txt) {
        btn.classList.add('copied');
        txt.textContent = 'Copied ' + ips.length + ' IPs!';
        setTimeout(function() {
          btn.classList.remove('copied');
          txt.textContent = 'Copy All Active (' + ips.length + ')';
        }, 2000);
      }
    });
  }

  function toggleTheme() {
    document.body.classList.toggle('light-theme');
  }
</script>
</body>
</html>`

// BuildMultiHTMLData aggregates HTMLData across multiple network interfaces using
// one shared duration. New callers should use BuildMultiHTMLDataWithDurations.
func BuildMultiHTMLData(cfgs []GridConfig, resultsList [][]scanner.HostResult, sweepDuration time.Duration, refreshInterval int, deltasList [][]scanner.HostDelta) MultiHTMLData {
	durations := make([]time.Duration, len(cfgs))
	for i := range durations {
		durations[i] = sweepDuration
	}
	return BuildMultiHTMLDataWithDurations(cfgs, resultsList, durations, refreshInterval, deltasList)
}

// BuildMultiHTMLDataWithDurations preserves each interface's measured sweep duration.
func BuildMultiHTMLDataWithDurations(cfgs []GridConfig, resultsList [][]scanner.HostResult, sweepDurations []time.Duration, refreshInterval int, deltasList [][]scanner.HostDelta) MultiHTMLData {
	var ifaceDatas []HTMLData
	var totalHosts, totalOnline, totalFast, totalSlow, totalSilent, totalOffline int
	var maxDuration time.Duration

	for i := range cfgs {
		var deltas []scanner.HostDelta
		if i < len(deltasList) {
			deltas = deltasList[i]
		}
		var duration time.Duration
		if i < len(sweepDurations) {
			duration = sweepDurations[i]
		}
		if duration > maxDuration {
			maxDuration = duration
		}
		data := buildHTMLData(cfgs[i], resultsList[i], duration, refreshInterval, deltas)
		ifaceDatas = append(ifaceDatas, data)

		totalHosts += data.Total
		totalOnline += data.Online
		totalFast += data.Fast
		totalSlow += data.Slow
		totalSilent += data.Silent
		totalOffline += data.Offline
	}

	return MultiHTMLData{
		Title:           "PingGrid - Multi-Adapter Network Subnet Monitor",
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05 MST"),
		Duration:        scanner.FormatDurationMS(maxDuration),
		RefreshInterval: refreshInterval,
		TotalInterfaces: len(cfgs),
		TotalHosts:      totalHosts,
		TotalOnline:     totalOnline,
		TotalFast:       totalFast,
		TotalSlow:       totalSlow,
		TotalSilent:     totalSilent,
		TotalOffline:    totalOffline,
		Interfaces:      ifaceDatas,
	}
}

// RenderMultiHTML generates a multi-interface interactive HTML dashboard.
func RenderMultiHTML(cfgs []GridConfig, resultsList [][]scanner.HostResult, sweepDuration time.Duration, refreshInterval int, deltasList [][]scanner.HostDelta) ([]byte, error) {
	durations := make([]time.Duration, len(cfgs))
	for i := range durations {
		durations[i] = sweepDuration
	}
	return RenderMultiHTMLWithDurations(cfgs, resultsList, durations, refreshInterval, deltasList)
}

// RenderMultiHTMLWithDurations generates a dashboard with per-interface timings.
func RenderMultiHTMLWithDurations(cfgs []GridConfig, resultsList [][]scanner.HostResult, sweepDurations []time.Duration, refreshInterval int, deltasList [][]scanner.HostDelta) ([]byte, error) {
	tmpl, err := template.New("grid_multi").Parse(multiHtmlTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multi-interface HTML template: %w", err)
	}

	data := BuildMultiHTMLDataWithDurations(cfgs, resultsList, sweepDurations, refreshInterval, deltasList)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render multi-interface HTML template: %w", err)
	}

	return buf.Bytes(), nil
}

const multiMinimalHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
{{- if gt .RefreshInterval 0 }}
<meta http-equiv="refresh" content="{{ .RefreshInterval }}">
{{- end }}
<title>{{ .Title }} (Embed)</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; }
  html, body { width: 100%; min-height: 100%; background: transparent; }
  body { display: flex; flex-direction: column; align-items: center; gap: 8px; overflow: auto; }
  .interface { display: flex; flex-direction: column; align-items: center; gap: 3px; }
  .label { color: #b8b8b8; font: 11px/1.2 system-ui, sans-serif; }
  .canvas { position: relative; overflow: hidden; border: 1px solid #111; }
  .table { display: grid; position: absolute; }
  .cell { width: var(--cell-w); height: var(--cell-h); cursor: default; }
  .cell:hover { outline: 1px solid #fff; z-index: 2; filter: brightness(1.2); }
</style>
</head>
<body>
{{- range .Interfaces }}
  <section class="interface" aria-label="{{ .InterfaceName }}">
    <div class="label">{{ .InterfaceName }}</div>
    <div class="canvas" style="width:{{ .Width }}px;height:{{ .Height }}px;background:{{ .ColorFrame }}">
      <div class="table" style="--cell-w:{{ .CellW }}px;--cell-h:{{ .CellH }}px;grid-template-columns:repeat({{ .Cols }},var(--cell-w));grid-template-rows:repeat({{ .Rows }},var(--cell-h));gap:{{ .BorderW }}px;background:{{ .ColorBorder }};top:{{ .PadTop }}px;left:{{ .PadLeft }}px;padding:{{ .BorderW }}px">
      {{- range .Cells }}
        <div class="cell" style="background:{{ .Color }}" title="{{ .Tooltip }}"></div>
      {{- end }}
      </div>
    </div>
  </section>
{{- end }}
</body>
</html>`

// RenderMultiMinimalHTML generates a button-free stacked view for iframe embedding.
func RenderMultiMinimalHTML(cfgs []GridConfig, resultsList [][]scanner.HostResult, sweepDurations []time.Duration, refreshInterval int, deltasList [][]scanner.HostDelta) ([]byte, error) {
	tmpl, err := template.New("grid_multi_minimal").Parse(multiMinimalHTMLTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multi-interface minimal HTML template: %w", err)
	}
	data := BuildMultiHTMLDataWithDurations(cfgs, resultsList, sweepDurations, refreshInterval, deltasList)
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render multi-interface minimal HTML template: %w", err)
	}
	return buf.Bytes(), nil
}
