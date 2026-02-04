/* NetPulse Dashboard JavaScript */

// ===== Global State =====
let latencyChart = null;
let currentTracePage = 1;
let map = null;
let pathLayer = null;
let markersLayer = null;
let currentTab = 'overview';
let updateInterval = null;

// ===== Initialization =====
document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    initMermaid();
    startLiveUpdates();
    setTimeout(loadTopology, 500);
});

// ===== Theme Management =====
function initTheme() {
    const savedTheme = localStorage.getItem('netpulse-theme') || 'hacker';
    document.documentElement.setAttribute('data-theme', savedTheme);

    document.querySelectorAll('.theme-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.theme === savedTheme);
        btn.addEventListener('click', () => setTheme(btn.dataset.theme));
    });
}

function setTheme(theme) {
    document.documentElement.setAttribute('data-theme', theme);
    localStorage.setItem('netpulse-theme', theme);
    document.querySelectorAll('.theme-btn').forEach(b => {
        b.classList.toggle('active', b.dataset.theme === theme);
    });
}

// ===== Mermaid =====
function initMermaid() {
    mermaid.initialize({
        startOnLoad: false,
        theme: 'dark',
        themeVariables: {
            primaryColor: '#00ff41',
            primaryTextColor: '#fff',
            primaryBorderColor: '#00ff41',
            lineColor: '#00ff41',
            background: '#0a0f0a'
        }
    });
}

// ===== Tab Navigation =====
function showTab(tabId) {
    currentTab = tabId;
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
    event.target.classList.add('active');
    document.getElementById(tabId).classList.add('active');

    // Immediate update
    updateActiveTab();

    if (tabId === 'topology') loadTopology();
    if (tabId === 'map') initMap();
    if (tabId === 'dns' && window.initDNS) window.initDNS();
}

// ===== Host Filtering =====
function filterHosts() {
    const filter = document.getElementById('hostFilter').value.toLowerCase();
    document.querySelectorAll('.host-card').forEach(card => {
        const ip = card.querySelector('.host-ip').textContent.toLowerCase();
        const name = card.querySelector('.host-name').textContent.toLowerCase();
        if (ip.includes(filter) || name.includes(filter)) {
            card.style.display = '';
        } else {
            card.style.display = 'none';
        }
    });
}

// ===== Traces with Pagination =====
async function loadTraces(page = 1) {
    currentTracePage = page;
    const target = document.getElementById('traceFilter')?.value || '';
    const limit = 10;

    // Get time filter params
    let params = `page=${page}&limit=${limit}&target=${encodeURIComponent(target)}`;
    if (window.getGlobalTimeParams) {
        params += '&' + window.getGlobalTimeParams();
    }

    try {
        const res = await fetch(`/api/traces?${params}`);
        const data = await res.json();

        const container = document.getElementById('traceResults');
        const pagination = document.getElementById('tracePagination');

        if (!data.traces || data.traces.length === 0) {
            container.innerHTML = '<p class="empty-state">> No traces found</p>';
            pagination.innerHTML = '';
            return;
        }

        container.innerHTML = data.traces.map(t => `
            <div class="trace-path">
                <strong>> ${t.target}</strong> @ ${new Date(t.timestamp).toLocaleTimeString()}
                ${(t.hops || []).map(h => `
                    <div class="trace-hop">
                        <span class="hop-num">${h.hop_num}.</span>
                        <span class="hop-ip">${h.lost ? '* * *' : h.ip}</span>
                        <span class="hop-latency">${h.lost ? '' : h.latency_ms.toFixed(1) + ' ms'}</span>
                    </div>
                `).join('')}
            </div>
        `).join('');

        renderPagination(pagination, data, 'loadTraces');
    } catch (e) { console.error('Traces error:', e); }
}

