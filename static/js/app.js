// ─── State ───────────────────────────────────────────
// Credentials now stored in DB, loaded from server
const state = {
    domain: document.getElementById('domain')?.value?.trim() || '',
};

// ─── DOM refs ────────────────────────────────────────
const $ = (s, p = document) => p.querySelector(s);
const $$ = (s, p = document) => [...p.querySelectorAll(s)];

// ─── Theme Toggle ──────────────────────────────────────
const themeToggle = document.getElementById('theme-toggle');
const htmlEl = document.documentElement;

// Check saved theme
const savedTheme = localStorage.getItem('theme');
if (savedTheme === 'dark') {
    htmlEl.classList.add('dark-mode');
    if (themeToggle) themeToggle.innerHTML = '<i class="bx bx-sun bx-sm"></i>';
}

if (themeToggle) {
    themeToggle.addEventListener('click', () => {
        htmlEl.classList.toggle('dark-mode');
        const isDark = htmlEl.classList.contains('dark-mode');
        localStorage.setItem('theme', isDark ? 'dark' : 'light');
        themeToggle.innerHTML = isDark ? '<i class="bx bx-sun bx-sm"></i>' : '<i class="bx bx-moon bx-sm"></i>';
    });
}

// ─── Credentials Form ────────────────────────────────
const credForm = document.getElementById('form-credentials');
const credResult = document.getElementById('cred-result');

if (credForm) {
    credForm.addEventListener('submit', async e => {
        e.preventDefault();
        const data = {
            api_token: document.getElementById('api_token').value.trim(),
            zone_id: document.getElementById('zone_id').value.trim(),
            domain: document.getElementById('domain').value.trim(),
            vps_ip: document.getElementById('vps_ip').value.trim(),
            ssh_host: document.getElementById('ssh_host')?.value?.trim() || '',
            ssh_port: document.getElementById('ssh_port')?.value?.trim() || '22',
            ssh_user: document.getElementById('ssh_user')?.value?.trim() || 'root',
            ssh_password: document.getElementById('ssh_password')?.value || '',
        };

        if (!data.api_token || !data.zone_id || !data.domain) {
            showResult(credResult, 'error', 'API Token, Zone ID, dan Domain wajib diisi.');
            return;
        }

        state.domain = data.domain;
        showResult(credResult, 'info', 'Memverifikasi & menyimpan kredensial...');

        try {
            const res = await fetch('/check-config', {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: new URLSearchParams(data),
            });
            const json = await res.json();
            if (json.success) {
                showResult(credResult, 'success', `✅ Kredensial valid & tersimpan! Zone: ${json.zone_name}`);
                updateConnStatus(true);
            } else {
                showResult(credResult, 'error', `❌ ${json.error || 'Kredensial tidak valid'}`);
                updateConnStatus(false);
            }
        } catch (err) {
            showResult(credResult, 'error', `❌ Network error: ${err.message}`);
            updateConnStatus(false);
        }
    });
}

// ─── Connection Status ───────────────────────────────
function updateConnStatus(connected) {
    const el = document.getElementById('conn-status');
    if (connected) {
        el.className = 'badge bg-label-success';
        el.textContent = '● Connected';
    } else {
        el.className = 'badge bg-label-secondary';
        el.textContent = '⏻ Disconnected';
    }
}

// ─── Setup Fallback ──────────────────────────────────
const btnFallback = document.getElementById('btn-setup-fallback');
const fallbackLog = document.getElementById('fallback-log');

