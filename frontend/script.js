const API_URL = 'http://localhost:8080/api';
let token = localStorage.getItem('token');
let expenseChart = null;

// ============ AUTH ============
if (document.getElementById('loginForm')) {
    document.getElementById('loginForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = document.getElementById('email').value;
        const password = document.getElementById('password').value;
        
        try {
            const response = await fetch(`${API_URL}/login`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email, password })
            });
            
            const data = await response.json();
            if (response.ok) {
                localStorage.setItem('token', data.token);
                localStorage.setItem('userId', data.user_id);
                localStorage.setItem('userRole', data.role);
                localStorage.setItem('username', data.username);
                localStorage.setItem('familyId', data.family_id);
                window.location.href = 'dashboard.html';
            } else {
                alert(data.error || 'Ошибка входа');
            }
        } catch (error) {
            alert('Ошибка соединения');
        }
    });
}

function showRegister() {
    document.getElementById('registerModal').style.display = 'block';
}

function hideRegister() {
    document.getElementById('registerModal').style.display = 'none';
}

if (document.getElementById('registerForm')) {
    document.getElementById('registerForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('regUsername').value;
        const email = document.getElementById('regEmail').value;
        const password = document.getElementById('regPassword').value;
        const familyCode = document.getElementById('familyCode').value;
        
        try {
            const response = await fetch(`${API_URL}/register`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ 
                    username, 
                    email, 
                    password, 
                    family_id: familyCode ? parseInt(familyCode) : 1 
                })
            });
            
            const data = await response.json();
            if (response.ok) {
                localStorage.setItem('token', data.token);
                localStorage.setItem('userId', data.user_id);
                localStorage.setItem('userRole', data.role);
                window.location.href = 'dashboard.html';
            } else {
                alert(data.error || 'Ошибка регистрации');
            }
        } catch (error) {
            alert('Ошибка соединения');
        }
    });
}

async function logout() {
    if (ws) ws.close();
    localStorage.clear();
    window.location.href = 'index.html';
}

// Check auth on protected pages
if (!window.location.pathname.includes('index.html') && !token) {
    window.location.href = 'index.html';
}

// Load user info
if (document.getElementById('username')) {
    loadUserInfo();
}