function renderPagination(container, data, fn) {
    let html = `<span class="page-info">Page ${data.page} of ${data.total_pages} (${data.total} total)</span>`;
    html += `<button onclick="${fn}(1)" ${data.page <= 1 ? 'disabled' : ''}>«</button>`;
    html += `<button onclick="${fn}(${data.page - 1})" ${data.page <= 1 ? 'disabled' : ''}>‹</button>`;

    for (let i = Math.max(1, data.page - 2); i <= Math.min(data.total_pages, data.page + 2); i++) {
        html += `<button onclick="${fn}(${i})" class="${i === data.page ? 'active' : ''}">${i}</button>`;
    }

    html += `<button onclick="${fn}(${data.page + 1})" ${data.page >= data.total_pages ? 'disabled' : ''}>›</button>`;
    html += `<button onclick="${fn}(${data.total_pages})" ${data.page >= data.total_pages ? 'disabled' : ''}>»</button>`;

    container.innerHTML = html;
}

// ===== Topology =====
async function loadTopology() {
    const target = document.getElementById('topologyTarget')?.value || '';
    try {
        const res = await fetch(`/api/analytics/mermaid?target=${encodeURIComponent(target)}`);
        const diagram = await res.text();
        const el = document.getElementById('topologyDiagram');
        el.innerHTML = diagram;
        el.removeAttribute('data-processed');
        mermaid.init(undefined, el);
    } catch (e) { console.error('Topology error:', e); }
}

// ===== Latency Chart =====
async function loadLatencyChart() {
    const target = document.getElementById('latencyTarget')?.value || '';
    try {
        const res = await fetch(`/api/analytics/latency?target=${encodeURIComponent(target)}`);
        const data = await res.json();

        if (latencyChart) latencyChart.destroy();

        const grouped = {};
        (data || []).forEach(p => {
            if (!grouped[p.target]) grouped[p.target] = [];
            grouped[p.target].push({ x: new Date(p.timestamp), y: p.latency_ms });
        });

        const colors = ['#00ff41', '#ff00ff', '#ff4444', '#ffaa00', '#00aaff'];
        const datasets = Object.keys(grouped).map((t, i) => ({
            label: t,
            data: grouped[t],
            borderColor: colors[i % colors.length],
            tension: 0.3,
            fill: false
        }));

        const ctx = document.getElementById('latencyChart').getContext('2d');
        latencyChart = new Chart(ctx, {
            type: 'line',
            data: { datasets },
            options: {
                responsive: true,
                scales: {
                    x: { type: 'time', time: { unit: 'minute' }, ticks: { color: '#666' }, grid: { color: '#222' } },
                    y: { title: { display: true, text: 'Latency (ms)', color: '#666' }, ticks: { color: '#666' }, grid: { color: '#222' } }
                },
                plugins: { legend: { labels: { color: '#888' } } }
            }
        });
    } catch (e) { console.error('Latency chart error:', e); }
}

// ===== Anomalies =====
async function loadAnomalies() {
    try {
        const res = await fetch('/api/analytics/anomalies');
        const data = await res.json();
        const el = document.getElementById('anomalyList');

        if (!data || data.length === 0) {
            el.innerHTML = '<p class="empty-state">> No route changes detected in last 24h</p>';
            return;
        }

        el.innerHTML = data.map(a => `
            <div class="anomaly-card">
                <div class="anomaly-title">⚠ Route Change to ${a.target}</div>
                <div>Detected: ${new Date(a.detected_at).toLocaleString()}</div>
                <div>Changed hops: ${a.changed_hops.join(', ')}</div>
                <div style="margin-top:0.5rem;font-size:0.75rem;color:var(--text-dim)">
                    Old: ${a.old_path.slice(0, 5).join(' → ')}${a.old_path.length > 5 ? '...' : ''}<br>
                    New: ${a.new_path.slice(0, 5).join(' → ')}${a.new_path.length > 5 ? '...' : ''}
                </div>
            </div>
        `).join('');
    } catch (e) { console.error('Anomalies error:', e); }
}

// ===== GeoIP Map =====
async function initMap() {
    const mapEl = document.getElementById('geoMap');
    if (!mapEl) return;

    if (!map) {
        map = L.map('geoMap').setView([30, 0], 2);
        L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
            attribution: '&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors &copy; <a href="https://carto.com/attributions">CARTO</a>',
            subdomains: 'abcd',
            maxZoom: 20
        }).addTo(map);
        pathLayer = L.layerGroup().addTo(map);
        markersLayer = L.layerGroup().addTo(map);
    }

    loadMapTraces();
}

