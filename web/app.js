const API_BASE = '/api';
let currentUser = null;
let token = localStorage.getItem('token');
let userId = localStorage.getItem('userId');
let userEmail = localStorage.getItem('userEmail');

// --- Auth Functions ---

async function register() {
    const email = document.getElementById('reg-email').value;
    const password = document.getElementById('reg-password').value;
    
    try {
        const res = await fetch(`${API_BASE}/auth/register`, {
            method: 'POST',
            body: JSON.stringify({ email, password })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data || 'Error');
        
        alert(data.message);
        
        // Auto-fill code if present (Demo convenience)
        if (data.code) {
            document.getElementById('verify-code').value = data.code;
            document.getElementById('demo-code-display').innerText = "Demo Code: " + data.code;
        }

        document.getElementById('verify-box').classList.remove('hidden');
    } catch (e) {
        alert(e.message);
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
        if (!res.ok) throw new Error(data || 'Error');

        alert('Verified! Please login.');
        document.getElementById('verify-box').classList.add('hidden');
    } catch (e) {
        alert(e.message);
    }
}

async function login() {
    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;

    try {
        const res = await fetch(`${API_BASE}/auth/login`, {
            method: 'POST',
            body: JSON.stringify({ email, password })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.message || 'Error');

        token = data.token;
        userId = data.user.id;
        userEmail = data.user.email;
        
        localStorage.setItem('token', token);
        localStorage.setItem('userId', userId);
        localStorage.setItem('userEmail', userEmail);
        
        showApp();
    } catch (e) {
        alert(e.message);
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
    
    loadLists();
    // Simple polling for collaboration demo
    setInterval(loadLists, 5000); 
}

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
    loadLists();
}

async function deleteList(id) {
    if (!confirm('Delete this list?')) return;
    await fetchAuth(`${API_BASE}/lists/${id}`, { method: 'DELETE' });
    loadLists();
}

async function loadLists() {
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
            <div style="margin-top:10px; padding-top:10px; border-top:1px solid #eee;">
                <div id="items-${list.id}">Loading items...</div>
                <div class="flex" style="margin-top:10px;">
                    <input type="text" id="new-item-${list.id}" placeholder="Add item...">
                    <button class="btn" onclick="addItem(${list.id})">Add</button>
                </div>
            </div>
        `;
        container.appendChild(div);
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
        div.className = 'flex';
        div.style.marginBottom = '5px';
        div.innerHTML = `
            <span class="${item.is_done ? 'completed' : ''}" onclick="toggleItem(${listId}, ${item.id}, ${!item.is_done})" style="cursor:pointer;">
                ${item.is_done ? '☑' : '☐'} ${item.content}
            </span>
            <button class="btn btn-red" style="font-size:0.8em; padding:2px 5px;" onclick="deleteItem(${listId}, ${item.id})">x</button>
        `;
        container.appendChild(div);
    });
}

async function addItem(listId) {
    const input = document.getElementById(`new-item-${listId}`);
    const content = input.value;
    if (!content) return;

    await fetchAuth(`${API_BASE}/lists/${listId}/items`, {
        method: 'POST',
        body: JSON.stringify({ content })
    });
    input.value = '';
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

