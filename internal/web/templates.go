package web

import (
	"html/template"
)

var dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>NetPulse Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/mermaid/dist/mermaid.min.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        
        :root, [data-theme="hacker"] {
            --bg-primary: #0a0f0a;
            --bg-secondary: #0d1a0d;
            --bg-card: rgba(0, 40, 0, 0.4);
            --border-color: #1a4a1a;
            --text-primary: #00ff41;
            --text-secondary: #00cc33;
            --text-dim: #336633;
            --accent: #00ff41;
            --accent-glow: rgba(0, 255, 65, 0.3);
            --danger: #ff3333;
            --success: #00ff41;
            --gradient-top: rgba(0, 50, 0, 0.3);
        }
        
        [data-theme="cyberpunk"] {
            --bg-primary: #0a0a14;
            --bg-secondary: #12121f;
            --bg-card: rgba(30, 20, 60, 0.5);
            --border-color: #4a2a6a;
            --text-primary: #ff00ff;
            --text-secondary: #00ffff;
            --text-dim: #8866aa;
            --accent: #ff00ff;
            --accent-glow: rgba(255, 0, 255, 0.3);
            --danger: #ff3366;
            --success: #00ffff;
            --gradient-top: rgba(80, 0, 120, 0.3);
        }
        
        [data-theme="blood"] {
            --bg-primary: #0f0808;
            --bg-secondary: #1a0c0c;
            --bg-card: rgba(60, 20, 20, 0.5);
            --border-color: #4a1a1a;
            --text-primary: #ff4444;
            --text-secondary: #ff6666;
            --text-dim: #884444;
            --accent: #ff4444;
            --accent-glow: rgba(255, 68, 68, 0.3);
            --danger: #ff0000;
            --success: #44ff44;
            --gradient-top: rgba(80, 0, 0, 0.3);
        }
        
        [data-theme="amber"] {
            --bg-primary: #0c0a06;
            --bg-secondary: #141008;
            --bg-card: rgba(50, 40, 20, 0.5);
            --border-color: #4a3a1a;
            --text-primary: #ffaa00;
            --text-secondary: #ffcc44;
            --text-dim: #886644;
            --accent: #ffaa00;
            --accent-glow: rgba(255, 170, 0, 0.3);
            --danger: #ff4444;
            --success: #00ff44;
            --gradient-top: rgba(80, 60, 0, 0.3);
        }
        
        [data-theme="ocean"] {
            --bg-primary: #060a10;
            --bg-secondary: #0a1018;
            --bg-card: rgba(20, 40, 80, 0.4);
            --border-color: #1a3a6a;
            --text-primary: #00aaff;
            --text-secondary: #44ccff;
            --text-dim: #446688;
            --accent: #00aaff;
            --accent-glow: rgba(0, 170, 255, 0.3);
            --danger: #ff4444;
            --success: #44ff88;
            --gradient-top: rgba(0, 40, 100, 0.3);
        }
        
        body {
            font-family: 'Courier New', monospace;
            background: var(--bg-primary);
            background-image: radial-gradient(ellipse at top, var(--gradient-top) 0%, transparent 50%);
            color: var(--text-primary);
            min-height: 100vh;
            padding: 1.5rem;
            transition: all 0.3s;
        }
        
        @keyframes blink { 0%, 50% { opacity: 1; } 51%, 100% { opacity: 0; } }
        
        .container { max-width: 1400px; margin: 0 auto; }
        
        header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 1.5rem;
            padding-bottom: 1rem;
            border-bottom: 1px solid var(--border-color);
            flex-wrap: wrap;
            gap: 1rem;
        }
        
        h1 {
            font-size: 1.6rem;
            color: var(--accent);
            text-shadow: 0 0 10px var(--accent-glow);
            letter-spacing: 3px;
        }
        
        .theme-switcher { display: flex; gap: 0.4rem; align-items: center; }
        .theme-btn {
            width: 24px; height: 24px;
            border-radius: 50%;
            border: 2px solid rgba(255,255,255,0.2);
            cursor: pointer;
            transition: all 0.2s;
        }
        .theme-btn:hover { transform: scale(1.2); }
        .theme-btn.active { border-color: #fff; }
        .theme-btn[data-theme="hacker"] { background: linear-gradient(135deg, #00ff41, #006622); }
        .theme-btn[data-theme="cyberpunk"] { background: linear-gradient(135deg, #ff00ff, #00ffff); }
        .theme-btn[data-theme="blood"] { background: linear-gradient(135deg, #ff4444, #880000); }
        .theme-btn[data-theme="amber"] { background: linear-gradient(135deg, #ffaa00, #884400); }
        .theme-btn[data-theme="ocean"] { background: linear-gradient(135deg, #00aaff, #003366); }
        
        .tabs { display: flex; gap: 0; margin-bottom: 1rem; border-bottom: 1px solid var(--border-color); overflow-x: auto; }
        .tab {
            padding: 0.6rem 1.2rem;
            background: transparent;
            color: var(--text-dim);
            border: none;
            border-bottom: 2px solid transparent;
            cursor: pointer;
            font-family: inherit;
            text-transform: uppercase;
            letter-spacing: 1px;
            font-size: 0.75rem;
            white-space: nowrap;
        }
        .tab:hover { color: var(--text-secondary); }
        .tab.active { color: var(--accent); border-bottom-color: var(--accent); }
        
        .tab-content { display: none; }
        .tab-content.active { display: block; }
        
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(320px, 1fr)); gap: 1rem; }
        
        .card {
            background: var(--bg-card);
            border: 1px solid var(--border-color);
            padding: 1rem;
            position: relative;
        }
        .card::before {
            content: '';
            position: absolute;
            top: 0; left: 0; right: 0;
            height: 2px;
            background: linear-gradient(90deg, transparent, var(--accent), transparent);
        }
        .card-title {
            font-size: 0.85rem;
            color: var(--text-secondary);
            margin-bottom: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .card-title::before { content: '> '; color: var(--accent); }
        
        .stat-row { display: flex; justify-content: space-between; padding: 0.4rem 0; border-bottom: 1px dashed var(--border-color); }
        .stat-label { color: var(--text-dim); font-size: 0.85rem; }
        .stat-value { color: var(--accent); font-weight: bold; font-size: 0.85rem; }
        
        .status-badge { padding: 0.15rem 0.5rem; font-size: 0.7rem; text-transform: uppercase; }
        .status-running { background: rgba(0,255,100,0.15); color: var(--success); border: 1px solid var(--success); }
        .status-stopped { background: rgba(255,50,50,0.15); color: var(--danger); border: 1px solid var(--danger); }
        
        table { width: 100%; border-collapse: collapse; font-size: 0.8rem; }
        th, td { text-align: left; padding: 0.5rem; }
        th { color: var(--text-secondary); font-weight: normal; text-transform: uppercase; font-size: 0.7rem; border-bottom: 1px solid var(--border-color); }
        td { border-bottom: 1px dashed var(--border-color); }
        
        .btn {
            display: inline-block;
            padding: 0.5rem 1rem;
            background: transparent;
            color: var(--accent);
            text-decoration: none;
            border: 1px solid var(--accent);
            font-family: inherit;
            text-transform: uppercase;
            letter-spacing: 1px;
            font-size: 0.75rem;
            cursor: pointer;
            transition: all 0.2s;
        }
        .btn:hover { background: var(--accent); color: var(--bg-primary); }
        
        .filter-bar { display: flex; gap: 0.75rem; margin-bottom: 1rem; flex-wrap: wrap; align-items: center; }
        .filter-bar select, .filter-bar input {
            background: var(--bg-secondary);
            border: 1px solid var(--border-color);
            color: var(--text-primary);
            padding: 0.4rem 0.6rem;
            font-family: inherit;
            font-size: 0.8rem;
        }
        
        .trace-path {
            background: var(--bg-secondary);
            padding: 0.75rem;
            margin: 0.4rem 0;
            border-left: 2px solid var(--accent);
            font-size: 0.8rem;
        }
        .trace-hop { display: flex; padding: 0.2rem 0; }
        .hop-num { color: var(--text-dim); width: 25px; }
        .hop-ip { color: var(--accent); flex: 1; }
        .hop-latency { color: var(--text-secondary); }
        
        .mermaid { background: var(--bg-secondary); padding: 1rem; border-radius: 4px; overflow-x: auto; }
        .mermaid svg { max-width: 100%; }
        
        #latencyChart { max-height: 300px; }
        
        .pagination { display: flex; gap: 0.5rem; margin-top: 1rem; align-items: center; justify-content: center; flex-wrap: wrap; }
        .pagination button { background: transparent; border: 1px solid var(--border-color); color: var(--text-secondary); padding: 0.3rem 0.6rem; font-family: inherit; font-size: 0.75rem; cursor: pointer; }
        .pagination button:hover { border-color: var(--accent); color: var(--accent); }
        .pagination button.active { background: var(--accent); color: var(--bg-primary); border-color: var(--accent); }
        .pagination button:disabled { opacity: 0.3; cursor: not-allowed; }
        .pagination .page-info { color: var(--text-dim); font-size: 0.75rem; }

        .anomaly-card {
            background: rgba(255, 100, 50, 0.1);
            border: 1px solid var(--danger);
            padding: 0.75rem;
            margin: 0.5rem 0;
            font-size: 0.8rem;
        }
        .anomaly-title { color: var(--danger); font-weight: bold; margin-bottom: 0.5rem; }
        
        .empty-state { text-align: center; padding: 2rem; color: var(--text-dim); font-size: 0.85rem; }
        .footer { text-align: center; color: var(--text-dim); margin-top: 1.5rem; font-size: 0.7rem; }
        .actions { margin-top: 1rem; display: flex; gap: 0.75rem; flex-wrap: wrap; }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>▶ NETPULSE<span style="animation: blink 1s infinite">_</span></h1>
            <div class="theme-switcher">
                <button class="theme-btn active" data-theme="hacker" title="Hacker"></button>
                <button class="theme-btn" data-theme="cyberpunk" title="Cyberpunk"></button>
                <button class="theme-btn" data-theme="blood" title="Blood"></button>
                <button class="theme-btn" data-theme="amber" title="Amber"></button>
                <button class="theme-btn" data-theme="ocean" title="Ocean"></button>
            </div>
        </header>
        
        <div class="tabs">
            <button class="tab active" onclick="showTab('overview')">Overview</button>
            <button class="tab" onclick="showTab('topology')">Topology</button>
            <button class="tab" onclick="showTab('traces')">Traces</button>
            <button class="tab" onclick="showTab('latency')">Latency</button>
            <button class="tab" onclick="showTab('anomalies')">Anomalies</button>
            <button class="tab" onclick="showTab('hosts')">Hosts</button>
        </div>
        
        <!-- Overview -->
        <div id="overview" class="tab-content active">
            <div class="grid">
                <div class="card">
                    <div class="card-title">IP Status</div>
                    <div class="stat-row"><span class="stat-label">IP</span><span class="stat-value">{{.current_ip}}</span></div>
                    <div class="stat-row"><span class="stat-label">ISP</span><span class="stat-value">{{.isp}}</span></div>
                    <div class="stat-row"><span class="stat-label">ASN</span><span class="stat-value">{{.asn}}</span></div>
                    <div class="stat-row"><span class="stat-label">Last Check</span><span class="stat-value">{{.last_check}}</span></div>
                </div>
                <div class="card">
                    <div class="card-title">Statistics</div>
                    <div class="stat-row">
                        <span class="stat-label">Daemon</span>
                        <span>{{if .daemon_running}}<span class="status-badge status-running">Online</span>{{else}}<span class="status-badge status-stopped">Offline</span>{{end}}</span>
                    </div>
                    <div class="stat-row"><span class="stat-label">IP Records</span><span class="stat-value">{{.ip_count}}</span></div>
                    <div class="stat-row"><span class="stat-label">Alive Hosts</span><span class="stat-value">{{.host_count}}</span></div>
                    <div class="stat-row"><span class="stat-label">Open Ports</span><span class="stat-value">{{.port_count}}</span></div>
                </div>
            </div>
            <div class="card" style="margin-top: 1rem;">
                <div class="card-title">Discovered Hosts</div>
                {{if .hosts}}
                <table>
                    <thead><tr><th>IP</th><th>Hostname</th><th>Latency</th><th>Last Seen</th></tr></thead>
                    <tbody>{{range .hosts}}<tr><td>{{.IP}}</td><td>{{if .Hostname}}{{.Hostname}}{{else}}-{{end}}</td><td>{{printf "%.1f" .LatencyMs}} ms</td><td>{{.LastSeen.Format "15:04:05"}}</td></tr>{{end}}</tbody>
                </table>
                {{else}}<p class="empty-state">> No hosts discovered</p>{{end}}
            </div>
        </div>
        
        <!-- Topology -->
        <div id="topology" class="tab-content">
            <div class="card">
                <div class="card-title">Network Topology</div>
                <div class="filter-bar">
                    <select id="topologyTarget" onchange="loadTopology()">
                        <option value="">All Targets</option>
                        {{range .trace_targets}}<option value="{{.}}">{{.}}</option>{{end}}
                    </select>
                    <button class="btn" onclick="loadTopology()">Refresh</button>
                </div>
                <div id="topologyDiagram" class="mermaid">graph LR
    Source[Your Network]
    Loading[Loading...]
    Source --> Loading</div>
            </div>
        </div>
        
        <!-- Traces -->
        <div id="traces" class="tab-content">
            <div class="card">
                <div class="card-title">Traceroute Results</div>
                <div class="filter-bar">
                    <select id="traceFilter" onchange="loadTraces(1)">
                        <option value="">All Targets</option>
                        {{range .trace_targets}}<option value="{{.}}">{{.}}</option>{{end}}
                    </select>
                    <button class="btn" onclick="loadTraces(1)">Refresh</button>
                </div>
                <div id="traceResults"><p class="empty-state">> Loading traces...</p></div>
                <div id="tracePagination" class="pagination"></div>
            </div>
        </div>
        
        <!-- Latency -->
        <div id="latency" class="tab-content">
            <div class="card">
                <div class="card-title">Latency Trends</div>
                <div class="filter-bar">
                    <select id="latencyTarget" onchange="loadLatencyChart()">
                        <option value="">All Targets</option>
                        {{range .trace_targets}}<option value="{{.}}">{{.}}</option>{{end}}
                    </select>
                </div>
                <canvas id="latencyChart"></canvas>
            </div>
        </div>
        
        <!-- Anomalies -->
        <div id="anomalies" class="tab-content">
            <div class="card">
                <div class="card-title">Route Changes & Anomalies</div>
                <div id="anomalyList"><p class="empty-state">> Loading anomalies...</p></div>
                <div id="anomalyPagination" class="pagination"></div>
            </div>
        </div>
        
        <!-- Hosts -->
        <div id="hosts" class="tab-content">
            <div class="card">
                <div class="card-title">Network Scan</div>
                <div class="filter-bar">
                    <input type="text" id="hostFilter" placeholder="Filter by IP..." oninput="filterHosts()">
                </div>
                {{if .hosts}}
                <table id="hostTable">
                    <thead><tr><th>IP</th><th>Hostname</th><th>Status</th><th>Latency</th><th>Last Seen</th></tr></thead>
                    <tbody>{{range .hosts}}<tr><td>{{.IP}}</td><td>{{if .Hostname}}{{.Hostname}}{{else}}-{{end}}</td><td><span class="status-badge status-running">Alive</span></td><td>{{printf "%.1f" .LatencyMs}} ms</td><td>{{.LastSeen.Format "15:04:05"}}</td></tr>{{end}}</tbody>
                </table>
                {{else}}<p class="empty-state">> No hosts</p>{{end}}
            </div>
        </div>
        
        <div class="actions">
            <a href="/report" class="btn">Download Report</a>
            <button class="btn" onclick="location.reload()">Refresh</button>
        </div>
        <div class="footer">NETPULSE v1.0 | Auto-refresh in <span id="countdown">60</span>s</div>
    </div>
    
    <script>
        mermaid.initialize({ startOnLoad: false, theme: 'dark', themeVariables: { primaryColor: '#00ff41', primaryTextColor: '#fff', primaryBorderColor: '#00ff41', lineColor: '#00ff41', background: '#0a0f0a' }});
        
        let latencyChart = null;
        
        // Theme switching
        document.querySelectorAll('.theme-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                document.documentElement.setAttribute('data-theme', btn.dataset.theme);
                document.querySelectorAll('.theme-btn').forEach(b => b.classList.remove('active'));
                btn.classList.add('active');
                localStorage.setItem('netpulse-theme', btn.dataset.theme);
            });
        });
        const savedTheme = localStorage.getItem('netpulse-theme') || 'hacker';
        document.documentElement.setAttribute('data-theme', savedTheme);
        document.querySelectorAll('.theme-btn').forEach(b => b.classList.toggle('active', b.dataset.theme === savedTheme));
        
        function showTab(tabId) {
            document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
            document.querySelectorAll('.tab-content').forEach(c => c.classList.remove('active'));
            event.target.classList.add('active');
            document.getElementById(tabId).classList.add('active');
            
            if (tabId === 'topology') loadTopology();
            if (tabId === 'traces') loadTraces(1);
            if (tabId === 'latency') loadLatencyChart();
            if (tabId === 'anomalies') loadAnomalies();
        }
        
        function filterHosts() {
            const filter = document.getElementById('hostFilter').value.toLowerCase();
            document.querySelectorAll('#hostTable tbody tr').forEach(row => {
                row.style.display = row.cells[0].textContent.toLowerCase().includes(filter) ? '' : 'none';
            });
        }
        
        let currentTracePage = 1;
        
        async function loadTraces(page = 1) {
            currentTracePage = page;
            const target = document.getElementById('traceFilter')?.value || '';
            const limit = 10;
            
            try {
                const res = await fetch('/api/traces?page=' + page + '&limit=' + limit + '&target=' + encodeURIComponent(target));
                const data = await res.json();
                
                const container = document.getElementById('traceResults');
                const pagination = document.getElementById('tracePagination');
                
                if (!data.traces || data.traces.length === 0) {
                    container.innerHTML = '<p class="empty-state">> No traces found</p>';
                    pagination.innerHTML = '';
                    return;
                }
                
                container.innerHTML = data.traces.map(t => ` + "`" + `
                    <div class="trace-path">
                        <strong>> ${t.target}</strong> @ ${new Date(t.timestamp).toLocaleTimeString()}
                        ${(t.hops || []).map(h => ` + "`" + `
                            <div class="trace-hop">
                                <span class="hop-num">${h.hop_num}.</span>
                                <span class="hop-ip">${h.lost ? '* * *' : h.ip}</span>
                                <span class="hop-latency">${h.lost ? '' : h.latency_ms.toFixed(1) + ' ms'}</span>
                            </div>
                        ` + "`" + `).join('')}
                    </div>
                ` + "`" + `).join('');
                
                // Render pagination
                let paginationHTML = '<span class="page-info">Page ' + data.page + ' of ' + data.total_pages + ' (' + data.total + ' total)</span>';
                paginationHTML += '<button onclick="loadTraces(1)" ' + (page <= 1 ? 'disabled' : '') + '>«</button>';
                paginationHTML += '<button onclick="loadTraces(' + (page - 1) + ')" ' + (page <= 1 ? 'disabled' : '') + '>‹</button>';
                
                // Show page numbers
                for (let i = Math.max(1, page - 2); i <= Math.min(data.total_pages, page + 2); i++) {
                    paginationHTML += '<button onclick="loadTraces(' + i + ')" class="' + (i === page ? 'active' : '') + '">' + i + '</button>';
                }
                
                paginationHTML += '<button onclick="loadTraces(' + (page + 1) + ')" ' + (page >= data.total_pages ? 'disabled' : '') + '>›</button>';
                paginationHTML += '<button onclick="loadTraces(' + data.total_pages + ')" ' + (page >= data.total_pages ? 'disabled' : '') + '>»</button>';
                
                pagination.innerHTML = paginationHTML;
            } catch(e) { console.error('Traces error:', e); }
        }
        
        async function loadTopology() {
            const target = document.getElementById('topologyTarget')?.value || '';
            try {
                const res = await fetch('/api/analytics/mermaid?target=' + encodeURIComponent(target));
                const diagram = await res.text();
                const el = document.getElementById('topologyDiagram');
                el.innerHTML = diagram;
                el.removeAttribute('data-processed');
                mermaid.init(undefined, el);
            } catch(e) { console.error('Topology error:', e); }
        }
        
        async function loadLatencyChart() {
            const target = document.getElementById('latencyTarget')?.value || '';
            try {
                const res = await fetch('/api/analytics/latency?target=' + encodeURIComponent(target));
                const data = await res.json();
                
                if (latencyChart) latencyChart.destroy();
                
                const grouped = {};
                (data || []).forEach(p => {
                    if (!grouped[p.target]) grouped[p.target] = [];
                    grouped[p.target].push({ x: new Date(p.timestamp), y: p.latency_ms });
                });
                
                const datasets = Object.keys(grouped).map((t, i) => ({
                    label: t,
                    data: grouped[t],
                    borderColor: ['#00ff41', '#ff00ff', '#ff4444', '#ffaa00', '#00aaff'][i % 5],
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
        
        async function loadAnomalies() {
            try {
                const res = await fetch('/api/analytics/anomalies');
                const data = await res.json();
                const el = document.getElementById('anomalyList');
                
                if (!data || data.length === 0) {
                    el.innerHTML = '<p class="empty-state">> No route changes detected in last 24h</p>';
                    return;
                }
                
                el.innerHTML = data.map(a => ` + "`" + `
                    <div class="anomaly-card">
                        <div class="anomaly-title">⚠ Route Change to ${a.target}</div>
                        <div>Detected: ${new Date(a.detected_at).toLocaleString()}</div>
                        <div>Changed hops: ${a.changed_hops.join(', ')}</div>
                        <div style="margin-top:0.5rem;font-size:0.75rem;color:var(--text-dim)">
                            Old: ${a.old_path.slice(0,5).join(' → ')}${a.old_path.length > 5 ? '...' : ''}<br>
                            New: ${a.new_path.slice(0,5).join(' → ')}${a.new_path.length > 5 ? '...' : ''}
                        </div>
                    </div>
                ` + "`" + `).join('');
            } catch(e) { console.error('Anomalies error:', e); }
        }
        
        // Countdown
        let countdown = 60;
        setInterval(() => {
            document.getElementById('countdown').textContent = --countdown;
            if (countdown <= 0) location.reload();
        }, 1000);
        
        // Initial load
        setTimeout(loadTopology, 500);
    </script>
    <script src="https://cdn.jsdelivr.net/npm/chartjs-adapter-date-fns"></script>
</body>
</html>`

func getDashboardTemplate() *template.Template {
	return template.Must(template.New("dashboard").Parse(dashboardHTML))
}