async function loadMapTraces() {
    try {
        const res = await fetch('/api/traces?limit=20');
        const data = await res.json();

        const panel = document.getElementById('mapTraceList');
        if (!panel) return;

        panel.innerHTML = (data.traces || []).map(t => `
            <div class="trace-item" onclick="showTraceOnMap(${t.id})" data-id="${t.id}">
                <div class="target">${t.target}</div>
                <div class="time">${new Date(t.timestamp).toLocaleString()}</div>
            </div>
        `).join('');

        if (data.traces && data.traces.length > 0) {
            showTraceOnMap(data.traces[0].id);
        }
    } catch (e) {
        console.error('Save status error:', e);
        alert('Failed to save metadata');
    }
}

// ===== Global Time Filter =====
window.getGlobalTimeParams = function () {
    const range = document.getElementById('timeRange')?.value || '24h';
    const now = new Date();
    let start = new Date();

    switch (range) {
        case '1h': start.setHours(now.getHours() - 1); break;
        case '6h': start.setHours(now.getHours() - 6); break;
        case '24h': start.setHours(now.getHours() - 24); break;
        case '7d': start.setDate(now.getDate() - 7); break;
        case '30d': start.setDate(now.getDate() - 30); break;
    }
    return `start=${start.toISOString()}&end=${now.toISOString()}`;
}

window.updateGlobalTime = function () {
    const filter = window.getGlobalTimeParams();
    console.log("Time Filter:", filter);

    // Trigger updates
    updateActiveTab();

    // Explicitly update DNS if needed (it has its own interval, but we want instant refresh)
    if (document.getElementById('dns').classList.contains('active') && window.updateDNS) {
        window.updateDNS();
    }
}    // Update selection
async function showTraceOnMap(traceId) {
    // Update selection
    document.querySelectorAll('.trace-item').forEach(el => {
        el.classList.toggle('selected', el.dataset.id == traceId);
    });

    // Clear previous
    pathLayer.clearLayers();
    markersLayer.clearLayers();

    try {
        const res = await fetch(`/api/traces/${traceId}/geo`);
        const data = await res.json();

        if (!data.hops || data.hops.length === 0) {
            return;
        }

        const points = data.hops.filter(h => h.lat && h.lon).map(h => [h.lat, h.lon]);

        if (points.length > 0) {
            // Draw path with animation
            L.polyline(points, {
                color: '#00ff41',
                weight: 3,
                opacity: 0.8,
                className: 'animated-path'
            }).addTo(pathLayer);

            // Add custom markers
            data.hops.filter(h => h.lat && h.lon).forEach((h, i) => {
                let type = 'hop';
                if (h.hop_num === 0) type = 'source';
                else if (i === data.hops.filter(hx => hx.lat && hx.lon).length - 1) type = 'target';

                const icon = L.divIcon({
                    className: '', // Clear default
                    html: `<div class="map-marker ${type}"><div class="map-marker-ring"></div><div class="map-marker-inner"></div></div>`,
                    iconSize: [24, 24],
                    iconAnchor: [12, 12]
                });

                const marker = L.marker([h.lat, h.lon], { icon: icon }).addTo(markersLayer);

                const typeLabel = type === 'source' ? 'Source' : (type === 'target' ? 'Target' : 'Hop ' + h.hop_num);
                const badgeClass = type;

                marker.bindPopup(`
                    <div class="popup-header">
                        <span>${typeLabel}</span>
                        <span class="popup-badge ${badgeClass}">${type.toUpperCase()}</span>
                    </div>
                    <div class="popup-body">
                        <div class="popup-row">
                            <span class="popup-label">IP Address</span>
                            <span class="popup-val">${h.ip}</span>
                        </div>
                        <div class="popup-row">
                            <span class="popup-label">Latency</span>
                            <span class="popup-val">${h.latency_ms.toFixed(1)} ms</span>
                        </div>
                        <div class="popup-row">
                            <span class="popup-label">Location</span>
                            <span class="popup-val">${h.city || 'Unknown'}, ${h.country || ''}</span>
                        </div>
                        ${h.as || h.org ? `
                        <div class="popup-asn">
                            <span class="popup-org">${h.org || 'Unknown Org'}</span>
                            <div>${h.as || ''}</div>
                        </div>
                        ` : ''}
                    </div>
                `, {
                    className: 'glass-popup',
                    maxWidth: 300
                });
            });

            // Fit bounds
            map.fitBounds(points, { padding: [50, 50] });
        }
    } catch (e) { console.error('Show trace on map error:', e); }
}