async function loadUserInfo() {
    try {
        const response = await fetch(`${API_URL}/user/info`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const user = await response.json();
        if (document.getElementById('username')) {
            document.getElementById('username').innerText = user.username;
        }
        if (document.getElementById('userRole')) {
            document.getElementById('userRole').innerText = user.role === 'admin' ? '👑 Админ' : 
                                                          user.role === 'adult' ? '👨 Взрослый' : '👶 Ребенок';
        }
        if (document.getElementById('familyInfo')) {
            document.getElementById('familyInfo').innerHTML = `Семья #${user.family_id}<br>Роль: ${user.role}`;
        }
    } catch (error) {
        console.error('Error loading user info:', error);
    }
}

// ============ FILES ============
async function uploadFile() {
    const fileInput = document.getElementById('fileInput');
    const accessType = document.getElementById('accessType').value;
    const file = fileInput.files[0];
    
    if (!file) {
        alert('Выберите файл');
        return;
    }
    
    const formData = new FormData();
    formData.append('file', file);
    formData.append('access_type', accessType);
    
    try {
        const response = await fetch(`${API_URL}/files/upload`, {
            method: 'POST',
            headers: { 'Authorization': `Bearer ${token}` },
            body: formData
        });
        
        const data = await response.json();
        if (response.ok) {
            alert('Файл загружен!');
            loadFiles();
            fileInput.value = '';
        } else {
            alert(data.error || 'Ошибка загрузки');
        }
    } catch (error) {
        alert('Ошибка соединения');
    }
}

async function loadFiles() {
    try {
        const response = await fetch(`${API_URL}/files`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const files = await response.json();
        
        const fileList = document.getElementById('fileList');
        if (fileList) {
            fileList.innerHTML = '';
            if (files.length === 0) {
                fileList.innerHTML = '<p style="text-align: center; padding: 20px; color: #999;">📁 Нет загруженных файлов</p>';
                return;
            }
            
            files.forEach(file => {
                const fileDiv = document.createElement('div');
                fileDiv.className = 'file-item';
                
                const isOwner = file.user_id == localStorage.getItem('userId');
                const isAdmin = localStorage.getItem('userRole') === 'admin';
                const canDelete = isOwner || isAdmin;
                
                // Иконка в зависимости от типа доступа
                let accessIcon = '';
                let accessText = '';
                switch(file.access_type) {
                    case 'private':
                        accessIcon = '🔒';
                        accessText = 'Личное';
                        break;
                    case 'family':
                        accessIcon = '👨‍👩‍👧‍👦';
                        accessText = 'Семейное';
                        break;
                    case 'public':
                        accessIcon = '🌍';
                        accessText = 'Публичное';
                        break;
                }
                
                const fileSize = file.size ? formatFileSize(file.size) : '';
                const fileDate = new Date(file.created_at).toLocaleDateString('ru-RU');
                
                fileDiv.innerHTML = `
                    <div style="flex: 1;">
                        <div style="display: flex; align-items: center; gap: 10px;">
                            <span style="font-size: 24px;">📄</span>
                            <div>
                                <strong style="font-size: 16px;">${escapeHtml(file.name)}</strong>
                                <div style="font-size: 12px; color: #666; margin-top: 4px;">
                                    <span>${accessIcon} ${accessText}</span>
                                    ${fileSize ? `<span style="margin-left: 10px;">📦 ${fileSize}</span>` : ''}
                                    <span style="margin-left: 10px;">📅 ${fileDate}</span>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div class="file-actions">
                        <button onclick="downloadFile(${file.id})" title="Скачать" style="background: #4CAF50;">📥</button>
                        ${canDelete ? `<button onclick="deleteFile(${file.id}, '${escapeHtml(file.name)}')" title="Удалить" style="background: #f44336;">🗑</button>` : ''}
                        ${file.access_type === 'public' && file.share_token ? `<button onclick="copyLink('${file.share_token}')" title="Поделиться" style="background: #2196F3;">🔗</button>` : ''}
                        ${canDelete ? `<button onclick="changeAccess(${file.id})" title="Изменить доступ" style="background: #FF9800;">🔧</button>` : ''}
                    </div>
                `;
                fileList.appendChild(fileDiv);
            });
        }
    } catch (error) {
        console.error('Error loading files:', error);
        const fileList = document.getElementById('fileList');
        if (fileList) {
            fileList.innerHTML = '<p style="text-align: center; padding: 20px; color: #e53e3e;">❌ Файлов нет!</p>';
        }
    }
}

// Функция для форматирования размера файла
function formatFileSize(bytes) {
    if (!bytes) return '';
    const sizes = ['Б', 'КБ', 'МБ', 'ГБ'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + sizes[i];
}

// Функция удаления файла
async function deleteFile(fileId, fileName) {
    if (!confirm(`Вы уверены, что хотите удалить файл "${fileName}"?`)) {
        return;
    }
    
    try {
        const response = await fetch(`${API_URL}/files/${fileId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });
        
        const data = await response.json();
        
        if (response.ok) {
            alert(`✅ Файл "${fileName}" успешно удален`);
            loadFiles(); // Перезагружаем список файлов
        } else {
            alert(`❌ Ошибка удаления: ${data.error || 'Неизвестная ошибка'}`);
        }
    } catch (error) {
        console.error('Error deleting file:', error);
        alert('❌ Ошибка соединения при удалении файла');
    }
}

// Обновите функцию changeAccess, добавив обработку обновления списка
async function changeAccess(fileId) {
    const newAccess = prompt('Введите тип доступа (private, family, public):');
    if (newAccess && ['private', 'family', 'public'].includes(newAccess)) {
        try {
            const response = await fetch(`${API_URL}/files/${fileId}/access`, {
                method: 'PUT',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ access_type: newAccess })
            });
            
            const data = await response.json();
            
            if (response.ok) {
                alert('✅ Доступ обновлен');
                loadFiles(); // Перезагружаем список
            } else {
                alert(`❌ Ошибка: ${data.error || 'Не удалось обновить доступ'}`);
            }
        } catch (error) {
            console.error('Error changing access:', error);
            alert('❌ Ошибка соединения');
        }
    } else if (newAccess) {
        alert('❌ Неверный тип доступа. Используйте: private, family, public');
    }
}

async function downloadFile(fileId) {
    const token = localStorage.getItem('token');
    if (!token) {
        alert('Не авторизован');
        return;
    }
    
    try {
        console.log('Downloading file:', fileId);
        
        const response = await fetch(`${API_URL}/files/${fileId}/download`, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${token}`
            }
        });
        
        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Ошибка скачивания');
        }
        
        // Получаем blob из ответа
        const blob = await response.blob();
        
        // Создаем URL для blob
        const url = window.URL.createObjectURL(blob);
        
        // Создаем временную ссылку и кликаем по ней
        const a = document.createElement('a');
        a.href = url;
        
        // Получаем имя файла из заголовка Content-Disposition
        const contentDisposition = response.headers.get('Content-Disposition');
        let filename = `file_${fileId}`;
        if (contentDisposition) {
            const match = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/);
            if (match && match[1]) {
                filename = match[1].replace(/['"]/g, '');
            }
        }
        
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        
        // Очищаем
        document.body.removeChild(a);
        window.URL.revokeObjectURL(url);
        
        console.log('File downloaded successfully');
        
    } catch (error) {
        console.error('Download error:', error);
        alert('Ошибка скачивания: ' + error.message);
    }
}


function copyLink(token) {
    const link = `${API_URL}/public/file/${token}`;
    navigator.clipboard.writeText(link);
    alert('Ссылка скопирована!');
}

// ============ FINANCE ============
async function loadTransactions() {
    try {
        const response = await fetch(`${API_URL}/finance/transactions`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const data = await response.json();
        
        // Update stats
        if (document.getElementById('totalIncome')) {
            document.getElementById('totalIncome').innerText = data.statistics.total_income.toFixed(2) + ' ₽';
            document.getElementById('totalExpense').innerText = data.statistics.total_expense.toFixed(2) + ' ₽';
            document.getElementById('balance').innerText = data.statistics.balance.toFixed(2) + ' ₽';
            
            const comparison = data.statistics.comparison_to_last_month;
            const comparisonText = comparison === 'higher' ? '📈 Больше' :
                                   comparison === 'lower' ? '📉 Меньше' : '➡️ Столько же';
            document.getElementById('comparison').innerHTML = comparisonText + '<br><small>чем в прошлом месяце</small>';
        }
        
        // Update chart
        updateChart(data.transactions);
        
        // Update transactions table
        const tbody = document.getElementById('transactionsTable');
        if (tbody) {
            tbody.innerHTML = '';
            if (data.transactions.length === 0) {
                tbody.innerHTML = '<tr><td colspan="5" style="text-align: center;">Нет транзакций</td></tr>';
                return;
            }
            data.transactions.forEach(t => {
                const row = tbody.insertRow();
                row.insertCell(0).innerText = new Date(t.date).toLocaleDateString();
                row.insertCell(1).innerText = t.type === 'income' ? '💰 Доход' : '💸 Расход';
                row.insertCell(2).innerText = t.category || '-';
                row.insertCell(3).innerText = t.description || '-';
                row.insertCell(4).innerText = t.amount.toFixed(2) + ' ₽';
                row.insertCell(4).className = t.type === 'income' ? 'stat-positive' : 'stat-negative';
            });
        }
    } catch (error) {
        console.error('Error loading transactions:', error);
    }
}

function updateChart(transactions) {
    const ctx = document.getElementById('expenseChart');
    if (!ctx) return;
    
    // Group expenses by category
    const expensesByCategory = {};
    transactions.forEach(t => {
        if (t.type === 'expense') {
            const category = t.category || 'other';
            expensesByCategory[category] = (expensesByCategory[category] || 0) + t.amount;
        }
    });
    
    const categories = {
        'food': '🍔 Еда',
        'transport': '🚗 Транспорт',
        'bills': '📄 Счета',
        'entertainment': '🎮 Развлечения',
        'other': '📦 Другое'
    };
    
    const labels = Object.keys(expensesByCategory).map(c => categories[c] || c);
    const data = Object.values(expensesByCategory);
    
    if (expenseChart) {
        expenseChart.destroy();
    }
    
    expenseChart = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: labels,
            datasets: [{
                data: data,
                backgroundColor: ['#FF6384', '#36A2EB', '#FFCE56', '#4BC0C0', '#9966FF']
            }]
        },
        options: {
            responsive: true,
            plugins: {
                legend: { position: 'bottom' }
            }
        }
    });
}

