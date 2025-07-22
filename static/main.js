let ws;
let reconnectInterval = 5000;
let reconnectTimeout;

function getWebSocketUrl() {
    const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws';
    return `${protocol}://${location.host}/ws`;
}

function connectWebSocket() {
    ws = new WebSocket(getWebSocketUrl());

    ws.onopen = () => {
        if (reconnectTimeout) {
            clearTimeout(reconnectTimeout);
            reconnectTimeout = null;
        }
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        if (data.type === 'welcome') {
            myUser = data.user;
        } else if (data.type === 'message') {
            appendMessage(data.text, data.user);
        } else if (data.type === 'typing') {
            showDraftBubble(data.user, data.text);
        }
    };

    ws.onclose = () => {
        reconnectTimeout = setTimeout(() => {
            connectWebSocket();
        }, reconnectInterval);
    };

    ws.onerror = () => {
        ws.close();
    };
}

connectWebSocket();

const messagesDiv = document.getElementById('messages');
const typingDiv = document.getElementById('typing');
const input = document.getElementById('input');
const sendBtn = document.getElementById('sendBtn');

let typing = false;
let typingTimeout;
let myUser = null;
let typingDrafts = {};

function getAvatarUrl(user) {
    return `https://api.dicebear.com/9.x/adventurer/svg?seed=${encodeURIComponent(user)}`;
}

function playNotificationSound() {
    const audio = new Audio('/notification.mp3'); // Replace with the actual path to your sound file
    audio.play().catch((error) => {
        console.error("Error playing notification sound:", error);
    });
}

function appendMessage(msg, user) {
    // Remove draft bubble if it exists for this user
    removeDraftBubble(user);
    const bubble = document.createElement('div');
    bubble.className = 'bubble';
    if (user === myUser) {
        bubble.classList.add('me');
    } else {
        bubble.classList.add('other');
    }
    const avatar = document.createElement('img');
    avatar.src = getAvatarUrl(user);
    avatar.alt = user + ' avatar';
    avatar.className = 'avatar';
    bubble.appendChild(avatar);
    const content = document.createElement('div');
    content.style.display = 'inline-block';
    const username = document.createElement('div');
    username.className = 'username';
    username.textContent = user;
    content.appendChild(username);
    const text = document.createElement('div');
    text.textContent = msg.replace("\n", "<br>");
    content.appendChild(text);
    bubble.appendChild(content);
    messagesDiv.appendChild(bubble);
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
    playNotificationSound();
}

function showDraftBubble(user, contentText) {
    if (!contentText) {
        removeDraftBubble(user);
        return;
    }
    let draft = document.getElementById('draft-' + user);
    if (!draft) {
        draft = document.createElement('div');
        draft.className = 'bubble ' + (user === myUser ? 'me draft' : 'other draft');
        draft.id = 'draft-' + user;
        const avatar = document.createElement('img');
        avatar.src = getAvatarUrl(user);
        avatar.alt = user + ' avatar';
        avatar.className = 'avatar';
        draft.appendChild(avatar);
        const content = document.createElement('div');
        content.style.display = 'inline-block';
        const username = document.createElement('div');
        username.className = 'username';
        username.textContent = user + ' (draft)';
        content.appendChild(username);
        const text = document.createElement('div');
        text.className = 'draft-text';
        content.appendChild(text);
        draft.appendChild(content);
        messagesDiv.appendChild(draft);
    } else {
        draft.className = 'bubble ' + (user === myUser ? 'me draft' : 'other draft');
    }
    draft.querySelector('.draft-text').textContent = contentText;
    messagesDiv.scrollTop = messagesDiv.scrollHeight;
    // No timeout: draft stays until erased or message is sent
}

function removeDraftBubble(user) {
    const draft = document.getElementById('draft-' + user);
    if (draft) draft.remove();
    clearTimeout(typingDrafts[user]);
    delete typingDrafts[user];
}

function sendMessage() {
    if (input.value.trim() !== '' && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'message', text: input.value }));
        input.value = '';
    }
}

input.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') {
        sendMessage();
    }
});

input.addEventListener('input', () => {
    if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: 'typing', text: input.value }));
    }
});
sendBtn.addEventListener('click', sendMessage); 