// ===== Real-Time Updates =====
function startLiveUpdates() {
    updateActiveTab();
    updateInterval = setInterval(updateActiveTab, 3000);
}

function updateActiveTab() {
    if (document.hidden) return;

    if (currentTab === 'overview') updateOverview();
    else if (currentTab === 'hosts') updateHosts();
    else if (currentTab === 'latency') loadLatencyChart();
    else if (currentTab === 'traces') loadTraces(currentTracePage);
    else if (currentTab === 'anomalies') loadAnomalies();
}

async function updateOverview() {
    try {
        const res = await fetch('/api/status');
        const data = await res.json();

        const set = (id, val) => { const el = document.getElementById(id); if (el) el.textContent = val; };

        if (data.current_ip) {
            set('stat-ip', data.current_ip);
            set('stat-isp', data.isp);
            set('stat-asn', data.asn);
            set('stat-last-check', data.last_check);
        }

        const daemonEl = document.getElementById('stat-daemon');
        if (daemonEl) {
            daemonEl.innerHTML = data.running
                ? '<span class="status-badge status-running">Online</span>'
                : '<span class="status-badge status-stopped">Offline</span>';
        }

        set('stat-ip-records', data.ip_records);
        set('stat-alive-hosts', data.alive_hosts);
        set('stat-open-ports', data.open_ports);

    } catch (e) { console.error('Overview update failed', e); }
}

async function updateHosts() {
    try {
        const res = await fetch('/api/hosts');
        const hosts = await res.json();
        const container = document.getElementById('hostGrid');
        if (!container) return;

        if (!hosts || hosts.length === 0) {
            return;
        }

        const currentFilter = document.getElementById('hostFilter').value.toLowerCase();

        container.innerHTML = hosts.map(h => {
            const portsHtml = h.ports && h.ports.length > 0
                ? `<div class="port-grid">${h.ports.map(p =>
                    `<div class="port-badge" title="${p.service}"><span class="port-num">${p.port}</span><span class="port-proto">${p.protocol}</span></div>`
                ).join('')}</div>`
                : `<div class="no-ports">No open ports found</div>`;

            const displayName = h.display_name || h.hostname || 'Unknown Device';
            const iconChar = getIconChar(h.icon);
            const safeJson = JSON.stringify(h).replace(/'/g, "&apos;").replace(/"/g, "&quot;");

            const match = h.ip.toLowerCase().includes(currentFilter) || displayName.toLowerCase().includes(currentFilter);
            const display = match ? '' : 'display:none';

            const tagsHtml = h.tags ? `<div class="host-tags">${h.tags.map(t => `<span class="host-tag">${t}</span>`).join('')}</div>` : '';

            return `
             <div class="host-card" data-ip="${h.ip}" style="${display}" onclick='openAssetModal(${safeJson})'>
                 <div class="host-header">
                     <div class="host-id">
                         <span class="host-icon">${iconChar}</span>
                         <div>
                             <span class="host-ip">${h.ip}</span>
                             <span class="host-name">${displayName}</span>
                         </div>
                     </div>
                     <div class="host-status">
                         <span class="status-dot online"></span>
                         <span class="latency">${h.latency_ms.toFixed(0)}MS</span>
                     </div>
                 </div>
                 ${tagsHtml}
                 <div class="port-section">
                     <div class="port-label">OPEN PORTS detected</div>
                     ${portsHtml}
                 </div>
                 <div class="host-footer">
                     <span class="last-seen">Seen: ${new Date(h.last_seen).toLocaleTimeString()}</span>
                 </div>
             </div>
             `;
        }).join('');

    } catch (e) { console.error('Hosts update failed', e); }
}

// ===== Asset Modal =====
function openAssetModal(asset) {
    document.getElementById('assetId').value = asset.id;
    document.getElementById('assetName').value = asset.display_name || asset.hostname || '';
    document.getElementById('assetTags').value = (asset.tags || []).join(', ');
    document.getElementById('assetIcon').value = asset.icon || '';

    // Highlight selected icon
    document.querySelectorAll('.icon-option').forEach(el => {
        el.classList.toggle('selected', el.dataset.icon === asset.icon);
    });

    document.getElementById('assetModal').style.display = 'block';
}

function closeAssetModal() {
    document.getElementById('assetModal').style.display = 'none';
}

function selectIcon(icon) {
    document.getElementById('assetIcon').value = icon;
    document.querySelectorAll('.icon-option').forEach(el => {
        el.classList.toggle('selected', el.dataset.icon === icon);
    });
}

async function saveAssetMetadata() {
    const id = document.getElementById('assetId').value;
    const displayName = document.getElementById('assetName').value;
    const tagsStr = document.getElementById('assetTags').value;
    const icon = document.getElementById('assetIcon').value;

    const tags = tagsStr.split(',').map(t => t.trim()).filter(t => t);

    try {
        const res = await fetch(`/api/hosts/${id}/metadata`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ display_name: displayName, tags, icon })
        });

        if (res.ok) {
            closeAssetModal();
            updateHosts(); // Refresh grid
        } else {
            alert('Failed to save');
        }
    } catch (e) { console.error('Save failed', e); }
}