async function addTransaction() {
    const amount = parseFloat(document.getElementById('amount').value);
    const type = document.getElementById('type').value;
    const category = document.getElementById('category').value;
    const description = document.getElementById('description').value;
    
    if (isNaN(amount) || amount <= 0) {
        alert('Введите корректную сумму');
        return;
    }
    
    // Используем текущую дату в формате ISO с временем
    const now = new Date();
    const dateTime = now.toISOString(); // Формат: 2026-05-11T15:30:00.000Z
    
    const transactionData = {
        amount: amount,
        type: type,
        category: category,
        description: description || '',
        date: dateTime  // Отправляем полную дату с временем
    };
    
    console.log('Sending transaction:', transactionData);
    
    try {
        const response = await fetch(`${API_URL}/finance/transactions`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(transactionData)
        });
        
        const data = await response.json();
        
        if (response.ok) {
            alert('✅ Транзакция добавлена');
            loadTransactions(); // Перезагружаем список
            // Очищаем форму
            document.getElementById('amount').value = '';
            document.getElementById('description').value = '';
        } else {
            alert('Ошибка: ' + (data.error || 'Неизвестная ошибка'));
        }
    } catch (error) {
        console.error('Error adding transaction:', error);
        alert('Ошибка соединения с сервером');
    }
}

// ============ CHAT ============
let ws = null;
let reconnectAttempts = 0;
const maxReconnectAttempts = 10;

function connectWebSocket() {
    const token = localStorage.getItem('token');
    if (!token) {
        console.log('No token, skipping WebSocket connection');
        return;
    }
    
    // Закрываем старое соединение если есть
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.close();
    }
    
    // Используем WebSocket с токеном в URL
    const wsUrl = `ws://localhost:8080/api/chat/ws?token=${encodeURIComponent(token)}`;
    console.log('Connecting to WebSocket:', wsUrl);
    
    ws = new WebSocket(wsUrl);
    
    ws.onopen = () => {
        console.log('✅ WebSocket connected successfully');
        reconnectAttempts = 0;
        // Показываем уведомление о подключении
        showChatStatus('Соединение установлено', 'success');
    };
    
    ws.onmessage = (event) => {
        try {
            const message = JSON.parse(event.data);
            console.log('📨 Received message:', message);
            addMessageToChat(message);
            playNotificationSound();
        } catch (error) {
            console.error('Error parsing message:', error);
        }
    };
    
    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        showChatStatus('Ошибка соединения', 'error');
    };
    
    ws.onclose = (event) => {
        console.log(`WebSocket disconnected. Code: ${event.code}, Reason: ${event.reason}`);
        
        if (event.code !== 1000) { // Нормальное закрытие
            showChatStatus('Соединение потеряно, переподключение...', 'warning');
            
            // Попытка переподключения с экспоненциальной задержкой
            if (reconnectAttempts < maxReconnectAttempts) {
                reconnectAttempts++;
                const delay = Math.min(30000, 1000 * Math.pow(2, reconnectAttempts));
                console.log(`Reconnecting in ${delay/1000} seconds... (Attempt ${reconnectAttempts}/${maxReconnectAttempts})`);
                setTimeout(connectWebSocket, delay);
            } else {
                console.log('Max reconnection attempts reached');
                showChatStatus('Не удалось подключиться к чату. Перезагрузите страницу.', 'error');
            }
        }
    };
}

function showChatStatus(message, type) {
    const messagesDiv = document.getElementById('messages');
    if (!messagesDiv) return;
    
    const statusDiv = document.createElement('div');
    statusDiv.className = 'chat-status';
    statusDiv.style.textAlign = 'center';
    statusDiv.style.padding = '5px';
    statusDiv.style.margin = '5px';
    statusDiv.style.borderRadius = '5px';
    
    if (type === 'success') {
        statusDiv.style.background = '#d4edda';
        statusDiv.style.color = '#155724';
    } else if (type === 'error') {
        statusDiv.style.background = '#f8d7da';
        statusDiv.style.color = '#721c24';
    } else {
        statusDiv.style.background = '#fff3cd';
        statusDiv.style.color = '#856404';
    }
    
    statusDiv.innerHTML = message;
    messagesDiv.appendChild(statusDiv);
    
    // Удаляем через 3 секунды
    setTimeout(() => {
        if (statusDiv.parentNode) {
            statusDiv.remove();
        }
    }, 3000);
}

function playNotificationSound() {
    // Опционально: воспроизводим звук уведомления
    try {
        const audio = new Audio('data:audio/wav;base64,U3RlYWx0aCBzb3VuZA==');
        audio.volume = 0.3;
        audio.play().catch(e => console.log('Audio play failed:', e));
    } catch(e) {}
}

function addMessageToChat(message) {
    const messagesDiv = document.getElementById('messages');
    if (!messagesDiv) return;
    
    const messageDiv = document.createElement('div');
    messageDiv.className = 'message';
    const currentUserId = parseInt(localStorage.getItem('userId'));
    const isOwn = message.user_id === currentUserId;
    
    messageDiv.style.background = isOwn ? '#667eea' : '#f0f0f0';
    messageDiv.style.color = isOwn ? 'white' : '#333';
    messageDiv.style.marginLeft = isOwn ? 'auto' : '0';
    messageDiv.style.marginRight = isOwn ? '0' : 'auto';
    messageDiv.style.maxWidth = '70%';
    messageDiv.style.padding = '10px 15px';
    messageDiv.style.borderRadius = '15px';
    messageDiv.style.marginBottom = '10px';
    messageDiv.style.wordWrap = 'break-word';
    
    const time = new Date(message.created_at).toLocaleTimeString('ru-RU', {
        hour: '2-digit',
        minute: '2-digit'
    });
    
    messageDiv.innerHTML = `
        <strong style="display: block; margin-bottom: 5px;">${escapeHtml(message.username)}</strong>
        <span style="font-size: 14px;">${escapeHtml(message.message)}</span>
        <span style="font-size: 10px; opacity: 0.7; display: block; margin-top: 5px;">${time}</span>
    `;
    
    messagesDiv.appendChild(messageDiv);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

async function sendMessage() {
    const input = document.getElementById('messageInput');
    const message = input.value.trim();
    
    if (!message) {
        return;
    }
    
    if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.error('WebSocket not connected. State:', ws ? ws.readyState : 'null');
        alert('Чат не подключен. Пожалуйста, подождите или перезагрузите страницу.');
        
        // Пытаемся переподключиться
        connectWebSocket();
        return;
    }
    
    try {
        ws.send(JSON.stringify({ message }));
        input.value = '';
        console.log('✅ Message sent:', message);
    } catch (error) {
        console.error('Error sending message:', error);
        alert('Ошибка отправки сообщения');
    }
}

