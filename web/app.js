const API_BASE = '/api';
let currentUser = null;
let token = localStorage.getItem('token');
let userId = localStorage.getItem('userId');
let userEmail = localStorage.getItem('userEmail');
let captchaIds = {
    login: null,
    reg: null,
};
let isEditingListForm = false;

const DUE_DATE_OPTIONS = [
    { label: 'No Due Date', value: '' },
    { label: 'Today', value: '0' },
    { label: 'Tomorrow', value: '1' },
    { label: 'In 3 Days', value: '3' },
    { label: 'In 1 Week', value: '7' },
    { label: 'In 2 Weeks', value: '14' },
];

function renderDueOptions() {
    return DUE_DATE_OPTIONS.map(opt => `<option value="${opt.value}">${opt.label}</option>`).join('');
}

// --- UI Helpers ---
let toastTimer = null;
function showMessage(msg, type = 'success') {
    const el = document.getElementById('toast');
    if (!el) return alert(msg);
    el.innerText = msg;
    if (type === 'error') {
        el.style.background = 'rgba(239,68,68,0.12)';
        el.style.border = '1px solid rgba(239,68,68,0.5)';
        el.style.color = '#fecdd3';
    } else {
        el.style.background = 'rgba(34,197,94,0.12)';
        el.style.border = '1px solid rgba(34,197,94,0.4)';
        el.style.color = '#bbf7d0';
    }
    el.classList.remove('hidden');
    if (toastTimer) clearTimeout(toastTimer);
    toastTimer = setTimeout(() => el.classList.add('hidden'), 2500);
}

// --- Captcha Helpers ---

async function refreshCaptcha(kind) {
    try {
        const res = await fetch(`${API_BASE}/captcha/generate`);
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed to get captcha');
        captchaIds[kind] = data.captcha_id;
        const img = document.getElementById(`${kind}-captcha-img`);
        if (img) {
            img.src = `${API_BASE}/captcha/image/${data.captcha_id}?t=${Date.now()}`;
        }
        const input = document.getElementById(`${kind}-captcha-input`);
        if (input) input.value = '';
    } catch (e) {
        showMessage(e.message, 'error');
    }
}