if (btnFallback) {
    btnFallback.addEventListener('click', async () => {
        const vpsIp = document.getElementById('vps_ip')?.value?.trim() || '';
        if (!vpsIp) {
            alert('Isi IP VPS terlebih dahulu di menu Credentials.');
            window.location.href = '/credentials';
            return;
        }

        btnFallback.disabled = true;
        btnFallback.innerHTML = '<span class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span> Memproses...';
        fallbackLog.classList.remove('d-none');
        fallbackLog.innerHTML = '<div class="log-entry info"><div class="log-step">Memulai setup fallback...</div></div>';

        try {
            const res = await fetch('/setup-fallback', {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: new URLSearchParams({ vps_ip: vpsIp }),
            });
            const json = await res.json();
            fallbackLog.innerHTML = '';

            if (json.success && json.steps) {
                json.steps.forEach(step => {
                    const isOk = step.result?.success || step.result?.skipped;
                    const div = document.createElement('div');
                    div.className = `log-entry ${isOk ? 'success' : 'error'}`;
                    div.innerHTML = `
                        <div class="log-step">${step.step}</div>
                        <div class="log-msg">${isOk ? '✅ Berhasil' : '❌ Gagal'} ${step.result?.skipped ? '(sudah ada)' : ''}</div>
                    `;
                    div.addEventListener('click', () => openLog(JSON.stringify(step.result, null, 2)));
                    div.style.cursor = 'pointer';
                    fallbackLog.appendChild(div);
                });
            } else {
                fallbackLog.innerHTML = `<div class="log-entry error"><div class="log-step">Error</div><div class="log-msg">${json.error || 'Gagal'}</div></div>`;
            }
        } catch (err) {
            fallbackLog.innerHTML = `<div class="log-entry error"><div class="log-step">Network Error</div><div class="log-msg">${err.message}</div></div>`;
        }

        btnFallback.disabled = false;
        btnFallback.innerHTML = '<i class="bx bx-rocket me-1"></i> Jalankan Setup Fallback';
    });
}

// ─── Create Hostname ─────────────────────────────────
const hostnameForm = document.getElementById('form-hostname');
const hostnameResult = document.getElementById('hostname-result');

if (hostnameForm) {
    hostnameForm.addEventListener('submit', async e => {
        e.preventDefault();
        const targetDomain = document.getElementById('target_domain').value.trim();
        const method = document.getElementById('method').value;

        if (!targetDomain) {
            showResult(hostnameResult, 'error', 'Target domain wajib diisi.');
            return;
        }

        const domain = state.domain || document.getElementById('domain')?.value?.trim() || '';
        showResult(hostnameResult, 'info', `Membuat hostname ${targetDomain}.${domain}...`);

        try {
            const res = await fetch('/create-hostname', {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: new URLSearchParams({ target_domain: targetDomain, method }),
            });
            const json = await res.json();

            if (json.success) {
                const r = json.result;
                if (r.skipped) {
                    showResult(hostnameResult, 'success', `⏭️ Hostname sudah ada: ${r.hostname}\nStatus: ${r.existing?.status || 'unknown'}\nSSL: ${r.existing?.ssl?.status || 'unknown'}`);
                } else if (r.success) {
                    const hostname = r.result?.hostname || `${targetDomain}.${domain}`;
                    const id = r.result?.id || '';
                    const sslStatus = r.result?.ssl?.status || 'pending_validation';

                    showResult(hostnameResult, 'info', `✅ Hostname dibuat: ${hostname}\nSSL: ${sslStatus}\n🔐 Mengambil validation records...`);

                    // Fetch detail hostname untuk dapatkan validation_records
                    if (method === 'http' && id) {
                        let validationRecords = [];
                        for (let attempt = 0; attempt < 3; attempt++) {
                            await new Promise(resolve => setTimeout(resolve, 3000));
                            try {
                                const statusRes = await fetch('/check-status', {
                                    method: 'POST',
                                    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                                    body: new URLSearchParams({ hostname_id: id }),
                                });
                                const statusJson = await statusRes.json();
                                validationRecords = statusJson?.result?.ssl?.validation_records || [];
                                if (validationRecords.length > 0) break;
                            } catch (e) { /* retry */ }
                        }

                        if (validationRecords.length > 0) {
                            let uploaded = 0;
                            let failed = 0;
                            let lastError = '';

                            for (const record of validationRecords) {
                                const urlParts = (record.http_url || '').split('/');
                                const token = urlParts[urlParts.length - 1];
                                const body = record.http_body || '';

                                if (token && body) {
                                    try {
                                        const challengeRes = await fetch('/create-challenge', {
                                            method: 'POST',
                                            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                                            body: new URLSearchParams({ challenge_token: token, challenge_body: body }),
                                        });
                                        const challengeJson = await challengeRes.json();
                                        if (challengeJson.success) {
                                            uploaded++;
                                        } else {
                                            failed++;
                                            lastError = challengeJson.error || 'Unknown error';
                                        }
                                    } catch (e) {
                                        failed++;
                                        lastError = e.message;
                                    }
                                }
                            }

                            if (uploaded === validationRecords.length) {
                                showResult(hostnameResult, 'success', `✅ Hostname: ${hostname}\n🔐 ${uploaded} challenge file(s) uploaded ke VPS\n⏳ Tunggu 1-2 menit hingga certificate active`);
                            } else {
                                showResult(hostnameResult, 'warning', `⚠️ Hostname dibuat, ${uploaded}/${validationRecords.length} challenge uploaded\n${failed > 0 ? 'Error: ' + lastError : ''}`);
                            }
                        } else {
                            showResult(hostnameResult, 'success', `✅ Hostname: ${hostname}\n⏳ Tunggu 2-5 menit hingga certificate active`);
                        }
                    } else {
                        showResult(hostnameResult, 'success', `✅ Hostname: ${hostname}\n⏳ Tunggu 2-5 menit hingga certificate active`);
                    }
                } else {
                    showResult(hostnameResult, 'error', `❌ Gagal: ${r.error || JSON.stringify(r)}`);
                }
            } else {
                showResult(hostnameResult, 'error', `❌ ${json.error || 'Gagal'}`);
            }
        } catch (err) {
            showResult(hostnameResult, 'error', `❌ Network error: ${err.message}`);
        }
    });
}