async function loadMessages() {
    const token = localStorage.getItem('token');
    if (!token) return;
    
    try {
        console.log('Loading messages...');
        const response = await fetch(`${API_URL}/chat/messages`, {
            headers: { 
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        const messages = await response.json();
        console.log(`Loaded ${messages.length} messages`);
        
        const messagesDiv = document.getElementById('messages');
        if (messagesDiv) {
            messagesDiv.innerHTML = '';
            if (messages.length === 0) {
                const emptyDiv = document.createElement('div');
                emptyDiv.className = 'message';
                emptyDiv.style.textAlign = 'center';
                emptyDiv.style.color = '#999';
                emptyDiv.style.padding = '20px';
                emptyDiv.innerHTML = '<p>💬 Нет сообщений. Напишите что-нибудь!</p>';
                messagesDiv.appendChild(emptyDiv);
            } else {
                messages.forEach(message => addMessageToChat(message));
            }
        }
    } catch (error) {
        console.error('Error loading messages:', error);
    }
}

// Обновление статуса подключения
function updateConnectionStatus() {
    const statusDiv = document.getElementById('connectionStatus');
    if (!statusDiv) return;
    
    if (ws && ws.readyState === WebSocket.OPEN) {
        statusDiv.innerHTML = '🟢 Подключен';
        statusDiv.style.background = '#d4edda';
        statusDiv.style.color = '#155724';
    } else if (ws && ws.readyState === WebSocket.CONNECTING) {
        statusDiv.innerHTML = '🟡 Подключение...';
        statusDiv.style.background = '#fff3cd';
        statusDiv.style.color = '#856404';
    } else {
        statusDiv.innerHTML = '🔴 Отключен';
        statusDiv.style.background = '#f8d7da';
        statusDiv.style.color = '#721c24';
    }
}

// Вызываем обновление статуса периодически
setInterval(updateConnectionStatus, 1000);

// Функция для проверки состояния WebSocket
function checkWebSocketStatus() {
    if (ws) {
        const status = {
            '0': 'CONNECTING',
            '1': 'OPEN',
            '2': 'CLOSING',
            '3': 'CLOSED'
        };
        console.log(`WebSocket status: ${status[ws.readyState]}`);
        return ws.readyState === WebSocket.OPEN;
    }
    return false;
}

// Периодическая проверка соединения
setInterval(() => {
    if (ws && ws.readyState === WebSocket.CLOSED) {
        console.log('WebSocket is closed, reconnecting...');
        connectWebSocket();
    }
}, 30000);

// ============ SMART HOME ============
async function loadDevices() {
    try {
        const response = await fetch(`${API_URL}/smart-home/devices`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        const devices = await response.json();
        
        const deviceGrid = document.getElementById('deviceGrid');
        if (deviceGrid) {
            deviceGrid.innerHTML = '';
            if (devices.length === 0) {
                deviceGrid.innerHTML = '<div class="card" style="text-align: center;">Нет добавленных устройств</div>';
                return;
            }
            devices.forEach(device => {
                const card = document.createElement('div');
                card.className = 'device-card';
                const icon = device.type === 'light' ? '💡' : device.type === 'thermostat' ? '🌡️' : '🔒';
                card.innerHTML = `
                    <h3>${icon} ${device.name}</h3>
                    <p>${device.type === 'light' ? 'Свет' : device.type === 'thermostat' ? 'Термостат' : 'Замок'}</p>
                    <div class="device-status ${device.status === 'on' ? 'device-on' : 'device-off'}">
                        ${device.status === 'on' ? '🟢 Включено' : '🔴 Выключено'}
                    </div>
                    <button onclick="toggleDevice(${device.id}, '${device.status}')" class="btn">
                        ${device.status === 'on' ? 'Выключить' : 'Включить'}
                    </button>
                    ${localStorage.getItem('userRole') === 'admin' ? 
                        `<button onclick="deleteDevice(${device.id})" style="margin-top: 10px; background: #e53e3e;">🗑 Удалить</button>` : ''}
                `;
                deviceGrid.appendChild(card);
            });
        }
    } catch (error) {
        console.error('Error loading devices:', error);
    }
}

async function toggleDevice(deviceId, currentStatus) {
    const newStatus = currentStatus === 'on' ? 'off' : 'on';
    try {
        const response = await fetch(`${API_URL}/smart-home/devices/${deviceId}/status`, {
            method: 'PUT',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ status: newStatus })
        });
        
        if (response.ok) {
            loadDevices();
        } else {
            alert('Ошибка управления устройством');
        }
    } catch (error) {
        alert('Ошибка');
    }
}

async function addDevice() {
    const name = prompt('Название устройства:');
    const type = prompt('Тип (light, thermostat, lock):');
    if (name && type) {
        try {
            const response = await fetch(`${API_URL}/smart-home/devices`, {
                method: 'POST',
                headers: {
                    'Authorization': `Bearer ${token}`,
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ name, type, status: 'off', settings: {} })
            });
            
            if (response.ok) {
                alert('Устройство добавлено');
                loadDevices();
            } else {
                alert('Ошибка добавления');
            }
        } catch (error) {
            alert('Ошибка');
        }
    }
}

async function deleteDevice(deviceId) {
    if (confirm('Удалить устройство?')) {
        try {
            const response = await fetch(`${API_URL}/smart-home/devices/${deviceId}`, {
                method: 'DELETE',
                headers: { 'Authorization': `Bearer ${token}` }
            });
            if (response.ok) {
                loadDevices();
            } else {
                alert('Ошибка удаления');
            }
        } catch (error) {
            alert('Ошибка');
        }
    }
}

// ============ CALENDAR ============
let currentDate = new Date();
let currentMonth = currentDate.getMonth();
let currentYear = currentDate.getFullYear();
let calendarEvents = []; // Инициализируем пустым массивом

async function loadCalendar() {
    console.log(`Loading calendar for ${currentYear}-${currentMonth + 1}`);
    
    // Сбросим события перед загрузкой
    calendarEvents = [];
    
    try {
        const response = await fetch(`${API_URL}/calendar/events?month=${currentMonth + 1}&year=${currentYear}`, {
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
        }
        
        const events = await response.json();
        calendarEvents = Array.isArray(events) ? events : [];
        console.log(`Loaded ${calendarEvents.length} events`);
        
        renderCalendar();
        updateUpcomingEvents(calendarEvents);
        
    } catch (error) {
        console.error('Error loading calendar:', error);
        calendarEvents = []; // Убедимся, что это массив
        renderCalendar();
        
        // Показываем сообщение об ошибке в ближайших событиях
        const upcomingDiv = document.getElementById('upcomingEvents');
        if (upcomingDiv) {
            upcomingDiv.innerHTML = `
                <div style="text-align: center; padding: 20px; color: #e53e3e;">
                    ⚠️ Ошибка загрузки событий: ${error.message}
                </div>
            `;
        }
    }
}

function renderCalendar() {
    const monthNames = ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь',
                        'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'];
    
    if (document.getElementById('currentMonth')) {
        document.getElementById('currentMonth').innerText = `${monthNames[currentMonth]} ${currentYear}`;
    }
    
    // Получаем первый день месяца (0 = воскресенье, 1 = понедельник, ...)
    const firstDay = new Date(currentYear, currentMonth, 1).getDay();
    // Корректируем для отображения с понедельника
    const startOffset = firstDay === 0 ? 6 : firstDay - 1;
    
    const daysInMonth = new Date(currentYear, currentMonth + 1, 0).getDate();
    const daysInPrevMonth = new Date(currentYear, currentMonth, 0).getDate();
    
    const calendarGrid = document.getElementById('calendarGrid');
    if (!calendarGrid) return;
    
    calendarGrid.innerHTML = '';
    
    // Заголовки дней недели
    const dayNames = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
    dayNames.forEach(day => {
        const dayHeader = document.createElement('div');
        dayHeader.className = 'calendar-day-header';
        dayHeader.innerText = day;
        calendarGrid.appendChild(dayHeader);
    });
    
    // Дни предыдущего месяца
    for (let i = 0; i < startOffset; i++) {
        const prevMonthDay = daysInPrevMonth - startOffset + i + 1;
        const dayDiv = createCalendarDay(prevMonthDay, true);
        calendarGrid.appendChild(dayDiv);
    }
    
    // Дни текущего месяца
    for (let day = 1; day <= daysInMonth; day++) {
        const dayDiv = createCalendarDay(day, false);
        calendarGrid.appendChild(dayDiv);
    }
    
    // Дни следующего месяца (чтобы заполнить сетку)
    const totalCells = Math.ceil((startOffset + daysInMonth) / 7) * 7;
    const remainingCells = totalCells - (startOffset + daysInMonth);
    for (let i = 1; i <= remainingCells; i++) {
        const dayDiv = createCalendarDay(i, true);
        calendarGrid.appendChild(dayDiv);
    }
}

function createCalendarDay(day, isOtherMonth) {
    const dayDiv = document.createElement('div');
    dayDiv.className = 'calendar-day';
    if (isOtherMonth) {
        dayDiv.style.opacity = '0.4';
        dayDiv.style.background = '#f9f9f9';
    }
    
    // Создаем контейнер для даты
    const dateDiv = document.createElement('div');
    dateDiv.style.fontWeight = 'bold';
    dateDiv.style.marginBottom = '5px';
    dateDiv.innerText = day;
    dayDiv.appendChild(dateDiv);
    
    // Находим события для этого дня - защита от null
    let currentDayEvents = [];
    if (!isOtherMonth && calendarEvents && Array.isArray(calendarEvents)) {
        currentDayEvents = calendarEvents.filter(event => {
            if (!event || !event.event_date) return false;
            const eventDate = new Date(event.event_date);
            return eventDate.getDate() === day && 
                   eventDate.getMonth() === currentMonth && 
                   eventDate.getFullYear() === currentYear;
        });
    }
    
    // Добавляем события
    const eventsContainer = document.createElement('div');
    eventsContainer.className = 'calendar-events';
    eventsContainer.style.fontSize = '11px';
    
    if (currentDayEvents.length > 0) {
        currentDayEvents.slice(0, 3).forEach(event => {
            const eventDiv = document.createElement('div');
            eventDiv.style.background = '#667eea';
            eventDiv.style.color = 'white';
            eventDiv.style.borderRadius = '3px';
            eventDiv.style.padding = '2px 4px';
            eventDiv.style.marginTop = '2px';
            eventDiv.style.cursor = 'pointer';
            eventDiv.style.fontSize = '10px';
            eventDiv.style.overflow = 'hidden';
            eventDiv.style.textOverflow = 'ellipsis';
            eventDiv.style.whiteSpace = 'nowrap';
            eventDiv.title = `${event.title || 'Без названия'}${event.event_time ? ` (${event.event_time})` : ''}`;
            
            const timeIcon = event.event_time ? '⏰ ' : '📅 ';
            const eventTitle = event.title || 'Без названия';
            eventDiv.innerText = timeIcon + (eventTitle.length > 12 ? eventTitle.slice(0, 10) + '...' : eventTitle);
            
            eventDiv.onclick = (e) => {
                e.stopPropagation();
                showEventDetails(event);
            };
            
            eventsContainer.appendChild(eventDiv);
        });
        
        if (currentDayEvents.length > 3) {
            const moreDiv = document.createElement('div');
            moreDiv.style.color = '#999';
            moreDiv.style.fontSize = '10px';
            moreDiv.style.marginTop = '2px';
            moreDiv.style.cursor = 'pointer';
            moreDiv.innerText = `+${currentDayEvents.length - 3} еще`;
            moreDiv.onclick = (e) => {
                e.stopPropagation();
                showAllEventsForDay(day, currentDayEvents);
            };
            eventsContainer.appendChild(moreDiv);
        }
    }
    
    dayDiv.appendChild(eventsContainer);
    
    // Добавляем обработчик клика для добавления события
    if (!isOtherMonth) {
        dayDiv.style.cursor = 'pointer';
        dayDiv.onclick = () => showAddEventDialog(day);
    }
    
    return dayDiv;
}

function updateUpcomingEvents(events) {
    const upcomingDiv = document.getElementById('upcomingEvents');
    if (!upcomingDiv) return;
    
    // Защита от null/undefined
    if (!events || !Array.isArray(events)) {
        upcomingDiv.innerHTML = `
            <div style="text-align: center; padding: 20px; color: #999;">
                📭 Нет предстоящих событий
            </div>
        `;
        return;
    }
    
    const now = new Date();
    now.setHours(0, 0, 0, 0); // Обнуляем время для сравнения только дат
    
    // Фильтруем предстоящие события (сегодня и позже)
    const upcoming = events.filter(event => {
        if (!event || !event.event_date) return false;
        const eventDate = new Date(event.event_date);
        eventDate.setHours(0, 0, 0, 0);
        return eventDate >= now;
    }).sort((a, b) => {
        // Сортируем по дате (сначала ближайшие)
        return new Date(a.event_date) - new Date(b.event_date);
    }).slice(0, 10); // Показываем максимум 10 событий
    
    if (upcoming.length === 0) {
        upcomingDiv.innerHTML = `
            <div style="text-align: center; padding: 20px; color: #999;">
                📭 Нет предстоящих событий
            </div>
        `;
        return;
    }
    
    upcomingDiv.innerHTML = '';
    
    upcoming.forEach(event => {
        const eventDate = new Date(event.event_date);
        const today = new Date();
        today.setHours(0, 0, 0, 0);
        const tomorrow = new Date(today);
        tomorrow.setDate(tomorrow.getDate() + 1);
        
        let dateText = '';
        const isToday = eventDate.getTime() === today.getTime();
        const isTomorrow = eventDate.getTime() === tomorrow.getTime();
        
        if (isToday) {
            dateText = '🎯 <strong style="color: #e53e3e;">СЕГОДНЯ</strong>';
        } else if (isTomorrow) {
            dateText = '📅 <strong style="color: #ed8936;">ЗАВТРА</strong>';
        } else {
            dateText = eventDate.toLocaleDateString('ru-RU', {
                day: 'numeric',
                month: 'long',
                year: 'numeric'
            });
        }
        
        const eventDiv = document.createElement('div');
        eventDiv.style.cssText = `
            padding: 12px;
            border-bottom: 1px solid #e0e0e0;
            transition: background 0.3s;
            cursor: pointer;
        `;
        
        eventDiv.onmouseover = () => eventDiv.style.background = '#f8f9fa';
        eventDiv.onmouseout = () => eventDiv.style.background = 'transparent';
        eventDiv.onclick = () => showEventDetails(event);
        
        eventDiv.innerHTML = `
            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                <div style="flex: 1;">
                    <div style="margin-bottom: 5px;">
                        <strong style="font-size: 16px; color: #333;">📌 ${escapeHtml(event.title || 'Без названия')}</strong>
                    </div>
                    <div style="font-size: 13px; color: #666; margin-bottom: 5px;">
                        ${dateText}
                        ${event.event_time ? `<span style="margin-left: 10px;">⏰ ${event.event_time}</span>` : ''}
                    </div>
                    ${event.description ? `<div style="font-size: 12px; color: #888; margin-top: 5px;">${escapeHtml(event.description)}</div>` : ''}
                </div>
                <div>
                    <span style="background: #667eea; color: white; padding: 2px 8px; border-radius: 12px; font-size: 11px;">
                        ${getDaysUntil(eventDate)}
                    </span>
                </div>
            </div>
        `;
        
        upcomingDiv.appendChild(eventDiv);
    });
}

function getDaysUntil(eventDate) {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    eventDate.setHours(0, 0, 0, 0);
    
    const diffTime = eventDate - today;
    const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));
    
    if (diffDays === 0) return 'сегодня';
    if (diffDays === 1) return 'завтра';
    if (diffDays < 7) return `через ${diffDays} дн.`;
    if (diffDays < 30) return `через ${Math.floor(diffDays / 7)} нед.`;
    return `через ${Math.floor(diffDays / 30)} мес.`;
}