async function verifyCaptcha(kind) {
    const captcha_id = captchaIds[kind];
    const solution = document.getElementById(`${kind}-captcha-input`)?.value?.trim();
    if (!captcha_id || !solution) {
        throw new Error('Please complete CAPTCHA');
    }
    const res = await fetch(`${API_BASE}/captcha/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ captcha_id, solution }),
    });
    const data = await res.json();
    if (!res.ok || !data.valid) {
        throw new Error('Invalid CAPTCHA');
    }
}

// --- Auth Functions ---

async function register() {
    const email = document.getElementById('reg-email').value;
    const password = document.getElementById('reg-password').value;
    
    try {
        // await verifyCaptcha('reg');

        const res = await fetch(`${API_BASE}/auth/register`, {
            method: 'POST',
            body: JSON.stringify({ email, password })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message || data.error || '注册失败');
        
        showMessage(data.message || '验证码已发送');
        
        // Auto-fill code if present (Demo convenience)
        if (data.code) {
            document.getElementById('verify-code').value = data.code;
            document.getElementById('demo-code-display').innerText = "Demo Code: " + data.code;
        }

        document.getElementById('verify-box').classList.remove('hidden');
    } catch (e) {
        showMessage(e.message, 'error');
    }
}

async function verify() {
    const email = document.getElementById('reg-email').value;
    const code = document.getElementById('verify-code').value;

    try {
        const res = await fetch(`${API_BASE}/auth/verify`, {
            method: 'POST',
            body: JSON.stringify({ email, code })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message || data.error || '验证码错误');

        showMessage('验证成功，请登录');
        document.getElementById('verify-box').classList.add('hidden');
    } catch (e) {
        showMessage(e.message, 'error');
    }
}

async function login() {
    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;

    try {
        // await verifyCaptcha('login');

        const res = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            body: JSON.stringify({ email, password })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message || data.error || '登录失败');

        token = data.token;
        userId = data.user.id;
        userEmail = data.user.email;
        
        localStorage.setItem('token', token);
        localStorage.setItem('userId', userId);
        localStorage.setItem('userEmail', userEmail);
        
        showMessage('登录成功');
        showApp();
    } catch (e) {
        showMessage(e.message, 'error');
    }
}

function logout() {
    localStorage.clear();
    location.reload();
}

function showApp() {
    if (!token) return;
    document.getElementById('auth-section').classList.add('hidden');
    document.getElementById('app-section').classList.remove('hidden');
    document.getElementById('current-user').innerText = userEmail;
    document.getElementById('current-user-id').innerText = userId;
    
    loadLists(true);
    // Simple polling for collaboration demo
    setInterval(() => loadLists(false), 5000); 
}

// Initialize captchas on load
document.addEventListener('DOMContentLoaded', () => {
    // refreshCaptcha('login');
    // refreshCaptcha('reg');
});

// --- List Functions ---

async function fetchAuth(url, options = {}) {
    const headers = {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + token
    };
    return fetch(url, { ...options, headers });
}

async function createList() {
    const title = document.getElementById('new-list-title').value;
    if (!title) return;

    await fetchAuth(`${API_BASE}/lists`, {
        method: 'POST',
        body: JSON.stringify({ title })
    });
    document.getElementById('new-list-title').value = '';
    loadLists(true);
}

async function deleteList(id) {
    if (!confirm('Delete this list?')) return;
    await fetchAuth(`${API_BASE}/lists/${id}`, { method: 'DELETE' });
    loadLists(true);
}

async function loadLists(force = false) {
    if (!force && isEditingListForm) {
        return;
    }
    // Save open state of lists to restore after refresh
    // Simplification: just reload all. In a real app we'd diff or keep state.
    // For this demo, we might lose collapsed state if we had it.
    
    const res = await fetchAuth(`${API_BASE}/lists`);
    if (res.status === 401) { logout(); return; }
    const lists = await res.json();
    
    const container = document.getElementById('lists-container');
    // We will rebuild the DOM. 
    // To prevent input loss if typing, ideally we wouldn't nuke everything, but for demo it's ok.
    // Check if we are focusing an input, if so skip update? No, that prevents seeing updates.
    // Let's just rebuild.
    
    container.innerHTML = '';
    
    if (!lists) return;

    for (const list of lists) {
        const div = document.createElement('div');
        div.className = 'card';
        div.innerHTML = `
            <div class="flex">
                <strong>${list.title} <small>(${list.role || 'OWNER'})</small></strong>
                <div>
                    ${list.role === 'OWNER' ? `<button class="btn" onclick="openShare(${list.id})">Share</button>` : ''}
                    ${list.role === 'OWNER' ? `<button class="btn btn-red" onclick="deleteList(${list.id})">Del</button>` : ''}
                </div>
            </div>
            <div class="list-form" style="margin-top:10px; padding-top:10px; border-top:1px solid #eee;">
                <div id="items-${list.id}">Loading items...</div>
                <div class="flex" style="margin-top:10px;">
                    <input type="text" id="new-item-${list.id}" placeholder="Add item...">
                    <button class="btn" onclick="addItemExtended(${list.id})">Add</button>
                </div>
                <div style="margin-top:6px; display:grid; grid-template-columns: repeat(auto-fit,minmax(150px,1fr)); gap:8px;">
                    <input type="text" id="new-desc-${list.id}" placeholder="Description (optional)">
                    <select id="new-due-${list.id}" style="padding:6px;">
                        ${renderDueOptions()}
                    </select>
                    <select id="new-status-${list.id}" style="padding:6px;">
                        <option value="not_started">Not Started</option>
                        <option value="in_progress">In Progress</option>
                        <option value="completed">Completed</option>
                    </select>
                </div>
            </div>
        `;
        container.appendChild(div);
        registerListFormHandlers(div);
        loadItems(list.id);
    }
}

// --- Item Functions ---

async function loadItems(listId) {
    const res = await fetchAuth(`${API_BASE}/lists/${listId}/items`);
    const items = await res.json();
    
    const container = document.getElementById(`items-${listId}`);
    if (!container) return;
    container.innerHTML = '';
    
    if (!items) {
        container.innerHTML = '<i>No items</i>';
        return;
    }

    items.forEach(item => {
        const div = document.createElement('div');
        div.className = 'card';
        div.style.marginBottom = '5px';
        const status = item.status || (item.is_done ? 'completed' : 'not_started');
        const due = item.due_date ? new Date(item.due_date).toISOString().slice(0, 10) : '';
        div.innerHTML = `
            <div class="flex" style="align-items:flex-start; gap:8px;">
                <span class="${item.is_done ? 'completed' : ''}" onclick="toggleItem(${listId}, ${item.id}, ${!item.is_done})" style="cursor:pointer; min-width:18px;">
                    ${item.is_done ? '☑' : '☐'}
                </span>
                <div style="flex:1;">
                    <div><strong>${item.name || item.content}</strong></div>
                    <div style="color:#666; font-size:0.9em;">${item.description || ''}</div>
                    <div style="color:#444; font-size:0.85em; margin-top:4px;">
                        Status: ${status} ${due ? ` | Due: ${due}` : ''}
                    </div>
                </div>
                <button class="btn btn-red" style="font-size:0.8em; padding:2px 5px; align-self:flex-start;" onclick="deleteItem(${listId}, ${item.id})">x</button>
            </div>
        `;
        container.appendChild(div);
    });
}

async function addItem(listId) {
    await addItemExtended(listId);
}

async function addItemExtended(listId) {
    const nameInput = document.getElementById(`new-item-${listId}`);
    const descInput = document.getElementById(`new-desc-${listId}`);
    const dueSelect = document.getElementById(`new-due-${listId}`);
    const statusSelect = document.getElementById(`new-status-${listId}`);

    const name = nameInput?.value?.trim();
    const description = descInput?.value?.trim() || '';
    const status = statusSelect?.value || 'not_started';
    let due_date = resolveDueDateValue(dueSelect?.value || '');
    if (!name) return;

    await fetchAuth(`${API_BASE}/lists/${listId}/items/extended`, {
        method: 'POST',
        body: JSON.stringify({
            name,
            content: name,
            description,
            status,
            due_date,
        }),
    });

    if (nameInput) nameInput.value = '';
    if (descInput) descInput.value = '';
    if (dueSelect) dueSelect.value = '';
    loadItems(listId);
}

async function toggleItem(listId, itemId, isDone) {
    await fetchAuth(`${API_BASE}/items/${itemId}`, {
        method: 'PUT',
        body: JSON.stringify({ list_id: listId, is_done: isDone }) // Send list_id for sharding
    });
    loadItems(listId);
}

async function deleteItem(listId, itemId) {
    await fetchAuth(`${API_BASE}/items/${itemId}?list_id=${listId}`, { method: 'DELETE' }); // Send list_id for sharding
    loadItems(listId);
}

function registerListFormHandlers(card) {
    const form = card.querySelector('.list-form');
    if (!form) return;
    const inputs = form.querySelectorAll('input, select, textarea');
    inputs.forEach((el) => {
        el.addEventListener('focus', () => {
            isEditingListForm = true;
        });
        el.addEventListener('blur', () => {
            setTimeout(() => {
                const active = document.activeElement;
                if (!active || !active.closest('.list-form')) {
                    isEditingListForm = false;
                }
            }, 0);
        });
    });
}

function resolveDueDateValue(raw) {
    if (!raw) return '';
    const offset = parseInt(raw, 10);
    if (isNaN(offset)) return '';
    const target = new Date();
    target.setHours(0, 0, 0, 0);
    target.setDate(target.getDate() + offset);
    return `${formatDate(target)}T00:00:00Z`;
}

function formatDate(date) {
    const y = date.getFullYear();
    const m = String(date.getMonth() + 1).padStart(2, '0');
    const d = String(date.getDate()).padStart(2, '0');
    return `${y}-${m}-${d}`;
}

// --- Share Functions ---

function openShare(listId) {
    document.getElementById('share-list-id').value = listId;
    document.getElementById('share-modal').classList.remove('hidden');
}

function closeShare() {
    document.getElementById('share-modal').classList.add('hidden');
}

async function submitShare() {
    const listId = document.getElementById('share-list-id').value;
    const email = document.getElementById('share-email').value;
    const role = document.getElementById('share-role').value;

    if (!email) return;

    try {
        const res = await fetchAuth(`${API_BASE}/lists/${listId}/share`, {
            method: 'POST',
            body: JSON.stringify({ email, role })
        });
        if (!res.ok) throw new Error('Failed to share (User not found?)');
        alert('Shared successfully!');
        closeShare();
    } catch (e) {
        alert(e.message);
    }
}

// Init
if (token) {
    showApp();
}