// ─── List Hostnames ──────────────────────────────────
const btnRefreshHostnames = document.getElementById('btn-refresh-hostnames');
const hostnamesList = document.getElementById('hostnames-list');
// const filterInput = document.getElementById('filter-hostname'); // Not implemented in UI yet

async function loadHostnames() {
    hostnamesList.innerHTML = '<div class="text-center p-3"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Loading...</span></div></div>';

    try {
        const res = await fetch('/list-hostnames', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        });
        const json = await res.json();

        if (!json.success || !json.result || json.result.length === 0) {
            hostnamesList.innerHTML = '<p class="text-muted p-3 mb-0">Belum ada custom hostname.</p>';
            return;
        }

        // const filter = (filterInput.value || '').toLowerCase();
        let items = json.result;
        // if (filter) {
        //     items = items.filter(h =>
        //         h.hostname?.toLowerCase().includes(filter) ||
        //         h.id?.toLowerCase().includes(filter)
        //     );
        // }

        let html = '<table class="table table-hover"><thead><tr><th>Hostname</th><th>Status</th><th>SSL Status</th><th>Aksi</th></tr></thead><tbody class="table-border-bottom-0">';
        items.forEach(h => {
            const sslStatus = h.ssl?.status || 'N/A';
            const statusBadge = statusLabel(h.status);
            const sslBadge = statusLabel(sslStatus);
            html += `<tr>
                <td><strong>${escHtml(h.hostname)}</strong></td>
                <td>${statusBadge}</td>
                <td>${sslBadge}</td>
                <td>
                    <button class="btn btn-sm btn-outline-secondary me-1" onclick="checkHostname('${h.id}')"><i class="bx bx-search"></i> Cek</button>
                    <button class="btn btn-sm btn-outline-danger" onclick="deleteHostname('${h.id}')"><i class="bx bx-trash"></i></button>
                </td>
            </tr>`;
        });
        html += '</tbody></table>';
        hostnamesList.innerHTML = html;
    } catch (err) {
        hostnamesList.innerHTML = `<p class="text-danger p-3 mb-0">Error: ${err.message}</p>`;
    }
}

function statusLabel(status) {
    const map = {
        active: 'bg-label-success',
        pending_validation: 'bg-label-warning',
        pending: 'bg-label-warning',
        pending_issuance: 'bg-label-warning',
        error: 'bg-label-danger',
        deleted: 'bg-label-info',
    };
    return `<span class="badge ${map[status] || 'bg-label-info'}">${status}</span>`;
}

function escHtml(s) {
    const d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
}

if (btnRefreshHostnames) {
    btnRefreshHostnames.addEventListener('click', loadHostnames);
}
// filterInput.addEventListener('input', loadHostnames);