function getIconChar(name) {
    const map = {
        'desktop': '🖥️', 'laptop': '💻', 'server': '🗄️',
        'phone': '📱', 'iot': '🔌', 'printer': '🖨️', 'router': '🌐', 'camera': '📷'
    };
    return map[name] || '💻';
}

// Close modal on outside click
window.onclick = function (event) {
    const modal = document.getElementById('assetModal');
    if (event.target === modal) {
        closeAssetModal();
    }
}

// ===== Global Time Filter =====
window.getGlobalTimeParams = function () {
    const range = document.getElementById('timeRange')?.value || '24h';
    const now = new Date();
    let start = new Date();

    switch (range) {
        case '1h': start.setHours(now.getHours() - 1); break;
        case '6h': start.setHours(now.getHours() - 6); break;
        case '24h': start.setHours(now.getHours() - 24); break;
        case '7d': start.setDate(now.getDate() - 7); break;
        case '30d': start.setDate(now.getDate() - 30); break;
    }
    return `start=${start.toISOString()}&end=${now.toISOString()}`;
}

window.updateGlobalTime = function () {
    const filter = window.getGlobalTimeParams();
    console.log("Time Filter:", filter);

    // Trigger updates
    updateActiveTab();

    // Explicitly update DNS if needed (it has its own interval, but we want instant refresh)
    if (document.getElementById('dns').classList.contains('active') && window.updateDNS) {
        window.updateDNS();
    }
}

// ===== Trace Comparison Functions =====
function showCompareMode() {
    document.getElementById('traces').style.display = 'none';
    document.getElementById('traceCompare').style.display = 'block';
}

function hideCompareMode() {
    document.getElementById('traces').style.display = 'block';
    document.getElementById('traceCompare').style.display = 'none';
    document.getElementById('compareSelector').style.display = 'none';
    document.getElementById('compareResults').style.display = 'none';
}