function showEventDetails(event) {
    if (!event) return;
    
    const eventDate = new Date(event.event_date);
    const formattedDate = eventDate.toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'long',
        year: 'numeric'
    });
    
    const modal = document.createElement('div');
    modal.style.position = 'fixed';
    modal.style.top = '0';
    modal.style.left = '0';
    modal.style.width = '100%';
    modal.style.height = '100%';
    modal.style.background = 'rgba(0,0,0,0.5)';
    modal.style.display = 'flex';
    modal.style.alignItems = 'center';
    modal.style.justifyContent = 'center';
    modal.style.zIndex = '1000';
    
    const modalContent = document.createElement('div');
    modalContent.style.background = 'white';
    modalContent.style.borderRadius = '10px';
    modalContent.style.padding = '20px';
    modalContent.style.maxWidth = '400px';
    modalContent.style.width = '90%';
    modalContent.style.maxHeight = '80%';
    modalContent.style.overflow = 'auto';
    
    const currentUserId = parseInt(localStorage.getItem('userId'));
    const userRole = localStorage.getItem('userRole');
    const canDelete = (event.user_id === currentUserId) || (userRole === 'admin');
    
    modalContent.innerHTML = `
        <h3 style="margin-bottom: 15px; color: #333;">📅 ${escapeHtml(event.title || 'Без названия')}</h3>
        <div style="margin-bottom: 10px;">
            <strong>📆 Дата:</strong> ${formattedDate}
            ${event.event_time ? `<br><strong>⏰ Время:</strong> ${event.event_time}` : ''}
        </div>
        ${event.description ? `<div style="margin-bottom: 15px;"><strong>📝 Описание:</strong><br>${escapeHtml(event.description)}</div>` : ''}
        <div style="margin-top: 20px; display: flex; gap: 10px; justify-content: flex-end;">
            ${canDelete ? `<button onclick="deleteEvent(${event.id})" style="background: #e53e3e; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer;">🗑 Удалить</button>` : ''}
            <button onclick="this.closest('div').parentElement.remove()" style="background: #999; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer;">Закрыть</button>
        </div>
    `;
    
    modal.appendChild(modalContent);
    document.body.appendChild(modal);
    
    modal.onclick = (e) => {
        if (e.target === modal) {
            modal.remove();
        }
    };
}