// ─── Check / Delete Hostname ─────────────────────────
async function checkHostname(id) {
    try {
        const res = await fetch('/check-status', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: new URLSearchParams({ hostname_id: id }),
        });
        const json = await res.json();
        openLog(JSON.stringify(json, null, 2));
    } catch (err) {
        openLog(`Error: ${err.message}`);
    }
}

async function deleteHostname(id) {
    if (!confirm('Yakin ingin menghapus custom hostname ini?')) return;
    try {
        const res = await fetch('/delete-hostname', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: new URLSearchParams({ hostname_id: id }),
        });
        const json = await res.json();
        if (json.success) {
            loadHostnames();
            const dnsMsg = json.dns_deleted?.length > 0
                ? `\n🗑️ DNS records dihapus: ${json.dns_deleted.join(', ')}`
                : '';
            openLog(`✅ Hostname berhasil dihapus.${dnsMsg}`);
        } else {
            openLog(`❌ Gagal: ${json.error || JSON.stringify(json)}`);
        }
    } catch (err) {
        openLog(`Error: ${err.message}`);
    }
}

// ─── Bulk Create Hostnames ───────────────────────────
const bulkForm = document.getElementById('form-bulk');
const bulkLog = document.getElementById('bulk-log');

if (bulkForm) {
    bulkForm.addEventListener('submit', async e => {
        e.preventDefault();
        const btnBulk = bulkForm.querySelector('button[type="submit"]');

        const raw = document.getElementById('bulk_domains').value.trim();
        const method = document.getElementById('bulk_method').value;
        if (!raw) {
            alert('Masukkan minimal satu domain.');
            return;
        }

        const domains = raw.split('\n')
            .map(d => d.trim())
            .filter(d => d.length > 0);

        if (domains.length === 0) {
            alert('Tidak ada domain yang valid.');
            return;
        }

        btnBulk.disabled = true;
        btnBulk.innerHTML = '<span class="spinner-border spinner-border-sm me-1" role="status" aria-hidden="true"></span> Memproses...';
        bulkLog.classList.remove('d-none');
        bulkLog.innerHTML = '';

        const total = domains.length;
        let done = 0;
        let success = 0;
        let failed = 0;

        for (const domain of domains) {
            done++;
            addBulkLog('info', `[${done}/${total}] ${domain}`, 'Membuat hostname...');

            try {
                const res = await fetch('/create-hostname', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                    body: new URLSearchParams({ target_domain: domain, method }),
                });
                const json = await res.json();

                if (json.success) {
                    const r = json.result;
                    if (r.skipped) {
                        addBulkLog('success', `[${done}/${total}] ${domain}`, '⏭️ Sudah ada');
                        success++;
                    } else if (r.success) {
                        const hostname = r.result?.hostname || `${domain}.${state.domain}`;
                        const id = r.result?.id || '';

                        // Fetch validation records + upload challenge via SSH
                        let challengeUploaded = 0;
                        if (method === 'http' && id) {
                            await new Promise(resolve => setTimeout(resolve, 2000));
                            try {
                                const statusRes = await fetch('/check-status', {
                                    method: 'POST',
                                    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                                    body: new URLSearchParams({ hostname_id: id }),
                                });
                                const statusJson = await statusRes.json();
                                const records = statusJson?.result?.ssl?.validation_records || [];

                                for (const rec of records) {
                                    const urlParts = (rec.http_url || '').split('/');
                                    const token = urlParts[urlParts.length - 1];
                                    const body = rec.http_body || '';
                                    if (token && body) {
                                        try {
                                            const cr = await fetch('/create-challenge', {
                                                method: 'POST',
                                                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                                                body: new URLSearchParams({ challenge_token: token, challenge_body: body }),
                                            });
                                            const cj = await cr.json();
                                            if (cj.success) challengeUploaded++;
                                        } catch (e) {}
                                    }
                                }
                            } catch (e) {}
                        }

                        const msg = challengeUploaded > 0
                            ? `✅ ${hostname} — ${challengeUploaded} challenge(s) uploaded`
                            : `✅ ${hostname}`;
                        addBulkLog('success', `[${done}/${total}] ${domain}`, msg);
                        success++;
                    } else {
                        addBulkLog('error', `[${done}/${total}] ${domain}`, `❌ ${r.error || 'Gagal'}`);
                        failed++;
                    }
                } else {
                    addBulkLog('error', `[${done}/${total}] ${domain}`, `❌ ${json.error || 'Gagal'}`);
                    failed++;
                }
            } catch (err) {
                addBulkLog('error', `[${done}/${total}] ${domain}`, `❌ ${err.message}`);
                failed++;
            }
        }

        // Done
        addBulkLog(failed > 0 ? 'warning' : 'success', 'SELESAI', `Total: ${total} | Berhasil: ${success} | Gagal: ${failed}`);

        btnBulk.disabled = false;
        btnBulk.innerHTML = '<i class="bx bx-layer-plus me-1"></i> Proses Bulk';
    });
}

