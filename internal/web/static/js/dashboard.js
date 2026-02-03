/* NetPulse Dashboard JavaScript */

// ===== Global State =====
let latencyChart = null;
let currentTracePage = 1;
let map = null;
let pathLayer = null;
let markersLayer = null;

// ===== Initialization =====
document.addEventListener('DOMContentLoaded', () => {
    initTheme();
    initMermaid();
    startCountdown();
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
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
    event.target.classList.add('active');
    document.getElementById(tabId).classList.add('active');
    
    if (tabId === 'topology') loadTopology();
    if (tabId === 'traces') loadTraces(1);
    if (tabId === 'latency') loadLatencyChart();
    if (tabId === 'anomalies') loadAnomalies();
    if (tabId === 'map') initMap();
}

// ===== Host Filtering =====
function filterHosts() {
    const filter = document.getElementById('hostFilter').value.toLowerCase();
    document.querySelectorAll('#hostTable tbody tr').forEach(row => {
        row.style.display = row.cells[0].textContent.toLowerCase().includes(filter) ? '' : 'none';
    });
}

// ===== Traces with Pagination =====
async function loadTraces(page = 1) {
    currentTracePage = page;
    const target = document.getElementById('traceFilter')?.value || '';
    const limit = 10;
    
    try {
        const res = await fetch(`/api/traces?page=${page}&limit=${limit}&target=${encodeURIComponent(target)}`);
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
    } catch(e) { console.error('Traces error:', e); }
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
    } catch(e) { console.error('Topology error:', e); }
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
    } catch(e) { console.error('Latency chart error:', e); }
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
                    Old: ${a.old_path.slice(0,5).join(' → ')}${a.old_path.length > 5 ? '...' : ''}<br>
                    New: ${a.new_path.slice(0,5).join(' → ')}${a.new_path.length > 5 ? '...' : ''}
                </div>
            </div>
        `).join('');
    } catch(e) { console.error('Anomalies error:', e); }
}

// ===== GeoIP Map =====
async function initMap() {
    const mapEl = document.getElementById('geoMap');
    if (!mapEl) return;
    
    if (!map) {
        map = L.map('geoMap').setView([30, 0], 2);
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap'
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
    } catch(e) { console.error('Map traces error:', e); }
}

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
            // Draw path
            L.polyline(points, {
                color: '#00ff41',
                weight: 3,
                opacity: 0.8
            }).addTo(pathLayer);
            
            // Add markers
            data.hops.filter(h => h.lat && h.lon).forEach((h, i) => {
                const marker = L.circleMarker([h.lat, h.lon], {
                    radius: i === 0 ? 10 : (i === data.hops.length - 1 ? 10 : 6),
                    color: i === 0 ? '#00ff41' : (i === data.hops.length - 1 ? '#ff4444' : '#ffaa00'),
                    fillColor: i === 0 ? '#00ff41' : (i === data.hops.length - 1 ? '#ff4444' : '#ffaa00'),
                    fillOpacity: 0.8
                }).addTo(markersLayer);
                
                marker.bindPopup(`
                    <b>Hop ${h.hop_num}</b><br>
                    IP: ${h.ip}<br>
                    Latency: ${h.latency_ms.toFixed(1)} ms<br>
                    ${h.city ? `Location: ${h.city}, ${h.country}` : ''}
                `);
            });
            
            // Fit bounds
            map.fitBounds(points, { padding: [50, 50] });
        }
    } catch(e) { console.error('Show trace on map error:', e); }
}

// ===== Countdown =====
function startCountdown() {
    let countdown = 60;
    setInterval(() => {
        const el = document.getElementById('countdown');
        if (el) {
            el.textContent = --countdown;
            if (countdown <= 0) location.reload();
        }
    }, 1000);
}
