function initResultsWebSocket(photoCount) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = protocol + '//' + window.location.host + '/ws/results';
    const statusEl = document.getElementById('wsStatus');

    let ws;
    let reconnectDelay = 1000;

    function connect() {
        ws = new WebSocket(wsUrl);

        ws.onopen = function () {
            statusEl.textContent = '● Live';
            statusEl.className = 'ws-status connected';
            reconnectDelay = 1000;
        };

        ws.onmessage = function (event) {
            try {
                const data = JSON.parse(event.data);
                updateResults(data);
            } catch (e) {
                console.error('Failed to parse WebSocket message:', e);
            }
        };

        ws.onclose = function () {
            statusEl.textContent = '○ Reconnecting...';
            statusEl.className = 'ws-status disconnected';
            setTimeout(connect, reconnectDelay);
            reconnectDelay = Math.min(reconnectDelay * 2, 10000);
        };

        ws.onerror = function () {
            ws.close();
        };
    }

    function updateResults(data) {
        const total = data.total || 0;
        document.getElementById('totalVotes').textContent = total;

        // Build a map of photo_id -> count
        const countMap = {};
        if (data.counts) {
            data.counts.forEach(function (c) {
                countMap[c.photo_id] = c.vote_count;
            });
        }

        // Update each result row
        document.querySelectorAll('.result-row').forEach(function (row) {
            const photoId = row.dataset.photoId;
            const count = countMap[photoId] || 0;
            const pct = total > 0 ? (count / total * 100) : 0;

            const bar = row.querySelector('.result-bar');
            const countEl = row.querySelector('.count-value');

            if (bar) {
                bar.style.width = pct + '%';
                bar.dataset.count = count;
            }
            if (countEl) {
                countEl.textContent = count;
            }
        });
    }

    connect();
}