function addBulkLog(type, step, msg) {
    const div = document.createElement('div');
    div.className = `log-entry ${type}`;
    div.innerHTML = `<div class="log-step">${escHtml(step)}</div><div class="log-msg">${msg}</div>`;
    bulkLog.appendChild(div);
    bulkLog.scrollTop = bulkLog.scrollHeight;
}

// ─── DNS Manager ─────────────────────────────────────
const btnDnsRefresh = document.getElementById('btn-dns-refresh');
const btnDnsAdd = document.getElementById('btn-dns-add');
const dnsFilter = document.getElementById('dns-filter');
const dnsList = document.getElementById('dns-list');
const formDnsEdit = document.getElementById('form-dns-edit');
let dnsModalInstance = null;

let allDnsRecords = [];

async function loadDnsRecords() {
    dnsList.innerHTML = '<div class="text-center p-3"><div class="spinner-border text-primary" role="status"><span class="visually-hidden">Loading...</span></div></div>';
    try {
        const res = await fetch('/dns-records', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        });
        const json = await res.json();
        if (!json.success || !json.result) {
            dnsList.innerHTML = `<p class="text-danger p-3 mb-0">Error: ${json.error || 'Gagal'}</p>`;
            return;
        }
        allDnsRecords = json.result;
        renderDnsRecords(allDnsRecords);
    } catch (err) {
        dnsList.innerHTML = `<p class="text-danger p-3 mb-0">Error: ${err.message}</p>`;
    }
}

function renderDnsRecords(records) {
    const filter = (dnsFilter.value || '').toLowerCase();
    let items = records;
    if (filter) {
        items = items.filter(r => r.name.toLowerCase().includes(filter) || r.content.toLowerCase().includes(filter));
    }

    if (items.length === 0) {
        dnsList.innerHTML = '<p class="text-muted p-3 mb-0">Tidak ada DNS record.</p>';
        return;
    }

    let html = '<table class="table table-hover"><thead><tr><th>Type</th><th>Name</th><th>Content</th><th>Proxied</th><th>TTL</th><th>Aksi</th></tr></thead><tbody class="table-border-bottom-0">';
    items.forEach(r => {
        const typeBadge = r.type === 'A' ? 'bg-label-success' : r.type === 'CNAME' ? 'bg-label-info' : 'bg-label-warning';
        const proxiedBadge = r.proxied ? '<span class="badge bg-label-success">ON</span>' : '<span class="badge bg-label-secondary">OFF</span>';
        const ttlText = r.ttl === 1 ? 'Auto' : r.ttl + 's';
        const isProtected = r.name === '*.zidstorepanel.web.id' || r.name === 'fallback.zidstorepanel.web.id';
        html += `<tr>
            <td><span class="badge ${typeBadge}">${r.type}</span></td>
            <td><strong>${escHtml(r.name)}</strong></td>
            <td><code>${escHtml(r.content)}</code></td>
            <td>${proxiedBadge}</td>
            <td>${ttlText}</td>
            <td>
                <button class="btn btn-sm btn-outline-secondary me-1" onclick="editDnsRecord('${r.id}','${r.type}','${escHtml(r.name)}','${escHtml(r.content)}',${r.proxied})"><i class="bx bx-edit-alt"></i></button>
                ${isProtected ? '' : `<button class="btn btn-sm btn-outline-danger" onclick="deleteDnsRecordConfirm('${r.id}','${escHtml(r.name)}')"><i class="bx bx-trash"></i></button>`}
            </td>
        </tr>`;
    });
    html += '</tbody></table>';
    dnsList.innerHTML = html;
}