function showAllEventsForDay(day, events) {
    if (!events || !Array.isArray(events)) return;
    
    const modal = document.createElement('div');
    modal.style.position = 'fixed';
    modal.style.top = '0';
    modal.style.left = '0';
    modal.style.width = '100%';
    modal.style.height = '100%';
    modal.style.background = 'rgba(0,0,0,0.5)';
    modal.style.display = 'flex';
    modal.style.alignItems = 'center';
    modal.style.justifyContent = 'center';
    modal.style.zIndex = '1000';
    
    const modalContent = document.createElement('div');
    modalContent.style.background = 'white';
    modalContent.style.borderRadius = '10px';
    modalContent.style.padding = '20px';
    modalContent.style.maxWidth = '500px';
    modalContent.style.width = '90%';
    modalContent.style.maxHeight = '80%';
    modalContent.style.overflow = 'auto';
    
    let eventsHtml = '';
    events.forEach(event => {
        const currentUserId = parseInt(localStorage.getItem('userId'));
        const userRole = localStorage.getItem('userRole');
        const canDelete = (event.user_id === currentUserId) || (userRole === 'admin');
        
        eventsHtml += `
            <div style="border-bottom: 1px solid #eee; padding: 10px 0;">
                <strong>${escapeHtml(event.title || 'Без названия')}</strong>
                ${event.event_time ? `<span style="color: #666; margin-left: 10px;">⏰ ${event.event_time}</span>` : ''}
                ${event.description ? `<div style="font-size: 12px; color: #666; margin-top: 5px;">${escapeHtml(event.description)}</div>` : ''}
                ${canDelete ? `<button onclick="deleteEvent(${event.id})" style="margin-top: 5px; background: #e53e3e; color: white; border: none; padding: 4px 12px; border-radius: 3px; cursor: pointer; font-size: 12px;">Удалить</button>` : ''}
            </div>
        `;
    });
    
    modalContent.innerHTML = `
        <h3 style="margin-bottom: 15px;">📅 События на ${day} число</h3>
        ${eventsHtml}
        <div style="margin-top: 15px; text-align: right;">
            <button onclick="this.closest('div').parentElement.remove()" style="background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer;">Закрыть</button>
        </div>
    `;
    
    modal.appendChild(modalContent);
    document.body.appendChild(modal);
    
    modal.onclick = (e) => {
        if (e.target === modal) {
            modal.remove();
        }
    };
}

