/*!
 * ChatSphere Widget v1.0.0
 * Global Embeddable Chat Platform
 *
 * Usage:
 *   <script src="https://your-domain.com/widget/chatsphere.js"
 *           data-server="https://your-domain.com"
 *           data-channel="general"></script>
 *
 * Or manual init:
 *   ChatSphere.init({ server: '...', channel: 'general' })
 */
(function (global) {
  'use strict';

  // ── Constants ────────────────────────────────────────────────────────────────

  const STORAGE_KEY = 'cs_user';
  const UNREAD_KEY  = 'cs_unread';
  const MAX_MSG     = 200;
  const EMOJIS      = ['😀','😂','😍','🥰','😎','🤔','😮','😢','😡','🎉',
                       '👍','👎','❤️','🔥','⭐','💯','🙌','🤝','👏','🚀',
                       '💬','✅','❌','⚠️','💡','🌍','🕐','📢','🔔','🎯'];

  // ── Helpers ──────────────────────────────────────────────────────────────────

  function h(tag, attrs, ...children) {
    const el = document.createElement(tag);
    if (attrs) {
      Object.entries(attrs).forEach(([k, v]) => {
        if (k === 'class') el.className = v;
        else if (k.startsWith('data-')) el.setAttribute(k, v);
        else el[k] = v;
      });
    }
    children.flat().forEach(c => {
      if (c == null) return;
      el.appendChild(typeof c === 'string' ? document.createTextNode(c) : c);
    });
    return el;
  }

  function cs(name, ...args) {
    return h(...[args[0] || 'div', {'data-cs': name, ...((args[1]) || {})}, ...(args.slice(2) || [])]);
  }

  function svg(path, vb = '0 0 24 24') {
    const s = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
    s.setAttribute('viewBox', vb);
    s.innerHTML = path;
    return s;
  }

  function flagEmoji(cc) {
    if (!cc || cc.length !== 2) return '';
    const offset = 127397;
    return String.fromCodePoint(cc.charCodeAt(0) + offset, cc.charCodeAt(1) + offset);
  }

  function timeStr(d) {
    const date = new Date(d);
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function initials(name) {
    return (name || '?').substring(0, 2).toUpperCase();
  }

  function storage(key, val) {
    try {
      if (val === undefined) return JSON.parse(localStorage.getItem(key));
      localStorage.setItem(key, JSON.stringify(val));
    } catch (_) {}
  }

  function loadCSS(url) {
    if (document.querySelector(`link[data-cs-css]`)) return;
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = url;
    link.setAttribute('data-cs-css', '1');
    document.head.appendChild(link);
  }

  // ── Widget class ─────────────────────────────────────────────────────────────

  function ChatSphereWidget(opts) {
    this.server      = (opts.server  || '').replace(/\/$/, '');
    this.channel     = opts.channel  || 'general';
    this.position    = opts.position || 'bottom-right';
    this.title       = opts.title    || 'ChatSphere';
    this.notice      = opts.notice   || '';
    this.telegramURL = opts.telegram || '';

    this.ws          = null;
    this.open        = false;
    this.expanded    = false;
    this.onlineCount = 0;
    this.unread      = 0;
    this.msgCount    = 0;
    this.emojiOpen   = false;
    this.user        = storage(STORAGE_KEY) || { name: '', token: null, anonymous: true };
    this.view        = 'chat'; // 'chat' | 'login' | 'register'

    this._build();
    this._connect();
  }

  ChatSphereWidget.prototype = {

    // ── DOM build ──────────────────────────────────────────────────────────────

    _build() {
      loadCSS(this.server + '/widget/chatsphere.css');

      const root = this.root = document.createElement('div');
      root.setAttribute('data-cs', 'root');
      root.setAttribute('data-pos', this.position);

      // Trigger button
      const btn = this.btn = cs('btn');
      btn.innerHTML = `<svg viewBox="0 0 24 24" fill="white"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/></svg>`;
      btn.addEventListener('click', () => this._toggle());

      this.badge = cs('badge');
      this.badge.style.display = 'none';
      btn.appendChild(this.badge);

      // Panel
      const panel = this.panel = cs('panel');

      // Header
      panel.appendChild(this._buildHeader());

      // Notice
      this.noticeEl = this._buildNotice();
      panel.appendChild(this.noticeEl);

      // Content wrapper
      const content = cs('content');

      // Chat area
      const chatArea = cs('chat-area');
      this.msgs = cs('msgs');

      this.inputArea = this._buildInput();
      chatArea.appendChild(this.msgs);
      chatArea.appendChild(this.inputArea);
      content.appendChild(chatArea);

      // Side panel (expanded mode)
      content.appendChild(this._buildSide());

      panel.appendChild(content);

      // Emoji picker (positioned relative to panel)
      this.emojiPicker = this._buildEmojiPicker();
      panel.appendChild(this.emojiPicker);

      root.appendChild(panel);
      root.appendChild(btn);

      document.body.appendChild(root);
    },

    _buildHeader() {
      const hdr = cs('header');

      const wave = cs('wave');
      for (let i = 0; i < 5; i++) wave.appendChild(h('span'));

      this.titleEl = cs('title', {}, this.title);

      this.onlineEl = cs('online');
      this.onlineEl.appendChild(cs('dot'));
      this.onlineSpan = document.createTextNode('0 online');
      this.onlineEl.appendChild(this.onlineSpan);

      const tgBtn = cs('hbtn', {'title': 'Telegram'});
      tgBtn.innerHTML = `<svg viewBox="0 0 24 24" fill="white"><path d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.48.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z"/></svg>`;
      if (this.telegramURL) {
        tgBtn.addEventListener('click', () => window.open(this.telegramURL, '_blank'));
      } else {
        tgBtn.style.display = 'none';
      }

      const expandBtn = this.expandBtn = cs('hbtn', {'title': 'Expand'});
      expandBtn.innerHTML = `<svg viewBox="0 0 24 24" fill="white"><path d="M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z"/></svg>`;
      expandBtn.addEventListener('click', () => this._toggleExpand());

      const minimiseBtn = cs('hbtn', {'title': 'Minimise'});
      minimiseBtn.innerHTML = `<svg viewBox="0 0 24 24" fill="white"><path d="M19 13H5v-2h14v2z"/></svg>`;
      minimiseBtn.addEventListener('click', () => this._toggle());

      hdr.appendChild(wave);
      hdr.appendChild(this.titleEl);
      hdr.appendChild(this.onlineEl);
      if (this.telegramURL) hdr.appendChild(tgBtn);
      hdr.appendChild(expandBtn);
      hdr.appendChild(minimiseBtn);

      return hdr;
    },

    _buildNotice() {
      const el = cs('notice');
      el.classList.toggle('hidden', !this.notice);

      const text = cs('notice-text', {}, this.notice);
      const close = cs('notice-close', {'title': 'Dismiss'});
      close.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>`;
      close.addEventListener('click', () => el.classList.add('hidden'));

      el.appendChild(text);
      el.appendChild(close);
      return el;
    },

    _buildSide() {
      const side = this.side = cs('side');

      const logo = cs('side-logo');
      logo.innerHTML = `<svg viewBox="0 0 24 24"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/></svg>`;

      const name = cs('side-name', {}, this.title);
      this.sideDesc = cs('side-desc', {}, '');

      this.sideOnline = cs('side-online');
      this.sideOnline.innerHTML = `<svg width="8" height="8" viewBox="0 0 8 8"><circle cx="4" cy="4" r="4" fill="#22c55e"/></svg>`;
      this.sideOnlineSpan = document.createTextNode('0 online');
      this.sideOnline.appendChild(this.sideOnlineSpan);

      const rules = cs('rules');
      rules.innerHTML = `<h4>Chat Rules</h4><ul>
        <li>Be friendly and respectful</li>
        <li>No ads, spam or referral links</li>
        <li>Max ${MAX_MSG} characters per message</li>
        <li>Recent messages are kept for review</li>
        <li>Violations can be reported and removed</li>
      </ul>`;

      const storage = cs('side-storage');
      storage.innerHTML = `<svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/></svg>
        <span>Avatar saved locally — IP changes don't affect it</span>`;

      side.appendChild(logo);
      side.appendChild(name);
      side.appendChild(this.sideDesc);
      side.appendChild(this.sideOnline);
      side.appendChild(rules);
      side.appendChild(storage);

      return side;
    },

    _buildInput() {
      const footer = cs('footer');

      // Mode bar
      const modeBar = cs('mode-bar');

      this.anonBtn = cs('mode-btn', {'class': 'cs-mode-btn'}, '匿名');
      this.anonBtn.setAttribute('data-cs', 'mode-btn');
      this.anonBtn.classList.add(this.user.anonymous ? 'active' : '');
      this.anonBtn.addEventListener('click', () => this._setMode('anonymous'));

      this.regBtn = cs('mode-btn', {'class': 'cs-mode-btn'}, '注册用户');
      this.regBtn.setAttribute('data-cs', 'mode-btn');
      this.regBtn.classList.toggle('active', !this.user.anonymous);
      this.regBtn.addEventListener('click', () => {
        this.user.anonymous ? this._showLogin() : null;
      });

      this.userPill = cs('user-pill');
      this.userDot = cs('user-dot');
      this.userNameEl = document.createTextNode(this.user.name || 'Guest');
      this.userPill.appendChild(this.userDot);
      this.userPill.appendChild(this.userNameEl);

      this.charCount = cs('char-count', {}, `0 / ${MAX_MSG}`);

      modeBar.appendChild(this.anonBtn);
      modeBar.appendChild(this.regBtn);
      modeBar.appendChild(this.userPill);
      modeBar.appendChild(this.charCount);

      // Input row
      const row = cs('input-row');

      const emojiBtn = cs('ibtn', {'title': 'Emoji'});
      emojiBtn.setAttribute('data-cs', 'ibtn');
      emojiBtn.innerHTML = `<svg viewBox="0 0 24 24"><path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm3.5-9c.83 0 1.5-.67 1.5-1.5S16.33 8 15.5 8 14 8.67 14 9.5s.67 1.5 1.5 1.5zm-7 0c.83 0 1.5-.67 1.5-1.5S9.33 8 8.5 8 7 8.67 7 9.5 7.67 11 8.5 11zm3.5 6.5c2.33 0 4.31-1.46 5.11-3.5H6.89c.8 2.04 2.78 3.5 5.11 3.5z"/></svg>`;
      emojiBtn.addEventListener('click', (e) => { e.stopPropagation(); this._toggleEmoji(); });

      const micBtn = cs('ibtn', {'title': 'Voice'});
      micBtn.setAttribute('data-cs', 'ibtn');
      micBtn.innerHTML = `<svg viewBox="0 0 24 24"><path d="M12 14c1.66 0 2.99-1.34 2.99-3L15 5c0-1.66-1.34-3-3-3S9 3.34 9 5v6c0 1.66 1.34 3 3 3zm5.3-3c0 3-2.54 5.1-5.3 5.1S6.7 14 6.7 11H5c0 3.41 2.72 6.23 6 6.72V21h2v-3.28c3.28-.48 6-3.3 6-6.72h-1.7z"/></svg>`;
      micBtn.addEventListener('click', () => this._voiceInput(micBtn));

      this.inputEl = h('textarea', {'data-cs': 'input', placeholder: '说点什么… (Enter 发送)', rows: 1});
      this.inputEl.addEventListener('input', () => this._onInput());
      this.inputEl.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); this._send(); }
      });

      const sendBtn = cs('send', {'title': 'Send'});
      sendBtn.innerHTML = `<svg viewBox="0 0 24 24"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>`;
      sendBtn.addEventListener('click', () => this._send());

      row.appendChild(emojiBtn);
      row.appendChild(micBtn);
      row.appendChild(this.inputEl);
      row.appendChild(sendBtn);

      footer.appendChild(modeBar);
      footer.appendChild(row);

      return footer;
    },

    _buildEmojiPicker() {
      const picker = document.createElement('div');
      picker.setAttribute('data-cs', 'emoji');
      EMOJIS.forEach(e => {
        const span = h('span', {title: e, onclick: () => { this._insertEmoji(e); }}, e);
        picker.appendChild(span);
      });
      document.addEventListener('click', () => {
        picker.classList.remove('open');
        this.emojiOpen = false;
      });
      return picker;
    },

    // ── WebSocket ──────────────────────────────────────────────────────────────

    _connect() {
      const proto = this.server.startsWith('https') ? 'wss' : 'ws';
      const url = `${proto}://${this.server.replace(/^https?:\/\//, '')}/ws/${this.channel}`;
      const headers = this.user.token ? { Authorization: 'Bearer ' + this.user.token } : {};

      try {
        const wsUrl = this.user.token
          ? url + '?token=' + encodeURIComponent(this.user.token)
          : url;

        this.ws = new WebSocket(wsUrl);
        this.ws.onopen    = () => this._onOpen();
        this.ws.onmessage = (e) => this._onMessage(JSON.parse(e.data));
        this.ws.onclose   = () => this._onClose();
        this.ws.onerror   = () => {};
      } catch (e) {}
    },

    _onOpen() {
      this._addSysMsg('Connected to ' + this.title);
    },

    _onMessage(evt) {
      switch (evt.type) {
        case 'history':
          if (evt.messages) {
            evt.messages.forEach(m => this._renderMsg(m, true));
            this._scrollBottom();
          }
          break;

        case 'message':
          if (evt.message) {
            this._renderMsg(evt.message);
            if (!this.open) this._incUnread();
            this._scrollBottom();
          }
          break;

        case 'online':
          if (evt.count != null) this._setOnline(evt.count);
          break;

        case 'system':
          this._addSysMsg(evt.content);
          break;

        case 'error':
          this._addSysMsg('⚠ ' + evt.error, true);
          break;

        case 'deleted':
          if (evt.id) {
            const el = this.msgs.querySelector(`[data-msgid="${evt.id}"]`);
            if (el) el.remove();
          }
          break;
      }
    },

    _onClose() {
      this._setOnline(0);
      setTimeout(() => this._connect(), 3000);
    },

    // ── Rendering ─────────────────────────────────────────────────────────────

    _renderMsg(msg, prepend) {
      const isAnon = msg.is_anonymous;
      const name = msg.display_name || 'Guest';
      const color = msg.avatar_color || '#6b7280';
      const country = msg.country || '';
      const rep = msg.reputation || 0;

      const avatarEl = cs('avatar', {style: `background:${color}`}, initials(name));

      const meta = cs('msg-meta');
      if (country) meta.appendChild(cs('flag', {}, flagEmoji(country)));
      if (rep > 0) meta.appendChild(cs('rep', {}, 'A+ ' + rep));
      const nameEl = cs('name', {class: isAnon ? 'cs-name anon' : 'cs-name'}, name);
      nameEl.setAttribute('data-cs', 'name');
      if (isAnon) nameEl.setAttribute('data-cs', 'name');
      meta.appendChild(nameEl);
      meta.appendChild(cs('time', {}, timeStr(msg.created_at)));

      const textEl = cs('text', {}, msg.content);

      const reportBtn = h('button', {'data-cs': 'report-btn', onclick: () => this._report(msg.id)}, 'Report');

      const body = cs('msg-body');
      body.appendChild(meta);
      body.appendChild(textEl);
      body.appendChild(reportBtn);

      const wrapper = cs('msg');
      wrapper.setAttribute('data-msgid', msg.id);
      wrapper.appendChild(avatarEl);
      wrapper.appendChild(body);

      if (prepend) {
        this.msgs.insertBefore(wrapper, this.msgs.firstChild);
      } else {
        this.msgs.appendChild(wrapper);
      }

      this.msgCount++;

      // Add time dividers every 20 messages
      if (!prepend && this.msgCount % 20 === 0) {
        const div = cs('divider', {}, timeStr(msg.created_at));
        this.msgs.appendChild(div);
      }
    },

    _addSysMsg(text, isError) {
      const el = cs('sys-msg', {style: isError ? 'color:#ef4444' : ''}, text);
      this.msgs.appendChild(el);
      this._scrollBottom();
    },

    _setOnline(n) {
      this.onlineCount = n;
      const label = n + (n === 1 ? ' online' : ' online');
      this.onlineSpan.textContent = label;
      if (this.sideOnlineSpan) this.sideOnlineSpan.textContent = label;
    },

    _scrollBottom() {
      requestAnimationFrame(() => {
        this.msgs.scrollTop = this.msgs.scrollHeight;
      });
    },

    // ── Actions ───────────────────────────────────────────────────────────────

    _toggle() {
      this.open = !this.open;
      this.panel.classList.toggle('open', this.open);
      this.btn.classList.toggle('open', this.open);
      if (this.open) {
        this._clearUnread();
        this._scrollBottom();
        setTimeout(() => this.inputEl.focus(), 300);
      }
    },

    _toggleExpand() {
      this.expanded = !this.expanded;
      this.panel.classList.toggle('expanded', this.expanded);
      this.expandBtn.title = this.expanded ? 'Collapse' : 'Expand';
    },

    _toggleEmoji() {
      this.emojiOpen = !this.emojiOpen;
      this.emojiPicker.classList.toggle('open', this.emojiOpen);
    },

    _insertEmoji(e) {
      const start = this.inputEl.selectionStart;
      const val = this.inputEl.value;
      this.inputEl.value = val.slice(0, start) + e + val.slice(start);
      this.inputEl.focus();
      this._onInput();
    },

    _send() {
      const content = this.inputEl.value.trim();
      if (!content || !this.ws || this.ws.readyState !== 1) return;

      if (content.length > MAX_MSG) {
        this._addSysMsg(`⚠ Max ${MAX_MSG} characters`, true);
        return;
      }

      this.ws.send(JSON.stringify({ type: 'send', content }));
      this.inputEl.value = '';
      this._onInput();
      this.inputEl.focus();
    },

    _report(msgId) {
      if (!msgId) return;
      if (!confirm('Report this message?')) return;
      if (this.ws && this.ws.readyState === 1) {
        this.ws.send(JSON.stringify({ type: 'report', message_id: msgId, reason: 'inappropriate' }));
      }
    },

    _onInput() {
      const len = this.inputEl.value.length;
      this.charCount.textContent = `${len} / ${MAX_MSG}`;
      this.charCount.style.color = len > MAX_MSG * 0.9 ? '#ef4444' : '';
      // Auto-resize textarea
      this.inputEl.style.height = 'auto';
      this.inputEl.style.height = Math.min(this.inputEl.scrollHeight, 80) + 'px';
    },

    _voiceInput(btn) {
      if (!('webkitSpeechRecognition' in window || 'SpeechRecognition' in window)) {
        this._addSysMsg('Speech recognition not supported in this browser', true);
        return;
      }
      const SR = window.SpeechRecognition || window.webkitSpeechRecognition;
      const rec = new SR();
      rec.lang = 'zh-CN';
      rec.interimResults = false;
      btn.style.color = '#ef4444';
      rec.onresult = (e) => {
        const text = e.results[0][0].transcript;
        this.inputEl.value += text;
        this._onInput();
      };
      rec.onend = () => { btn.style.color = ''; };
      rec.start();
    },

    _setMode(mode) {
      if (mode === 'anonymous') {
        this.user.anonymous = true;
        this.user.token = null;
        storage(STORAGE_KEY, this.user);
        this._updateModeUI();
        this.ws && this.ws.close();
        this._connect();
      }
    },

    _showLogin() {
      // Inject a lightweight login/register form into the chat area
      const overlay = h('div', {style: 'position:absolute;inset:0;background:var(--cs-bg);z-index:10;padding:16px;overflow-y:auto'});

      const title = h('h3', {style: 'margin-bottom:8px;font-size:15px'}, 'Sign in to ChatSphere');
      const desc = h('p', {style: 'font-size:12px;color:var(--cs-muted);margin-bottom:16px'},
        'Register or log in to chat with a persistent identity.');

      const unField = h('input', {type:'text', placeholder:'Username', style:'width:100%;margin-bottom:8px;padding:8px 12px;border:1.5px solid var(--cs-border);border-radius:8px;font-size:13px;outline:none;box-sizing:border-box'});
      const pwField = h('input', {type:'password', placeholder:'Password', style:'width:100%;margin-bottom:8px;padding:8px 12px;border:1.5px solid var(--cs-border);border-radius:8px;font-size:13px;outline:none;box-sizing:border-box'});

      const loginBtn = h('button', {style:'width:100%;padding:9px;background:var(--cs-primary);color:white;border:none;border-radius:8px;font-size:13px;font-weight:600;cursor:pointer;margin-bottom:6px'}, 'Sign In');
      const regBtn  = h('button', {style:'width:100%;padding:9px;background:var(--cs-bg2);color:var(--cs-muted);border:none;border-radius:8px;font-size:13px;cursor:pointer;margin-bottom:10px'}, 'Create Account');
      const cancelBtn = h('button', {style:'background:none;border:none;color:var(--cs-muted);font-size:12px;cursor:pointer;width:100%;padding:4px'}, 'Continue as anonymous');

      const msgEl = h('p', {style:'font-size:12px;color:#ef4444;text-align:center;min-height:18px'});

      loginBtn.onclick = async () => {
        const res = await this._apiLogin(unField.value, pwField.value);
        if (res.error) { msgEl.textContent = res.error; return; }
        this._applyAuth(res.token, res.user);
        this.panel.removeChild(overlay);
      };

      regBtn.onclick = async () => {
        const res = await this._apiRegister(unField.value, pwField.value);
        if (res.error) { msgEl.textContent = res.error; return; }
        this._applyAuth(res.token, res.user);
        this.panel.removeChild(overlay);
      };

      cancelBtn.onclick = () => this.panel.removeChild(overlay);

      overlay.appendChild(title);
      overlay.appendChild(desc);
      overlay.appendChild(unField);
      overlay.appendChild(pwField);
      overlay.appendChild(loginBtn);
      overlay.appendChild(regBtn);
      overlay.appendChild(msgEl);
      overlay.appendChild(cancelBtn);

      this.panel.style.position = 'relative';
      this.panel.appendChild(overlay);
    },

    // ── API ───────────────────────────────────────────────────────────────────

    async _apiLogin(username, password) {
      try {
        const r = await fetch(this.server + '/api/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password })
        });
        return r.json();
      } catch (e) { return { error: 'Network error' }; }
    },

    async _apiRegister(username, password) {
      try {
        const r = await fetch(this.server + '/api/auth/register', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password })
        });
        return r.json();
      } catch (e) { return { error: 'Network error' }; }
    },

    _applyAuth(token, user) {
      this.user = { name: user.username, token, anonymous: false };
      storage(STORAGE_KEY, this.user);
      this._updateModeUI();
      this.ws && this.ws.close();
      this._connect();
    },

    _updateModeUI() {
      this.anonBtn.classList.toggle('active', this.user.anonymous);
      this.regBtn.classList.toggle('active', !this.user.anonymous);
      this.userNameEl.textContent = this.user.name || 'Guest';
    },

    // ── Unread ────────────────────────────────────────────────────────────────

    _incUnread() {
      this.unread++;
      this.badge.style.display = 'flex';
      this.badge.textContent = this.unread > 9 ? '9+' : this.unread;
    },

    _clearUnread() {
      this.unread = 0;
      this.badge.style.display = 'none';
    },

    // ── Public API ────────────────────────────────────────────────────────────

    setTitle(t) { this.titleEl.textContent = this.title = t; },
    setNotice(t) {
      this.notice = t;
      const el = this.noticeEl.querySelector('[data-cs="notice-text"]');
      if (el) el.textContent = t;
      this.noticeEl.classList.toggle('hidden', !t);
    },
    setDescription(d) { if (this.sideDesc) this.sideDesc.textContent = d; },
  };

  // ── Auto-init from script tag ────────────────────────────────────────────────

  function autoInit() {
    const script = document.currentScript ||
      document.querySelector('script[data-server],[data-channel]');
    if (!script) return;

    const opts = {
      server:   script.getAttribute('data-server')   || window.location.origin,
      channel:  script.getAttribute('data-channel')  || 'general',
      position: script.getAttribute('data-position') || 'bottom-right',
      title:    script.getAttribute('data-title')    || 'ChatSphere',
      notice:   script.getAttribute('data-notice')   || '',
      telegram: script.getAttribute('data-telegram') || '',
    };

    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', () => { global.ChatSphere = new ChatSphereWidget(opts); });
    } else {
      global.ChatSphere = new ChatSphereWidget(opts);
    }
  }

  // ── Export ───────────────────────────────────────────────────────────────────

  global.ChatSphereWidget = ChatSphereWidget;
  global.ChatSphere = {
    init(opts) {
      const instance = new ChatSphereWidget(opts);
      global.ChatSphere = instance;
      return instance;
    }
  };

  autoInit();

}(window));