function editDnsRecord(id, type, name, content, proxied) {
    document.getElementById('dns-edit-id').value = id;
    document.getElementById('dns-edit-type').value = type;
    document.getElementById('dns-edit-name').value = name;
    document.getElementById('dns-edit-content').value = content;
    const proxiedSelect = document.getElementById('dns-edit-proxied');
    proxiedSelect.value = proxied ? 'true' : 'false';
    document.getElementById('dns-modal-title').textContent = `Edit: ${name}`;
    
    if (!dnsModalInstance) {
        dnsModalInstance = new bootstrap.Modal(document.getElementById('dns-modal'));
    }
    dnsModalInstance.show();

    // NS, MX, TXT, SRV cannot be proxied on Cloudflare
    const unproxyable = ['NS', 'MX', 'TXT', 'SRV'];
    proxiedSelect.disabled = unproxyable.includes(type);
    if (unproxyable.includes(type)) proxiedSelect.value = 'false';
}

if (formDnsEdit) {
    formDnsEdit.addEventListener('submit', async e => {
        e.preventDefault();
        const isEdit = !!document.getElementById('dns-edit-id').value;
        const data = {
            record_id: document.getElementById('dns-edit-id').value,
            type: document.getElementById('dns-edit-type').value,
            name: document.getElementById('dns-edit-name').value.trim(),
            content: document.getElementById('dns-edit-content').value.trim(),
            proxied: document.getElementById('dns-edit-proxied').value,
        };
        if (!data.name || !data.content) { alert('Name dan content wajib diisi.'); return; }

        try {
            const url = isEdit ? '/dns-update' : '/dns-create';
            const res = await fetch(url, {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: new URLSearchParams(data),
            });
            const json = await res.json();
            if (json.success) {
                if (dnsModalInstance) dnsModalInstance.hide();
                loadDnsRecords();
            } else {
                alert(`Error: ${json.errors?.[0]?.message || json.error || 'Gagal'}`);
            }
        } catch (err) { alert(`Error: ${err.message}`); }
    });
}

async function deleteDnsRecordConfirm(id, name) {
    if (!confirm(`Hapus DNS record "${name}"?`)) return;
    try {
        const res = await fetch('/delete-dns-record', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: new URLSearchParams({ record_id: id }),
        });
        const json = await res.json();
        if (json.success) {
            loadDnsRecords();
        } else {
            alert(`Error: ${json.errors?.[0]?.message || 'Gagal hapus'}`);
        }
    } catch (err) {
        alert(`Error: ${err.message}`);
    }
}

if (btnDnsRefresh) btnDnsRefresh.addEventListener('click', loadDnsRecords);
if (dnsFilter) dnsFilter.addEventListener('input', () => renderDnsRecords(allDnsRecords));
if (btnDnsAdd) {
    btnDnsAdd.addEventListener('click', () => {
        document.getElementById('dns-edit-id').value = '';
        document.getElementById('dns-edit-type').value = 'A';
        document.getElementById('dns-edit-name').value = '';
        document.getElementById('dns-edit-content').value = '';
        document.getElementById('dns-edit-proxied').value = 'false';
        document.getElementById('dns-modal-title').textContent = 'Add DNS Record';
        document.getElementById('dns-edit-proxied').disabled = false;
        
        if (!dnsModalInstance) {
            dnsModalInstance = new bootstrap.Modal(document.getElementById('dns-modal'));
        }
        dnsModalInstance.show();
    });
}

// ─── Utility ─────────────────────────────────────────
function showResult(el, type, msg) {
    if (!el) return;
    el.classList.remove('d-none', 'alert-success', 'alert-danger', 'alert-info', 'alert-warning');
    el.classList.add('alert', `alert-${type === 'error' ? 'danger' : type}`);
    el.textContent = msg;
}

let logModalInstance = null;
function openLog(content) {
    document.getElementById('log-modal-body').textContent = content;
    if (!logModalInstance) {
        logModalInstance = new bootstrap.Modal(document.getElementById('log-modal'));
    }
    logModalInstance.show();
}