async function deleteEvent(eventId) {
    if (!confirm('Удалить это событие?')) return;
    
    try {
        const response = await fetch(`${API_URL}/calendar/events/${eventId}`, {
            method: 'DELETE',
            headers: { 'Authorization': `Bearer ${token}` }
        });
        
        if (response.ok) {
            alert('✅ Событие удалено');
            // Закрываем все модальные окна
            document.querySelectorAll('div[style*="position: fixed"]').forEach(modal => modal.remove());
            await loadCalendar(); // Перезагружаем календарь
        } else {
            alert('❌ Ошибка удаления');
        }
    } catch (error) {
        console.error('Error deleting event:', error);
        alert('Ошибка соединения');
    }
}

function showAddEventDialog(day) {
    const modal = document.createElement('div');
    modal.style.position = 'fixed';
    modal.style.top = '0';
    modal.style.left = '0';
    modal.style.width = '100%';
    modal.style.height = '100%';
    modal.style.background = 'rgba(0,0,0,0.5)';
    modal.style.display = 'flex';
    modal.style.alignItems = 'center';
    modal.style.justifyContent = 'center';
    modal.style.zIndex = '1000';
    
    const modalContent = document.createElement('div');
    modalContent.style.background = 'white';
    modalContent.style.borderRadius = '10px';
    modalContent.style.padding = '20px';
    modalContent.style.maxWidth = '400px';
    modalContent.style.width = '90%';
    
    // Форматируем дату для отображения
    const eventDate = new Date(currentYear, currentMonth, day);
    const formattedDate = eventDate.toLocaleDateString('ru-RU', {
        day: 'numeric',
        month: 'long',
        year: 'numeric'
    });
    
    modalContent.innerHTML = `
        <h3 style="margin-bottom: 15px;">➕ Добавить событие</h3>
        <div style="margin-bottom: 15px; padding: 8px; background: #f0f0f0; border-radius: 5px;">
            📅 ${formattedDate}
        </div>
        <div class="form-group">
            <label>Название *</label>
            <input type="text" id="eventTitle" placeholder="Название события" style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 5px;">
        </div>
        <div class="form-group">
            <label>Описание</label>
            <textarea id="eventDesc" rows="3" placeholder="Описание (необязательно)" style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 5px;"></textarea>
        </div>
        <div class="form-group">
            <label>Время</label>
            <input type="time" id="eventTime" style="width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 5px;">
        </div>
        <div style="display: flex; gap: 10px; justify-content: flex-end; margin-top: 20px;">
            <button onclick="this.closest('div').parentElement.remove()" style="background: #999; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer;">Отмена</button>
            <button onclick="addEventFromDialog(${day})" style="background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer;">Сохранить</button>
        </div>
    `;
    
    modal.appendChild(modalContent);
    document.body.appendChild(modal);
    
    modal.onclick = (e) => {
        if (e.target === modal) {
            modal.remove();
        }
    };
}

async function addEventFromDialog(day) {
    const title = document.getElementById('eventTitle')?.value.trim();
    if (!title) {
        alert('Введите название события');
        return;
    }
    
    const description = document.getElementById('eventDesc')?.value.trim() || '';
    const eventTime = document.getElementById('eventTime')?.value || null;
    
    // Формируем дату в формате YYYY-MM-DD
    const year = currentYear;
    const month = String(currentMonth + 1).padStart(2, '0');
    const dayStr = String(day).padStart(2, '0');
    const eventDate = `${year}-${month}-${dayStr}`;
    
    console.log('Adding event:', { title, description, eventDate, eventTime });
    
    await addEvent(title, description, eventDate, eventTime);
    
    // Закрываем модальное окно
    const modal = document.querySelector('div[style*="position: fixed"]');
    if (modal) modal.remove();
}

