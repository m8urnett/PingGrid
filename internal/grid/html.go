package grid

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/m8urnett/PingGrid/internal/scanner"
)

// HTMLCell represents a single grid cell for HTML template rendering.
type HTMLCell struct {
	Index   int
	IP      string
	Status  string
	Color   string
	RTT     string
	IsUp    bool
	Tooltip string
	Delta   string // "joined", "dropped", "changed", or ""
	IsEmpty bool
}

// HTMLData holds the complete template data for HTML export.
type HTMLData struct {
	Title           string
	GeneratedAt     string
	Rows            int
	Cols            int
	Width           int
	Height          int
	CellW           int
	CellH           int
	BorderW         int
	PadLeft         int
	PadTop          int
	ColorFrame      string
	ColorBorder     string
	ColorOffline    string
	ColorOnline     string
	ColorHigh       string
	ColorSlow       string
	Total           int
	Online          int
	Fast            int
	Slow            int
	Offline         int
	Duration        string
	RefreshInterval int
	JoinedCount     int
	DroppedCount    int
	ChangedCount    int
	Deltas          []scanner.HostDelta
	Cells           []HTMLCell
	ScanMode        string
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
{{- if gt .RefreshInterval 0 }}
<meta http-equiv="refresh" content="{{ .RefreshInterval }}">
{{- end }}
<title>{{ .Title }}</title>
<style>
  :root {
    --color-frame: {{ .ColorFrame }};
    --color-border: {{ .ColorBorder }};
    --color-offline: {{ .ColorOffline }};
    --color-online: {{ .ColorOnline }};
    --color-highlight: {{ .ColorHigh }};
    --color-slow: {{ .ColorSlow }};
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    background-color: #1a1a1a;
    color: #e0e0e0;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 24px;
  }
  .card {
    background-color: #222222;
    border: 1px solid #333333;
    border-radius: 8px;
    padding: 24px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 18px;
    min-width: 350px;
    max-width: 100%;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    padding-bottom: 12px;
    border-bottom: 1px solid #333333;
    gap: 16px;
    flex-wrap: wrap;
  }
  h1 {
    font-size: 1.1rem;
    font-weight: 600;
    letter-spacing: 0.5px;
    color: #f5f5f5;
    white-space: nowrap;
  }
  .header-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  .meta {
    font-size: 0.8rem;
    color: #888888;
    white-space: nowrap;
  }
  .refresh-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: #2a2a2a;
    border: 1px solid #3a3a3a;
    padding: 3px 8px;
    border-radius: 4px;
    font-size: 0.75rem;
    color: #a0c4ff;
  }
  .delta-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    background: #1e293b;
    border: 1px solid #334155;
    padding: 3px 8px;
    border-radius: 4px;
    font-size: 0.75rem;
    color: #e2e8f0;
  }
  .pulse-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background-color: #52c41a;
    animation: pulse 1.5s infinite;
  }
  @keyframes pulse {
    0% { transform: scale(0.9); opacity: 0.7; }
    50% { transform: scale(1.3); opacity: 1; }
    100% { transform: scale(0.9); opacity: 0.7; }
  }
  .pause-btn {
    background: #383838;
    border: none;
    color: #dddddd;
    font-size: 0.7rem;
    padding: 1px 6px;
    border-radius: 3px;
    cursor: pointer;
  }
  .pause-btn:hover {
    background: #484848;
  }
  .header-actions {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }
  .action-btn {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    background: #282c34;
    border: 1px solid #3e4451;
    color: #abb2bf;
    font-size: 0.75rem;
    padding: 3px 9px;
    border-radius: 4px;
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
  body.light-theme {
    background-color: #f3f4f6;
    color: #1f2937;
  }
  body.light-theme .card {
    background-color: #ffffff;
    border-color: #e5e7eb;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.08);
  }
  body.light-theme h1 { color: #111827; }
  body.light-theme header { border-bottom-color: #e5e7eb; }
  body.light-theme .meta { color: #6b7280; }
  body.light-theme .action-btn {
    background: #f3f4f6;
    border-color: #d1d5db;
    color: #374151;
  }
  body.light-theme .action-btn:hover {
    background: #e5e7eb;
    color: #111827;
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
  body.light-theme .deltas-card {
    background: #f9fafb;
    border-color: #e5e7eb;
  }
  body.light-theme .deltas-title { color: #1f2937; }
  body.light-theme .delta-ip { color: #111827; }
  body.light-theme .scale-btn {
    background: #f3f4f6;
    border-color: #d1d5db;
    color: #374151;
  }
  .stats-bar {
    display: flex;
    gap: 8px;
    font-size: 0.82rem;
    flex-wrap: wrap;
    align-items: center;
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
  .stat-item {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .stat-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }
  .dot-online { background-color: var(--color-online); }
  .dot-fast { background-color: var(--color-highlight); }
  .dot-slow { background-color: var(--color-slow); }
  .dot-offline { background-color: var(--color-offline); }

  /* Exact pixel frame reproducing grid.png */
  .grid-canvas {
    width: {{ .Width }}px;
    height: {{ .Height }}px;
    background-color: var(--color-frame);
    position: relative;
    border: 1px solid #111111;
    overflow: hidden;
    user-select: none;
    box-shadow: 0 2px 8px rgba(0,0,0,0.5);
  }
  .grid-table {
    display: grid;
    grid-template-columns: repeat({{ .Cols }}, {{ .CellW }}px);
    grid-template-rows: repeat({{ .Rows }}, {{ .CellH }}px);
    gap: {{ .BorderW }}px;
    background-color: var(--color-border);
    position: absolute;
    top: {{ .PadTop }}px;
    left: {{ .PadLeft }}px;
    padding: {{ .BorderW }}px;
  }
  .cell {
    width: {{ .CellW }}px;
    height: {{ .CellH }}px;
    cursor: pointer;
    transition: transform 0.08s ease, filter 0.08s ease;
  }
  .cell.cell-empty {
    background-color: var(--color-frame) !important;
    pointer-events: none;
    cursor: default;
  }
  .cell:hover {
    outline: 1px solid #ffffff;
    z-index: 10;
    transform: scale(1.3);
    filter: brightness(1.2);
  }
  .cell.dimmed {
    opacity: 0.15;
    filter: grayscale(0.8) brightness(0.7);
  }
  .cell-joined {
    outline: 1.5px solid #52c41a;
    box-shadow: 0 0 8px #52c41a;
    animation: joinPulse 1.6s infinite ease-in-out;
  }
  @keyframes joinPulse {
    0%, 100% { box-shadow: 0 0 3px #52c41a; }
    50% { box-shadow: 0 0 10px #52c41a, inset 0 0 4px #52c41a; }
  }
  .cell-dropped {
    outline: 1.5px dashed #ff4d4f;
    box-shadow: 0 0 6px rgba(255, 77, 79, 0.7);
    animation: dropBlink 1.6s infinite ease-in-out;
  }
  @keyframes dropBlink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }
  .cell-changed {
    outline: 1.5px solid #faad14;
    box-shadow: 0 0 6px rgba(250, 173, 20, 0.6);
  }
  .deltas-card {
    width: 100%;
    background-color: #171b21;
    border: 1px solid #2d3748;
    border-radius: 6px;
    padding: 10px 14px;
    margin-top: 6px;
    font-size: 0.8rem;
  }
  .deltas-title {
    font-weight: 600;
    color: #e2e8f0;
    margin-bottom: 8px;
    font-size: 0.82rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  .deltas-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
    max-height: 140px;
    overflow-y: auto;
  }
  .delta-row {
    display: flex;
    align-items: center;
    gap: 10px;
    font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
    font-size: 0.78rem;
    padding: 2px 0;
  }
  .delta-badge-inline {
    text-transform: uppercase;
    font-size: 0.68rem;
    font-weight: 700;
    padding: 1px 6px;
    border-radius: 3px;
  }
  .delta-badge-joined { background: rgba(82, 196, 26, 0.2); color: #52c41a; border: 1px solid rgba(82, 196, 26, 0.4); }
  .delta-badge-dropped { background: rgba(255, 77, 79, 0.2); color: #ff4d4f; border: 1px solid rgba(255, 77, 79, 0.4); }
  .delta-badge-changed { background: rgba(250, 173, 20, 0.2); color: #faad14; border: 1px solid rgba(250, 173, 20, 0.4); }
  .delta-ip { color: #ffffff; font-weight: 600; }
  .delta-detail { color: #94a3b8; }
  .hud {
    width: 100%;
    min-height: 28px;
    background-color: #181818;
    border: 1px solid #2e2e2e;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.82rem;
    font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
    color: #cccccc;
    padding: 4px 12px;
  }
  .scale-controls {
    display: flex;
    gap: 8px;
    font-size: 0.75rem;
    color: #777777;
  }
  .scale-btn {
    background: #2b2b2b;
    border: 1px solid #3a3a3a;
    color: #cccccc;
    padding: 2px 8px;
    border-radius: 4px;
    cursor: pointer;
  }
  .scale-btn:hover {
    background: #383838;
  }
</style>
</head>

<body>

<div class="card">
  <header>
    <h1>PingGrid Subnet Activity</h1>
    <div class="header-right">
      <div class="header-actions">
        <button id="refresh-btn" class="action-btn" onclick="location.reload()" title="Reload sweep dashboard">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="23 4 23 10 17 10"></polyline><polyline points="1 20 1 14 7 14"></polyline><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg>
          <span>Refresh</span>
        </button>
        <button id="copy-btn" class="action-btn" onclick="copyActiveIPs()" title="Copy active responding IP addresses to clipboard">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
          <span id="copy-text">Copy Active ({{ .Online }})</span>
        </button>
        <button id="export-json-btn" class="action-btn" onclick="exportJSON()" title="Export scan results as JSON file">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path><polyline points="7 10 12 15 17 10"></polyline><line x1="12" y1="15" x2="12" y2="3"></line></svg>
          <span>JSON</span>
        </button>
        <button id="export-csv-btn" class="action-btn" onclick="exportCSV()" title="Export scan results as CSV file">
          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="8" y1="13" x2="16" y2="13"></line><line x1="8" y1="17" x2="16" y2="17"></line></svg>
          <span>CSV</span>
        </button>
        <button id="theme-btn" class="action-btn" onclick="toggleTheme()" title="Toggle Dark/Light theme">
          <span id="theme-icon">🌓</span>
          <span>Theme</span>
        </button>
        {{- if gt .RefreshInterval 0 }}
        <div class="refresh-badge">
          <span class="pulse-dot"></span>
          <span id="countdown-text">Refresh in {{ .RefreshInterval }}s</span>
          <button id="pause-btn" class="pause-btn" onclick="togglePause()">Pause</button>
        </div>
        {{- end }}
      </div>
      <div class="meta">{{ .GeneratedAt }} &bull; {{ .Duration }}{{ if or (eq .ScanMode "arp_cache") (eq .ScanMode "two_phase_arp") }} &bull; <span style="color: #60a5fa; font-weight: 600;">[Two-Phase ARP]</span>{{ end }}</div>
    </div>
  </header>

  <div class="stats-bar">
    <button class="filter-pill active" onclick="setFilter('all', this)" title="Show all hosts">All: {{ .Total }}</button>
    <button class="filter-pill" onclick="setFilter('active', this)" title="Highlight active responding hosts"><span class="stat-dot dot-online"></span> Active: {{ .Online }}</button>
    <button class="filter-pill" onclick="setFilter('fast', this)" title="Highlight fast responses (&le;20ms)"><span class="stat-dot dot-fast"></span> Fast: {{ .Fast }}</button>
    <button class="filter-pill" onclick="setFilter('slow', this)" title="Highlight slow responses (&ge;100ms)"><span class="stat-dot dot-slow"></span> Slow: {{ .Slow }}</button>
    <button class="filter-pill" onclick="setFilter('offline', this)" title="Highlight offline hosts"><span class="stat-dot dot-offline"></span> Offline: {{ .Offline }}</button>
    {{- if or (gt .JoinedCount 0) (gt .DroppedCount 0) }}
    <button class="filter-pill" onclick="setFilter('deltas', this)" title="Highlight recently changed hosts">
      {{- if gt .JoinedCount 0 }}<span style="color: #4ade80;">+{{ .JoinedCount }} joined</span>{{- end }}
      {{- if gt .DroppedCount 0 }} <span style="color: #f87171;">-{{ .DroppedCount }} dropped</span>{{- end }}
    </button>
    {{- end }}
  </div>

  <div id="canvas-wrapper" class="grid-canvas">
    <div class="grid-table">
      {{- range .Cells }}
      <div class="cell{{ if .Delta }} cell-{{ .Delta }}{{ end }}{{ if .IsEmpty }} cell-empty{{ end }}" style="background-color: {{ .Color }};"
           data-ip="{{ .IP }}" data-status="{{ .Status }}" data-rtt="{{ .RTT }}" data-delta="{{ .Delta }}"
           title="{{ .Tooltip }}"
           {{ if not .IsEmpty }}onmouseover="updateHud('{{ .IP }}', '{{ .Status }}', '{{ .RTT }}', '{{ .Delta }}')"
           onmouseout="resetHud()"{{ end }}></div>
      {{- end }}
    </div>
  </div>

  <div id="hud" class="hud">Hover over any grid cell to view host details</div>

  <div class="scale-controls">
    <span>Zoom:</span>
    <button class="scale-btn" onclick="setScale(1)">1x ({{ .Width }}&times;{{ .Height }})</button>
    <button class="scale-btn" onclick="setScale(2)">2x</button>
    <button class="scale-btn" onclick="setScale(3)">3x</button>
  </div>

  {{- if .Deltas }}
  <div class="deltas-card">
    <div class="deltas-title">
      <span>Recent State Changes ({{ len .Deltas }})</span>
      <span style="font-size: 0.72rem; color: #888;">Live Delta Tracker</span>
    </div>
    <div class="deltas-list">
      {{- range .Deltas }}
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

<script>
  function updateHud(ip, status, rtt, delta) {
    var hud = document.getElementById('hud');
    if (!hud) return;
    var deltaBadge = '';
    if (delta === 'joined') deltaBadge = ' &bull; <span style="color:#4ade80">[Recently Joined]</span>';
    else if (delta === 'dropped') deltaBadge = ' &bull; <span style="color:#f87171">[Recently Dropped]</span>';
    else if (delta === 'changed') deltaBadge = ' &bull; <span style="color:#fbbf24">[Status Shift]</span>';
    hud.innerHTML = '<strong style="color:#ffffff">' + ip + '</strong> &mdash; Status: <span style="color:#7dbeff">' + status + '</span> (' + rtt + ')' + deltaBadge;
  }
  function resetHud() {
    var hud = document.getElementById('hud');
    if (!hud) return;
    hud.innerHTML = 'Hover over any grid cell to view host details';
  }
  function setScale(factor) {
    var wrapper = document.getElementById('canvas-wrapper');
    if (!wrapper) return;
    wrapper.style.transform = 'scale(' + factor + ')';
    wrapper.style.transformOrigin = 'center center';
    wrapper.style.margin = ((factor - 1) * 38) + 'px 0';
  }

  var refreshSeconds = {{ .RefreshInterval }};
  var remaining = refreshSeconds;
  var isPaused = false;

  if (refreshSeconds > 0) {
    setInterval(function() {
      if (isPaused) return;
      remaining--;
      var el = document.getElementById('countdown-text');
      if (el) {
        if (remaining <= 0) {
          el.innerText = 'Refreshing...';
          location.reload();
        } else {
          el.innerText = 'Refresh in ' + remaining + 's';
        }
      }
    }, 1000);
  }

  function togglePause() {
    isPaused = !isPaused;
    var btn = document.getElementById('pause-btn');
    var el = document.getElementById('countdown-text');
    if (btn) {
      btn.innerText = isPaused ? 'Resume' : 'Pause';
    }
    if (el && isPaused) {
      el.innerText = 'Paused';
    }
  }

  function setFilter(type, el) {
    var pills = document.querySelectorAll('.filter-pill');
    for (var i = 0; i < pills.length; i++) {
      pills[i].classList.remove('active');
    }
    if (el) el.classList.add('active');
    var cells = document.querySelectorAll('.cell');
    for (var j = 0; j < cells.length; j++) {
      var c = cells[j];
      var st = c.getAttribute('data-status');
      var delta = c.getAttribute('data-delta');
      var match = true;
      if (type === 'active') {
        match = (st && st !== 'Offline');
      } else if (type === 'fast') {
        match = (st && st.indexOf('Fast') !== -1);
      } else if (type === 'slow') {
        match = (st && st.indexOf('Slow') !== -1);
      } else if (type === 'offline') {
        match = (st === 'Offline');
      } else if (type === 'deltas') {
        match = (delta && delta !== '');
      }
      if (match) {
        c.classList.remove('dimmed');
      } else {
        c.classList.add('dimmed');
      }
    }
  }

  function copyActiveIPs() {
    var cells = document.querySelectorAll('.cell:not(.cell-empty)');
    var active = [];
    for (var i = 0; i < cells.length; i++) {
      var st = cells[i].getAttribute('data-status');
      var ip = cells[i].getAttribute('data-ip');
      if (st && st !== 'Offline' && ip) {
        active.push(ip);
      }
    }
    if (active.length === 0) return;
    var text = active.join('\n');
    var doFeedback = function() {
      var btn = document.getElementById('copy-btn');
      var txt = document.getElementById('copy-text');
      if (txt) txt.innerText = 'Copied (' + active.length + ')!';
      if (btn) btn.classList.add('copied');
      setTimeout(function() {
        if (txt) txt.innerText = 'Copy Active (' + active.length + ')';
        if (btn) btn.classList.remove('copied');
      }, 2000);
    };
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(doFeedback).catch(function() {
        fallbackCopy(text, doFeedback);
      });
    } else {
      fallbackCopy(text, doFeedback);
    }
  }

  function fallbackCopy(text, cb) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.focus();
    ta.select();
    try {
      document.execCommand('copy');
      if (cb) cb();
    } catch (e) {}
    document.body.removeChild(ta);
  }

  function exportJSON() {
    var cells = document.querySelectorAll('.cell:not(.cell-empty)');
    var data = [];
    for (var i = 0; i < cells.length; i++) {
      data.push({
        ip: cells[i].getAttribute('data-ip'),
        status: cells[i].getAttribute('data-status'),
        rtt: cells[i].getAttribute('data-rtt'),
        delta: cells[i].getAttribute('data-delta') || ''
      });
    }
    var blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = 'pinggrid-results.json';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  function exportCSV() {
    var cells = document.querySelectorAll('.cell:not(.cell-empty)');
    var lines = ['IP,Status,RTT,Delta'];
    for (var i = 0; i < cells.length; i++) {
      lines.push(
        cells[i].getAttribute('data-ip') + ',' +
        '"' + (cells[i].getAttribute('data-status') || '') + '",' +
        '"' + (cells[i].getAttribute('data-rtt') || '') + '",' +
        '"' + (cells[i].getAttribute('data-delta') || '') + '"'
      );
    }
    var blob = new Blob([lines.join('\n')], { type: 'text/csv' });
    var url = URL.createObjectURL(blob);
    var a = document.createElement('a');
    a.href = url;
    a.download = 'pinggrid-results.csv';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  }

  function toggleTheme() {
    document.body.classList.toggle('light-theme');
    var isLight = document.body.classList.contains('light-theme');
    var icon = document.getElementById('theme-icon');
    if (icon) icon.innerText = isLight ? '☀️' : '🌓';
  }
</script>

</body>
</html>
`

const minimalHtmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
{{- if gt .RefreshInterval 0 }}
<meta http-equiv="refresh" content="{{ .RefreshInterval }}">
{{- end }}
<title>{{ .Title }} (Embed)</title>
<style>
  :root {
    --color-frame: {{ .ColorFrame }};
    --color-border: {{ .ColorBorder }};
    --color-offline: {{ .ColorOffline }};
    --color-online: {{ .ColorOnline }};
    --color-highlight: {{ .ColorHigh }};
    --color-slow: {{ .ColorSlow }};
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  html, body {
    margin: 0;
    padding: 0;
    width: 100%;
    height: 100%;
    overflow: hidden;
    background: transparent;
    display: flex;
    align-items: center;
    justify-content: center;
    user-select: none;
  }
  .grid-canvas {
    width: {{ .Width }}px;
    height: {{ .Height }}px;
    background-color: var(--color-frame);
    position: relative;
    border: 1px solid #111111;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0,0,0,0.5);
  }
  .grid-table {
    display: grid;
    grid-template-columns: repeat({{ .Cols }}, {{ .CellW }}px);
    grid-template-rows: repeat({{ .Rows }}, {{ .CellH }}px);
    gap: {{ .BorderW }}px;
    background-color: var(--color-border);
    position: absolute;
    top: {{ .PadTop }}px;
    left: {{ .PadLeft }}px;
    padding: {{ .BorderW }}px;
  }
  .cell {
    width: {{ .CellW }}px;
    height: {{ .CellH }}px;
    cursor: pointer;
    transition: transform 0.08s ease, filter 0.08s ease;
  }
  .cell.cell-empty {
    background-color: var(--color-frame) !important;
    pointer-events: none;
    cursor: default;
  }
  .cell:hover {
    outline: 1px solid #ffffff;
    z-index: 10;
    transform: scale(1.3);
    filter: brightness(1.2);
  }
  .cell-joined {
    outline: 1.5px solid #52c41a;
    box-shadow: 0 0 8px #52c41a;
    animation: joinPulse 1.6s infinite ease-in-out;
  }
  @keyframes joinPulse {
    0%, 100% { box-shadow: 0 0 3px #52c41a; }
    50% { box-shadow: 0 0 10px #52c41a, inset 0 0 4px #52c41a; }
  }
  .cell-dropped {
    outline: 1.5px dashed #ff4d4f;
    box-shadow: 0 0 6px rgba(255, 77, 79, 0.7);
    animation: dropBlink 1.6s infinite ease-in-out;
  }
  @keyframes dropBlink {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }
  .cell-changed {
    outline: 1.5px solid #faad14;
    box-shadow: 0 0 6px rgba(250, 173, 20, 0.6);
  }
</style>
</head>
<body>
  <div class="grid-canvas">
    <div class="grid-table">
      {{- range .Cells }}
      <div class="cell{{ if .Delta }} cell-{{ .Delta }}{{ end }}{{ if .IsEmpty }} cell-empty{{ end }}" style="background-color: {{ .Color }};"
           data-ip="{{ .IP }}" data-status="{{ .Status }}" data-rtt="{{ .RTT }}" data-delta="{{ .Delta }}"
           title="{{ .Tooltip }}"></div>
      {{- end }}
    </div>
  </div>
  {{- if gt .RefreshInterval 0 }}
  <script>
    setTimeout(function() {
      location.reload();
    }, {{ .RefreshInterval }} * 1000);
  </script>
  {{- end }}
</body>
</html>
`

// buildHTMLData prepares the unified data structure needed by both HTML templates.
func buildHTMLData(cfg GridConfig, results []scanner.HostResult, sweepDuration time.Duration, refreshInterval int, deltas []scanner.HostDelta) HTMLData {
	deltaMap := make(map[string]scanner.HostDelta, len(deltas))
	var joinedCount, droppedCount, changedCount int
	for _, d := range deltas {
		deltaMap[d.IP.String()] = d
		switch d.Kind {
		case scanner.DeltaJoined:
			joinedCount++
		case scanner.DeltaDropped:
			droppedCount++
		case scanner.DeltaChanged:
			changedCount++
		}
	}

	var (
		cellW   = 8
		cellH   = 8
		borderW = cfg.BorderWidth
		padLeft = 3
		padTop  = 1
	)
	if borderW <= 0 {
		borderW = 1
	}

	if cfg.Width != DefaultWidth || cfg.Height != DefaultHeight || cfg.Rows != DefaultRows || cfg.Cols != DefaultCols {
		availW := cfg.Width - (cfg.Cols+1)*borderW
		if availW < cfg.Cols {
			availW = cfg.Cols
		}
		availH := cfg.Height - (cfg.Rows+1)*borderW
		if availH < cfg.Rows {
			availH = cfg.Rows
		}
		cW := availW / cfg.Cols
		cH := availH / cfg.Rows
		cellSize := cW
		if cH < cellSize {
			cellSize = cH
		}
		if cellSize < 1 {
			cellSize = 1
		}
		cellW = cellSize
		cellH = cellSize

		totalGridW := cfg.Cols*cellW + (cfg.Cols+1)*borderW
		totalGridH := cfg.Rows*cellH + (cfg.Rows+1)*borderW

		padLeft = (cfg.Width - totalGridW) / 2
		if padLeft < 0 {
			padLeft = 0
		}
		padTop = (cfg.Height - totalGridH) / 2
		if padTop < 0 {
			padTop = 0
		}
	}

	totalSlots := cfg.Rows * cfg.Cols
	cells := make([]HTMLCell, totalSlots)

	var onlineCount, fastCount, slowCount, offlineCount int

	for i := 0; i < totalSlots; i++ {
		if i >= len(results) {
			cells[i] = HTMLCell{
				Index:   i,
				IP:      "",
				Status:  "",
				Color:   "transparent",
				RTT:     "",
				IsUp:    false,
				Tooltip: "",
				Delta:   "",
				IsEmpty: true,
			}
			continue
		}

		var (
			ipStr    = fmt.Sprintf("Slot %d", i)
			status   string
			cellCol  string
			rttStr   = "no reply"
			isUp     = false
			deltaTag = ""
		)

		res := results[i]
		if res.IP != nil {
			ipStr = res.IP.String()
		}

		if d, ok := deltaMap[ipStr]; ok {
			deltaTag = string(d.Kind)
		}

		switch res.Status {
		case scanner.StatusOnline:
			status = "Online"
			cellCol = HexString(cfg.ColorOnline)
			rttStr = scanner.FormatDurationMS(res.RTT)
			isUp = true
			onlineCount++

		case scanner.StatusHighlight:
			status = "Fast / Gateway"
			cellCol = HexString(cfg.ColorHighlight)
			rttStr = scanner.FormatDurationMS(res.RTT)
			isUp = true
			fastCount++
			onlineCount++

		case scanner.StatusSlow:
			status = "Slow Latency"
			cellCol = HexString(cfg.ColorSlow)
			rttStr = scanner.FormatDurationMS(res.RTT)
			isUp = true
			slowCount++
			onlineCount++

		default:
			status = "Offline"
			cellCol = HexString(cfg.ColorOffline)
			offlineCount++
		}

		tooltip := fmt.Sprintf("%s - %s (%s)", ipStr, status, rttStr)
		if deltaTag != "" {
			tooltip += fmt.Sprintf(" [%s]", deltaTag)
		}

		cells[i] = HTMLCell{
			Index:   i,
			IP:      ipStr,
			Status:  status,
			Color:   cellCol,
			RTT:     rttStr,
			IsUp:    isUp,
			Tooltip: tooltip,
			Delta:   deltaTag,
			IsEmpty: false,
		}
	}

	return HTMLData{
		Title:           "PingGrid - Network Subnet Status",
		GeneratedAt:     time.Now().Format("2006-01-02 15:04:05 MST"),
		Rows:            cfg.Rows,
		Cols:            cfg.Cols,
		Width:           cfg.Width,
		Height:          cfg.Height,
		CellW:           cellW,
		CellH:           cellH,
		BorderW:         borderW,
		PadLeft:         padLeft,
		PadTop:          padTop,
		ColorFrame:      HexString(cfg.ColorFrame),
		ColorBorder:     HexString(cfg.ColorBorder),
		ColorOffline:    HexString(cfg.ColorOffline),
		ColorOnline:     HexString(cfg.ColorOnline),
		ColorHigh:       HexString(cfg.ColorHighlight),
		ColorSlow:       HexString(cfg.ColorSlow),
		Total:           len(results),
		Online:          onlineCount,
		Fast:            fastCount,
		Slow:            slowCount,
		Offline:         offlineCount,
		Duration:        scanner.FormatDurationMS(sweepDuration),
		RefreshInterval: refreshInterval,
		JoinedCount:     joinedCount,
		DroppedCount:    droppedCount,
		ChangedCount:    changedCount,
		Deltas:          deltas,
		Cells:           cells,
		ScanMode:        cfg.ScanMode,
	}
}

// RenderHTML generates a standalone interactive HTML dashboard representing the ping grid.
func RenderHTML(cfg GridConfig, results []scanner.HostResult, sweepDuration time.Duration, refreshInterval int, deltas []scanner.HostDelta) ([]byte, error) {
	tmpl, err := template.New("grid_standalone").Parse(htmlTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse standalone HTML template: %w", err)
	}

	data := buildHTMLData(cfg, results, sweepDuration, refreshInterval, deltas)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render standalone HTML template: %w", err)
	}

	return buf.Bytes(), nil
}

// RenderMinimalHTML generates a minimal, button-free HTML view optimized for embedded iframe use.
func RenderMinimalHTML(cfg GridConfig, results []scanner.HostResult, sweepDuration time.Duration, refreshInterval int, deltas []scanner.HostDelta) ([]byte, error) {
	tmpl, err := template.New("grid_minimal").Parse(minimalHtmlTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse minimal HTML template: %w", err)
	}

	data := buildHTMLData(cfg, results, sweepDuration, refreshInterval, deltas)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render minimal HTML template: %w", err)
	}

	return buf.Bytes(), nil
}

// DeriveEmbedPath derives the minimal embed/iframe file path from a base output path.
// e.g. "grid.html" -> "grid-embed.html", "dashboard.htm" -> "dashboard-embed.htm".
func DeriveEmbedPath(outputPath string) string {
	ext := filepath.Ext(outputPath)
	if ext == "" {
		return outputPath + "-embed.html"
	}
	base := strings.TrimSuffix(outputPath, ext)
	return base + "-embed" + ext
}

// SaveHTML writes HTML content to the specified file path.
func SaveHTML(content []byte, path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return err
		}
	}
	return os.WriteFile(path, content, 0600)
}