async function loadCompareTraceList() {
    const target = document.getElementById('compareTarget').value;
    if (!target) {
        document.getElementById('compareSelector').style.display = 'none';
        document.getElementById('compareResults').style.display = 'none';
        return;
    }

    try {
        const res = await fetch(`/api/traces/by-target?target=${encodeURIComponent(target)}&limit=100`);
        const traces = await res.json();

        if (!traces || traces.length === 0) {
            alert('No traces found for this target');
            return;
        }

        const select1 = document.getElementById('compareTrace1');
        const select2 = document.getElementById('compareTrace2');

        const options = traces.map(t => {
            const date = new Date(t.timestamp);
            const dateStr = date.toLocaleString();
            return `<option value="${t.id}" data-timestamp="${t.timestamp}">${dateStr}</option>`;
        }).join('');

        select1.innerHTML = '<option value="">Select timestamp...</option>' + options;
        select2.innerHTML = '<option value="">Select timestamp...</option>' + options;

        document.getElementById('compareSelector').style.display = 'block';
        document.getElementById('compareResults').style.display = 'none';
    } catch (e) {
        console.error('Failed to load traces:', e);
        alert('Failed to load traces');
    }
}

async function performComparison() {
    const trace1Id = document.getElementById('compareTrace1').value;
    const trace2Id = document.getElementById('compareTrace2').value;

    if (!trace1Id || !trace2Id) {
        alert('Please select both traces');
        return;
    }

    if (trace1Id === trace2Id) {
        alert('Please select different traces');
        return;
    }

    try {
        // Fetch both traces
        const [res1, res2] = await Promise.all([
            fetch(`/api/traces/${trace1Id}`),
            fetch(`/api/traces/${trace2Id}`)
        ]);

        const trace1 = await res1.json();
        const trace2 = await res2.json();

        // Get public IPs for both timestamps
        const [ip1Res, ip2Res] = await Promise.all([
            fetch(`/api/public-ip-at-time?timestamp=${encodeURIComponent(trace1.timestamp)}`),
            fetch(`/api/public-ip-at-time?timestamp=${encodeURIComponent(trace2.timestamp)}`)
        ]);

        const ip1Data = await ip1Res.json();
        const ip2Data = await ip2Res.json();

        // Display results
        displayComparison(trace1, trace2, ip1Data, ip2Data);
    } catch (e) {
        console.error('Comparison failed:', e);
        alert('Failed to perform comparison');
    }
}

function displayComparison(trace1, trace2, ip1Data, ip2Data) {
    // Update headers
    document.getElementById('compare1Header').textContent =
        `Trace - ${new Date(trace1.timestamp).toLocaleString()}`;
    document.getElementById('compare2Header').textContent =
        `Trace - ${new Date(trace2.timestamp).toLocaleString()}`;

    // Display public IP info
    document.getElementById('compare1IP').innerHTML = `
        <strong>Public IP:</strong> ${ip1Data.ip}<br>
        <strong>ISP:</strong> ${ip1Data.isp || 'Unknown'}<br>
        <strong>ASN:</strong> ${ip1Data.asn || 'Unknown'}
    `;

    document.getElementById('compare2IP').innerHTML = `
        <strong>Public IP:</strong> ${ip2Data.ip}<br>
        <strong>ISP:</strong> ${ip2Data.isp || 'Unknown'}<br>
        <strong>ASN:</strong> ${ip2Data.asn || 'Unknown'}
    `;

    // Display trace hops
    document.getElementById('compare1Content').innerHTML = renderTraceHops(trace1.hops);
    document.getElementById('compare2Content').innerHTML = renderTraceHops(trace2.hops);

    // Show results
    document.getElementById('compareResults').style.display = 'block';
}

function renderTraceHops(hops) {
    if (!hops || hops.length === 0) {
        return '<p class="empty-state">> No hops</p>';
    }

    return hops.map(h => `
        <div class="trace-hop" style="display: flex; padding: 0.4rem 0; border-bottom: 1px dashed var(--border-color);">
            <span class="hop-num" style="color: var(--text-dim); width: 40px;">${h.hop_num}.</span>
            <span class="hop-ip" style="color: var(--accent); flex: 1;">${h.lost ? '* * *' : h.ip}</span>
            <span class="hop-latency" style="color: var(--text-secondary);">${h.lost ? '' : h.latency_ms.toFixed(1) + ' ms'}</span>
        </div>
    `).join('');
}