async function addEvent(title, description, eventDate, eventTime) {
    // Отправляем дату в формате YYYY-MM-DD без времени
    const dataToSend = {
        title: title,
        description: description || '',
        event_date: eventDate, // Уже в формате YYYY-MM-DD
        event_time: eventTime || null
    };
    
    console.log('Sending event data:', dataToSend);
    
    try {
        const response = await fetch(`${API_URL}/calendar/events`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${token}`,
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(dataToSend)
        });
        
        const data = await response.json();
        
        if (response.ok) {
            alert('✅ Событие добавлено');
            await loadCalendar(); // Перезагружаем календарь
        } else {
            alert('Ошибка: ' + (data.error || 'Неизвестная ошибка'));
        }
    } catch (error) {
        console.error('Error adding event:', error);
        alert('Ошибка соединения с сервером');
    }
}

function prevMonth() {
    currentMonth--;
    if (currentMonth < 0) {
        currentMonth = 11;
        currentYear--;
    }
    loadCalendar();
}

function nextMonth() {
    currentMonth++;
    if (currentMonth > 11) {
        currentMonth = 0;
        currentYear++;
    }
    loadCalendar();
}
function renderCalendar() {
    const monthNames = ['Январь', 'Февраль', 'Март', 'Апрель', 'Май', 'Июнь',
                        'Июль', 'Август', 'Сентябрь', 'Октябрь', 'Ноябрь', 'Декабрь'];
    
    if (document.getElementById('currentMonth')) {
        document.getElementById('currentMonth').innerText = `${monthNames[currentMonth]} ${currentYear}`;
    }
    
    // Получаем первый день месяца (0 = воскресенье, 1 = понедельник, ...)
    const firstDay = new Date(currentYear, currentMonth, 1).getDay();
    // Корректируем для отображения с понедельника
    const startOffset = firstDay === 0 ? 6 : firstDay - 1;
    
    const daysInMonth = new Date(currentYear, currentMonth + 1, 0).getDate();
    const daysInPrevMonth = new Date(currentYear, currentMonth, 0).getDate();
    
    const calendarGrid = document.getElementById('calendarGrid');
    if (!calendarGrid) return;
    
    calendarGrid.innerHTML = '';
    
    // Заголовки дней недели
    const dayNames = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
    dayNames.forEach(day => {
        const dayHeader = document.createElement('div');
        dayHeader.className = 'calendar-day-header';
        dayHeader.innerText = day;
        calendarGrid.appendChild(dayHeader);
    });
    
    // Дни предыдущего месяца
    for (let i = 0; i < startOffset; i++) {
        const prevMonthDay = daysInPrevMonth - startOffset + i + 1;
        const dayDiv = createCalendarDay(prevMonthDay, true);
        calendarGrid.appendChild(dayDiv);
    }
    
    // Дни текущего месяца
    for (let day = 1; day <= daysInMonth; day++) {
        const dayDiv = createCalendarDay(day, false);
        calendarGrid.appendChild(dayDiv);
    }
    
    // Дни следующего месяца (чтобы заполнить сетку)
    const totalCells = Math.ceil((startOffset + daysInMonth) / 7) * 7;
    const remainingCells = totalCells - (startOffset + daysInMonth);
    for (let i = 1; i <= remainingCells; i++) {
        const dayDiv = createCalendarDay(i, true);
        calendarGrid.appendChild(dayDiv);
    }
}


function showAllUpcomingEvents() {
    if (!calendarEvents || !Array.isArray(calendarEvents) || calendarEvents.length === 0) {
        alert('Нет событий');
        return;
    }
    
    const now = new Date();
    now.setHours(0, 0, 0, 0);
    
    const upcoming = calendarEvents.filter(event => {
        if (!event || !event.event_date) return false;
        const eventDate = new Date(event.event_date);
        eventDate.setHours(0, 0, 0, 0);
        return eventDate >= now;
    }).sort((a, b) => new Date(a.event_date) - new Date(b.event_date));
    
    if (upcoming.length === 0) {
        alert('Нет предстоящих событий');
        return;
    }
    
    const modal = document.createElement('div');
    modal.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 100%;
        height: 100%;
        background: rgba(0,0,0,0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 1000;
    `;
    
    const modalContent = document.createElement('div');
    modalContent.style.cssText = `
        background: white;
        border-radius: 10px;
        padding: 20px;
        max-width: 600px;
        width: 90%;
        max-height: 80%;
        overflow: auto;
    `;
    
    let eventsHtml = '<h3 style="margin-bottom: 15px;">📅 Все предстоящие события</h3>';
    
    upcoming.forEach(event => {
        const eventDate = new Date(event.event_date);
        eventsHtml += `
            <div style="border-bottom: 1px solid #eee; padding: 10px; cursor: pointer;" onclick="this.closest('div').parentElement.parentElement.remove(); showEventDetails(${JSON.stringify(event).replace(/"/g, '&quot;')})">
                <div style="display: flex; justify-content: space-between;">
                    <strong>📌 ${escapeHtml(event.title || 'Без названия')}</strong>
                    <span style="color: #667eea;">${eventDate.toLocaleDateString('ru-RU')}</span>
                </div>
                ${event.event_time ? `<div style="font-size: 12px; color: #666;">⏰ ${event.event_time}</div>` : ''}
                ${event.description ? `<div style="font-size: 12px; color: #888; margin-top: 5px;">${escapeHtml(event.description)}</div>` : ''}
            </div>
        `;
    });
    
    eventsHtml += `
        <div style="margin-top: 15px; text-align: right;">
            <button onclick="this.closest('div').parentElement.remove()" style="background: #667eea; color: white; border: none; padding: 8px 16px; border-radius: 5px; cursor: pointer;">Закрыть</button>
        </div>
    `;
    
    modalContent.innerHTML = eventsHtml;
    modal.appendChild(modalContent);
    document.body.appendChild(modal);
    
    modal.onclick = (e) => {
        if (e.target === modal) modal.remove();
    };
}

// Initialize page-specific functions
if (document.getElementById('fileList')) {
    loadFiles();
}

if (document.getElementById('messages')) {
    console.log('Chat page detected, initializing chat...');
    setTimeout(() => {
        connectWebSocket();
        loadMessages();
    }, 500);
}

if (document.getElementById('totalIncome')) {
    loadTransactions();
    // Добавляем обработчик для смены типа транзакции
    const typeSelect = document.getElementById('type');
    if (typeSelect) {
        typeSelect.addEventListener('change', updateCategories);
        updateCategories();
    }
}

if (document.getElementById('deviceGrid')) {
    loadDevices();
}

if (document.getElementById('calendarGrid')) {
    console.log('Calendar page detected, initializing calendar...');
    loadCalendar();
}