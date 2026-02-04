/* DNS Monitoring Logic */

let dnsChart = null;
let dnsInterval = null;

function initDNS() {
    const ctx = document.getElementById('dnsChart');
    if (!ctx) return;

    if (dnsChart) {
        dnsChart.destroy();
    }

    dnsChart = new Chart(ctx, {
        type: 'line',
        data: { datasets: [] },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: false,
            interaction: {
                mode: 'index',
                intersect: false,
            },
            plugins: {
                legend: {
                    position: 'top',
                    labels: { color: '#888' }
                },
                tooltip: {
                    callbacks: {
                        label: function (context) {
                            return context.dataset.label + ': ' + context.parsed.y + ' ms';
                        }
                    }
                }
            },
            scales: {
                x: {
                    type: 'time',
                    time: { unit: 'minute' },
                    grid: { color: '#333' },
                    ticks: { color: '#888' }
                },
                y: {
                    beginAtZero: true,
                    grid: { color: '#333' },
                    ticks: { color: '#888' },
                    title: { display: true, text: 'Latency (ms)', color: '#888' }
                }
            }
        }
    });

    updateDNS();
    if (dnsInterval) clearInterval(dnsInterval);
    dnsInterval = setInterval(updateDNS, 10000);

    // Load Targets
    loadDNSTargets();
}

async function updateDNS() {
    try {
        let params = 'limit=100';
        // Use global time filter if available
        if (window.getGlobalTimeParams) {
            params = window.getGlobalTimeParams();
        }

        const res = await fetch('/api/dns/history?' + params);
        const data = await res.json();

        if (!data) return;

        // Group by Server+Protocol and Server for Integrity
        const grouped = {};
        const integrity = {}; // { "Google": { udp: {ip, lat}, doh: {ip, lat} } }

        data.forEach(d => {
            // Chart Data
            const key = `${d.server} [${d.protocol.toUpperCase()}]`;
            if (!grouped[key]) grouped[key] = [];
            grouped[key].push({
                x: d.timestamp,
                y: d.latency_ms
            });

            // Integrity Data (Only latest)
            if (!integrity[d.server]) integrity[d.server] = {};
            const ts = new Date(d.timestamp).getTime();
            // Store if newer
            if (!integrity[d.server][d.protocol] || ts > integrity[d.server][d.protocol].ts) {
                integrity[d.server][d.protocol] = {
                    ip: d.resolved_ip,
                    lat: d.latency_ms,
                    ts: ts
                };
            }
        });

        updateChart(grouped);
        renderIntegrityTable(integrity);

    } catch (e) {
        console.error('DNS update failed', e);
    }
}

function updateChart(grouped) {
    const datasets = Object.keys(grouped).map((key) => {
        const isDoH = key.includes('DOH');
        const color = getDNSColor(key);

        return {
            label: key,
            data: grouped[key],
            borderColor: color,
            backgroundColor: color,
            borderWidth: isDoH ? 2 : 1,
            borderDash: isDoH ? [] : [5, 5],
            pointRadius: 0,
            tension: 0.4
        };
    });

    dnsChart.data.datasets = datasets;
    dnsChart.update('none');
}

function renderIntegrityTable(data) {
    const tbody = document.getElementById('dnsIntegrityBody');
    if (!tbody) return;

    tbody.innerHTML = Object.keys(data).map(server => {
        const udp = data[server].udp || { ip: 'Pending...', lat: '-' };
        const doh = data[server].doh || { ip: 'Pending...', lat: '-' };

        let status = '<span class="status-badge status-running">Secure</span>';
        let rowClass = '';

        if (udp.ip && doh.ip && udp.ip !== doh.ip) {
            status = '<span class="status-badge status-stopped">MISMATCH</span>';
            rowClass = 'style="background: rgba(255,0,0,0.1)"';
        } else if (!udp.ip || !doh.ip) {
            status = '<span class="status-badge" style="background:#555">Checking</span>';
        }

        return `
        <tr ${rowClass}>
            <td>${server}</td>
            <td>${udp.ip} <span class="latency-sm">(${udp.lat}ms)</span></td>
            <td>${doh.ip} <span class="latency-sm">(${doh.lat}ms)</span></td>
            <td>${status}</td>
        </tr>
        `;
    }).join('');
}

function getDNSColor(key) {
    if (key.includes('Google')) return '#4285F4';
    if (key.includes('Cloudflare')) return '#F48120';
    if (key.includes('Quad9')) return '#9A2425';
    // Hash string for random stable color
    let hash = 0;
    for (let i = 0; i < key.length; i++) {
        hash = key.charCodeAt(i) + ((hash << 5) - hash);
    }
    const c = (hash & 0x00FFFFFF).toString(16).toUpperCase();
    return '#' + "00000".substring(0, 6 - c.length) + c;
}

// ===== Target Management =====

async function loadDNSTargets() {
    // Populate list in modal or separate UI
    const res = await fetch('/api/dns/targets');
    const targets = await res.json();
    const list = document.getElementById('targetList');
    if (!list) return;

    if (!targets || targets.length === 0) {
        list.innerHTML = '<p class="empty-state">No custom targets</p>';
        return;
    }

    list.innerHTML = targets.map(t => `
        <div class="target-item" id="target-${t.id}">
            <div class="target-info">
                <strong>${t.name}</strong><br>
                <small>UDP: ${t.ip} | DoH: ${t.doh_url || 'N/A'}</small>
            </div>
            <button onclick="deleteDNSTarget(${t.id}, this)" class="btn-icon" title="Delete Monitor" style="background:none; border:none; cursor:pointer; padding:5px; opacity:0.7; transition:opacity 0.2s">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="#ff4444" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <polyline points="3 6 5 6 21 6"></polyline>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
                </svg>
            </button>
        </div>
    `).join('');
}

async function addDNSTarget() {
    const name = document.getElementById('newTargetName').value;
    const ip = document.getElementById('newTargetIP').value;
    const doh = document.getElementById('newTargetDoH').value;

    if (!name || (!ip && !doh)) {
        alert("Name and at least one address required");
        return;
    }

    const btn = document.querySelector('button[onclick="addDNSTarget()"]');
    if (btn) { btn.innerText = 'Adding...'; btn.disabled = true; }

    try {
        await fetch('/api/dns/targets', {
            method: 'POST',
            body: JSON.stringify({ name, ip, doh_url: doh })
        });

        document.getElementById('newTargetName').value = '';
        document.getElementById('newTargetIP').value = '';
        document.getElementById('newTargetDoH').value = '';
        await loadDNSTargets();
    } finally {
        if (btn) { btn.innerText = 'Add Target'; btn.disabled = false; }
    }
}

async function deleteDNSTarget(id, btn) {
    if (!confirm('Remove this monitor?')) return;

    // Visual feedback
    if (btn) {
        btn.innerHTML = '<span style="font-size:12px; color:#ff4444">...</span>';
        btn.disabled = true;
    }
    const item = document.getElementById(`target-${id}`);
    if (item) item.style.opacity = '0.5';

    await fetch(`/api/dns/targets/${id}`, { method: 'DELETE' });
    loadDNSTargets();
}

// Auto-init
document.addEventListener('DOMContentLoaded', () => {
    window.initDNS = initDNS;
});
