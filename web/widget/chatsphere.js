/*!
 * ChatSphere Widget v1.0.0
 * Zero-dependency embeddable chat — single file, all styles inlined.
 *
 * Usage (script tag):
 *   <script src="/widget/chatsphere.js"
 *           data-server="https://your-domain.com"
 *           data-channel="general"></script>
 *
 * Usage (manual init):
 *   ChatSphere.init({ server: '...', channel: 'general' })
 */
(function (global) {
  'use strict';

  // ── CSS injection — singleton shared by all instances on the page ────────────

  function injectCSS() {
    if (document.getElementById('cs-styles')) return;
    var s = document.createElement('style');
    s.id = 'cs-styles';
    s.textContent = [
      /* Tokens — dark first */
      '[data-cs="root"]{',
      '  --bg:#0e1017;--surface:#13151e;--surface-2:#1a1d28;',
      '  --accent:#f5a623;--accent-dim:rgba(245,166,35,.15);--accent-glow:rgba(245,166,35,.35);',
      '  --fg:#e8e9ef;--muted:#9ca3af;--border:rgba(255,255,255,.08);',
      '  --ok:#22c55e;--err:#ef4444;',
      '  --radius:16px;--w:360px;--h:520px;--btn:56px;',
      '  --shadow:0 8px 40px rgba(0,0,0,.45),0 2px 8px rgba(0,0,0,.3);',
      '  font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;',
      '  font-size:14px;line-height:1.5;color:var(--fg);',
      '  position:fixed;z-index:2147483647;',
      '}',
      /* Light mode */
      '@media(prefers-color-scheme:light){',
      '  [data-cs="root"]{',
      '    --bg:#fff;--surface:#f9fafb;--surface-2:#f3f4f6;',
      '    --fg:#111827;--muted:#6b7280;--border:#e5e7eb;',
      '    --shadow:0 8px 40px rgba(0,0,0,.14),0 2px 8px rgba(0,0,0,.08);',
      '  }',
      '}',
      /* Reduced motion */
      '@media(prefers-reduced-motion:reduce){',
      '  [data-cs="root"] *{',
      '    animation-duration:.01ms!important;',
      '    animation-iteration-count:1!important;',
      '    transition-duration:.01ms!important;',
      '  }',
      '}',
      /* Focus ring for all interactive elements */
      '[data-cs="root"] *:focus-visible{outline:2px solid var(--accent);outline-offset:2px}',
      /* Position variants */
      '[data-cs="root"][data-pos="bottom-right"]{bottom:var(--offset,20px);right:var(--offset,20px)}',
      '[data-cs="root"][data-pos="bottom-left"]{bottom:var(--offset,20px);left:var(--offset,20px)}',
      '[data-cs="root"][data-pos="top-right"]{top:var(--offset,20px);right:var(--offset,20px)}',
      '[data-cs="root"][data-pos="top-left"]{top:var(--offset,20px);left:var(--offset,20px)}',

      /* ── Button (card shape when closed, circle when open) ── */
      '@keyframes cs-pop{from{transform:scale(0);opacity:0}to{transform:scale(1);opacity:1}}',
      '[data-cs="btn"]{',
      '  width:min(260px,calc(100vw - 48px));height:56px;border-radius:28px;',
      '  background:var(--accent);border:none;cursor:pointer;',
      '  display:flex;align-items:center;gap:10px;padding:0 12px 0 10px;',
      '  box-shadow:0 4px 20px var(--accent-glow);',
      '  transition:width .3s cubic-bezier(.4,0,.2,1),height .3s,border-radius .3s,padding .3s;',
      '  position:relative;overflow:hidden;',
      '  animation:cs-pop .4s cubic-bezier(.34,1.56,.64,1) .5s both;',
      '}',
      '[data-cs="btn"]:hover{box-shadow:0 6px 24px var(--accent-glow)}',
      /* icon circle inside the card */
      '[data-cs="btn-icon"]{',
      '  width:36px;height:36px;min-width:36px;border-radius:50%;',
      '  background:rgba(0,0,0,.18);display:flex;align-items:center;justify-content:center;',
      '  transition:opacity .2s;flex-shrink:0;',
      '}',
      '[data-cs="btn-icon"] svg{width:20px;height:20px;fill:#fff}',
      /* text column */
      '[data-cs="btn-info"]{flex:1;min-width:0;display:flex;flex-direction:column;gap:1px;overflow:hidden;transition:opacity .2s,max-width .3s}',
      '[data-cs="btn-title"]{font-size:13px;font-weight:700;color:#fff;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}',
      '[data-cs="btn-preview"]{font-size:11px;color:rgba(255,255,255,.75);white-space:nowrap;overflow:hidden;text-overflow:ellipsis}',
      /* close icon (shown when open) */
      '[data-cs="btn-close-icon"]{width:20px;height:20px;fill:#fff;opacity:0;position:absolute;transition:opacity .2s;pointer-events:none}',
      /* open state: collapse to circle */
      '[data-cs="btn"].cs-open{width:var(--btn);height:var(--btn);padding:0;border-radius:50%;justify-content:center}',
      '[data-cs="btn"].cs-open [data-cs="btn-icon"]{opacity:0;width:0;min-width:0;overflow:hidden}',
      '[data-cs="btn"].cs-open [data-cs="btn-info"]{opacity:0;max-width:0}',
      '[data-cs="btn"].cs-open [data-cs="btn-close-icon"]{opacity:1}',
      /* unread badge (top-right of card) */
      '[data-cs="badge"]{',
      '  position:absolute;top:-3px;right:-3px;min-width:18px;height:18px;',
      '  background:var(--err);color:#fff;border-radius:9px;font-size:11px;font-weight:700;',
      '  display:none;align-items:center;justify-content:center;padding:0 4px;',
      '  border:2px solid var(--bg);',
      '}',
      /* DM @ badge (bottom-right of card, hidden when open) */
      '[data-cs="dm-badge"]{',
      '  position:absolute;bottom:-2px;right:10px;min-width:18px;height:18px;',
      '  background:var(--accent2,#6366f1);color:#fff;border-radius:9px;font-size:10px;font-weight:700;',
      '  display:none;align-items:center;justify-content:center;padding:0 4px;',
      '  border:2px solid var(--bg);',
      '}',
      '[data-cs="dm-badge"].cs-vis{display:flex}',
      '[data-cs="btn"].cs-open [data-cs="dm-badge"]{display:none!important}',

      /* ── Panel ── */
      '[data-cs="panel"]{',
      '  position:absolute;bottom:calc(var(--btn) + 14px);',
      '  width:var(--w);height:var(--h);max-height:calc(100vh - 100px);',
      '  background:var(--surface);border-radius:var(--radius);box-shadow:var(--shadow);',
      '  display:flex;flex-direction:column;overflow:hidden;',
      '  transform-origin:bottom right;',
      '  transform:translateY(20px);opacity:0;pointer-events:none;',
      '  transition:transform 320ms cubic-bezier(.4,0,.2,1),opacity 320ms cubic-bezier(.4,0,.2,1);',
      '}',
      '[data-cs="root"][data-pos="bottom-left"] [data-cs="panel"]{transform-origin:bottom left}',
      '[data-cs="root"][data-pos="top-right"] [data-cs="panel"]{bottom:auto;top:calc(var(--btn)+14px);transform-origin:top right}',
      '[data-cs="root"][data-pos="top-left"] [data-cs="panel"]{bottom:auto;top:calc(var(--btn)+14px);transform-origin:top left}',
      '[data-cs="root"][data-pos="top-right"] [data-cs="panel"]:not(.cs-open){transform:translateY(-20px)}',
      '[data-cs="root"][data-pos="top-left"] [data-cs="panel"]:not(.cs-open){transform:translateY(-20px)}',
      '[data-cs="panel"].cs-open{transform:translateY(0);opacity:1;pointer-events:all}',

      /* ── Status bar (yellow pulse — shown when disconnected/reconnecting) ── */
      '@keyframes cs-pulse{0%,100%{opacity:.6}50%{opacity:1}}',
      '[data-cs="status"]{',
      '  height:3px;background:var(--accent);flex-shrink:0;',
      '  display:none;animation:cs-pulse 1.5s ease-in-out infinite;',
      '}',
      '[data-cs="status"].cs-vis{display:block}',

      /* ── Header ── */
      '[data-cs="header"]{',
      '  background:linear-gradient(135deg,#1a1d28 0%,#0e1017 100%);',
      '  border-bottom:1px solid var(--border);',
      '  padding:11px 14px;display:flex;align-items:center;gap:8px;flex-shrink:0;',
      '}',
      '@media(prefers-color-scheme:light){',
      '  [data-cs="header"]{background:linear-gradient(135deg,#c47f0a 0%,#e09612 100%)}',
      '}',
      /* Wave animation (green bars — shown when connected) */
      '@keyframes cs-wave{0%,100%{transform:scaleY(1)}50%{transform:scaleY(1.7)}}',
      '[data-cs="wave"]{display:flex;align-items:center;gap:2px;flex-shrink:0}',
      '[data-cs="wave"] span{',
      '  width:3px;border-radius:2px;background:var(--ok);',
      '  animation:cs-wave 1.1s ease-in-out infinite;',
      '}',
      '[data-cs="wave"] span:nth-child(1){height:8px;animation-delay:0s}',
      '[data-cs="wave"] span:nth-child(2){height:14px;animation-delay:.15s}',
      '[data-cs="wave"] span:nth-child(3){height:10px;animation-delay:.3s}',
      '[data-cs="wave"] span:nth-child(4){height:16px;animation-delay:.45s}',
      '[data-cs="wave"] span:nth-child(5){height:9px;animation-delay:.6s}',
      '[data-cs="wave"].cs-off span{animation-play-state:paused;opacity:.35}',
      '[data-cs="title"]{',
      '  color:var(--fg);font-size:15px;font-weight:700;flex:1;',
      '  overflow:hidden;text-overflow:ellipsis;white-space:nowrap;',
      '}',
      '@media(prefers-color-scheme:light){[data-cs="title"]{color:#fff}}',
      '[data-cs="online"]{',
      '  font-size:11px;color:var(--muted);',
      '  display:flex;align-items:center;gap:4px;flex-shrink:0;',
      '}',
      '@media(prefers-color-scheme:light){[data-cs="online"]{color:rgba(255,255,255,.85)}}',
      '[data-cs="dot"]{width:7px;height:7px;border-radius:50%;background:var(--ok);flex-shrink:0}',
      '[data-cs="hbtn"]{',
      '  width:32px;height:32px;min-width:32px;border-radius:8px;border:none;cursor:pointer;',
      '  background:transparent;display:flex;align-items:center;justify-content:center;',
      '  color:var(--muted);transition:background .15s,color .15s;flex-shrink:0;',
      '}',
      '@media(prefers-color-scheme:light){[data-cs="hbtn"]{color:rgba(255,255,255,.8)}}',
      '[data-cs="hbtn"]:hover{background:var(--surface-2);color:var(--fg)}',
      '@media(prefers-color-scheme:light){[data-cs="hbtn"]:hover{background:rgba(0,0,0,.15);color:#fff}}',
      '[data-cs="hbtn"] svg{width:16px;height:16px;fill:currentColor}',
      '[data-cs="hbtn"][data-lang-btn]{width:auto;min-width:0;padding:0 7px;font-size:11px;font-weight:700;letter-spacing:.02em}',

      /* ── Notice banner ── */
      '[data-cs="notice"]{',
      '  padding:8px 14px;font-size:12px;color:#78350f;',
      '  background:#fffbeb;border-bottom:1px solid #fde68a;',
      '  display:flex;align-items:flex-start;gap:8px;flex-shrink:0;',
      '}',
      '[data-cs="notice"].cs-hidden{display:none}',
      '[data-cs="notice-text"]{flex:1;line-height:1.5}',
      '[data-cs="notice-close"]{',
      '  cursor:pointer;opacity:.6;flex-shrink:0;',
      '  background:none;border:none;padding:2px;line-height:1;font-size:14px;color:inherit;',
      '}',
      '[data-cs="notice-close"]:hover{opacity:1}',

      /* ── Messages area ── */
      '[data-cs="msgs-wrap"]{flex:1;position:relative;overflow:hidden;display:flex;flex-direction:column}',
      '[data-cs="msgs"]{',
      '  flex:1;overflow-y:auto;padding:12px 14px;',
      '}',
      '[data-cs="msgs"]::-webkit-scrollbar{width:4px}',
      '[data-cs="msgs"]::-webkit-scrollbar-thumb{background:var(--border);border-radius:2px}',
      /* "New messages" float button */
      '[data-cs="new-msgs"]{',
      '  position:absolute;bottom:10px;left:50%;transform:translateX(-50%);',
      '  background:var(--accent);color:#000;border:none;cursor:pointer;',
      '  padding:5px 14px;border-radius:20px;font-size:12px;font-weight:600;',
      '  box-shadow:0 2px 12px var(--accent-glow);display:none;white-space:nowrap;',
      '}',
      '[data-cs="new-msgs"].cs-vis{display:block}',

      /* ── Message bubbles (iMessage style) ── */
      '@keyframes cs-fadein{from{opacity:0;transform:translateY(4px)}to{opacity:1;transform:translateY(0)}}',
      '[data-cs="msg"]{',
      '  display:flex;align-items:flex-end;gap:8px;margin-bottom:10px;',
      '  animation:cs-fadein .2s ease;',
      '}',
      '[data-cs="msg"][data-mine]{flex-direction:row-reverse}',
      '[data-cs="avatar"]{',
      '  width:34px;height:34px;min-width:34px;border-radius:50%;flex-shrink:0;',
      '  display:flex;align-items:center;justify-content:center;',
      '  font-size:12px;font-weight:700;color:#fff;',
      '}',
      '[data-cs="msg-wrap"]{display:flex;flex-direction:column;max-width:75%;min-width:0}',
      '[data-cs="msg"][data-mine] [data-cs="msg-wrap"]{align-items:flex-end}',
      '[data-cs="msg-meta"]{',
      '  display:flex;align-items:center;gap:4px;margin-bottom:3px;flex-wrap:wrap;',
      '}',
      '[data-cs="msg"][data-mine] [data-cs="msg-meta"]{flex-direction:row-reverse}',
      '[data-cs="msg"][data-mine] [data-cs="name"]{display:none}',
      '[data-cs="name"]{font-size:12px;font-weight:600;color:var(--fg)}',
      '[data-cs="name"].cs-anon{color:var(--muted);font-weight:500}',
      '[data-cs="time"]{font-size:11px;color:var(--muted)}',
      '[data-cs="flag"]{font-size:12px;line-height:1}',
      '[data-cs="bubble"]{',
      '  padding:8px 12px;font-size:13px;line-height:1.55;word-break:break-word;',
      '  background:var(--surface-2);color:var(--fg);',
      '  border-radius:16px 16px 16px 4px;',
      '}',
      '[data-cs="msg"][data-mine] [data-cs="bubble"]{',
      '  background:var(--accent);color:#fff;',
      '  border-radius:16px 16px 4px 16px;',
      '}',
      '[data-cs="bubble-row"]{display:flex;align-items:center;gap:6px}',
      '[data-cs="msg-actions"]{display:flex;gap:6px;opacity:0;transition:opacity .15s;flex-shrink:0}',
      '[data-cs="msg"]:hover [data-cs="msg-actions"]{opacity:1}',
      '[data-cs="report-btn"],[data-cs="dm-msg-btn"]{font-size:10px;color:var(--muted);cursor:pointer;background:none;border:none;padding:0}',
      '[data-cs="report-btn"]:hover{color:var(--err)}',
      '[data-cs="sys-msg"]{',
      '  text-align:center;font-size:12px;color:var(--muted);margin:6px 20px;',
      '}',
      '[data-cs="sys-msg"].cs-err{color:var(--err)}',
      '[data-cs="divider"]{',
      '  display:flex;align-items:center;gap:8px;',
      '  font-size:11px;color:var(--muted);margin:10px 0;',
      '}',
      '[data-cs="divider"]::before,[data-cs="divider"]::after{content:"";flex:1;height:1px;background:var(--border)}',

      /* ── Footer / input ── */
      '[data-cs="footer"]{',
      '  border-top:1px solid var(--border);flex-shrink:0;',
      '  background:var(--surface);',
      '}',
      /* Emoji picker (in-flow, above input row) */
      '[data-cs="emoji"]{',
      '  display:none;flex-direction:column;',
      '  border-bottom:1px solid var(--border);',
      '  max-height:180px;',
      '}',
      '[data-cs="emoji"].cs-open{display:flex}',
      '[data-cs="emoji-tabs"]{',
      '  display:flex;overflow-x:auto;padding:4px 8px 0;gap:2px;flex-shrink:0;',
      '  border-bottom:1px solid var(--border);scrollbar-width:none;',
      '}',
      '[data-cs="emoji-tabs"]::-webkit-scrollbar{display:none}',
      '[data-cs="etab"]{',
      '  flex-shrink:0;border:none;background:none;cursor:pointer;',
      '  font-size:16px;padding:4px 6px;border-radius:6px 6px 0 0;',
      '  opacity:.55;transition:opacity .15s,background .15s;',
      '}',
      '[data-cs="etab"]:hover{opacity:.85;background:var(--surface-2)}',
      '[data-cs="etab"].cs-active{opacity:1;background:var(--surface-2)}',
      '[data-cs="emoji-grid"]{',
      '  display:flex;flex-wrap:wrap;gap:3px;',
      '  padding:6px 8px;overflow-y:auto;flex:1;',
      '  scrollbar-width:thin;',
      '}',
      '[data-cs="emoji-grid"] span{',
      '  font-size:20px;cursor:pointer;padding:3px 4px;',
      '  border-radius:4px;transition:background .1s;',
      '}',
      '[data-cs="emoji-grid"] span:hover{background:var(--surface-2)}',
      '[data-cs="input-row"]{display:flex;align-items:flex-end;gap:6px;padding:8px 10px 10px}',
      '[data-cs="ibtn"]{',
      '  width:44px;height:44px;min-width:44px;border-radius:50%;border:none;cursor:pointer;',
      '  background:transparent;display:flex;align-items:center;justify-content:center;',
      '  color:var(--muted);transition:color .15s,background .15s;flex-shrink:0;',
      '}',
      '[data-cs="ibtn"]:hover{background:var(--surface-2);color:var(--accent)}',
      '[data-cs="ibtn"] svg{width:20px;height:20px;fill:currentColor}',
      '[data-cs="input"]{',
      '  flex:1;border:1.5px solid var(--border);border-radius:22px;',
      '  padding:10px 14px;font-size:13px;font-family:inherit;',
      '  resize:none;outline:none;overflow-y:auto;line-height:1.4;',
      '  min-height:44px;max-height:88px;',
      '  transition:border-color .2s;',
      '  background:var(--surface-2);color:var(--fg);box-sizing:border-box;',
      '}',
      '[data-cs="input"]:focus{border-color:var(--accent)}',
      '[data-cs="input"].cs-over{border-color:var(--err)!important}',
      '[data-cs="input"]::placeholder{color:var(--muted)}',
      '[data-cs="send"]{',
      '  width:44px;height:44px;min-width:44px;border-radius:50%;border:none;cursor:pointer;',
      '  background:var(--accent);display:flex;align-items:center;justify-content:center;',
      '  transition:filter .15s,transform .1s;flex-shrink:0;',
      '}',
      '[data-cs="send"]:hover{filter:brightness(1.1)}',
      '[data-cs="send"]:active{transform:scale(.92)}',
      '[data-cs="send"] svg{width:18px;height:18px;fill:#000;transform:translateX(1px)}',
      '[data-cs="char-count"]{',
      '  font-size:11px;color:var(--muted);',
      '  text-align:right;padding:0 12px 6px;min-height:16px;',
      '}',
      '[data-cs="char-count"].cs-warn{color:var(--err)}',

      /* ── Level badge ── */
      '[data-cs="badge-rep"]{',
      '  font-size:10px;font-weight:700;padding:1px 6px;border-radius:4px;',
      '  background:var(--surface-2);color:var(--muted);letter-spacing:.03em;flex-shrink:0;',
      '}',
      '[data-cs="badge-rep"].cs-lv1{background:#1c2030;color:#94a3b8}',
      '[data-cs="badge-rep"].cs-lv2{background:#1a2a1a;color:#86efac}',
      '[data-cs="badge-rep"].cs-lv3{background:#1a2535;color:#60a5fa}',
      '[data-cs="badge-rep"].cs-lv4{background:#251a35;color:#c084fc}',
      '[data-cs="badge-rep"].cs-lv5{background:#2a1a1a;color:#f97316}',
      '[data-cs="badge-rep"].cs-lv6{background:#2a2000;color:#facc15}',
      '[data-cs="badge-rep"].cs-lv7{background:#001a2a;color:#22d3ee}',
      '[data-cs="badge-rep"].cs-lv8{background:#1a0a2a;color:#e879f9}',
      '[data-cs="badge-rep"].cs-lv9{background:#2a0a0a;color:#f87171}',
      '[data-cs="badge-rep"].cs-lv10{background:linear-gradient(90deg,#78350f,#1e1b4b);color:#fde68a}',
      '[data-cs="badge-rep"].cs-admin{background:#1a0505;color:#ef4444}',
      '[data-cs="badge-rep"].cs-guest{background:var(--surface-2);color:var(--muted)}',
      /* @mention highlight */
      '[data-cs="mention"]{color:var(--accent);font-weight:600}',
      '[data-cs="msg"][data-mine] [data-cs="mention"]{color:inherit;opacity:.85}',

      /* ── Auth overlay ── */
      '[data-cs="auth-overlay"]{',
      '  position:absolute;inset:0;background:var(--surface);',
      '  display:flex;flex-direction:column;padding:20px 20px 16px;',
      '  transform:translateY(100%);transition:transform 320ms cubic-bezier(.4,0,.2,1);',
      '  z-index:10;overflow-y:auto;',
      '}',
      '[data-cs="auth-overlay"].cs-open{transform:translateY(0)}',
      '[data-cs="auth-tabs"]{display:flex;gap:0;margin-bottom:18px;border-bottom:1px solid var(--border)}',
      '[data-cs="auth-tab"]{',
      '  flex:1;padding:8px 0;font-size:13px;font-weight:600;cursor:pointer;',
      '  background:none;border:none;color:var(--muted);',
      '  border-bottom:2px solid transparent;margin-bottom:-1px;transition:color .15s,border-color .15s;',
      '}',
      '[data-cs="auth-tab"].cs-active{color:var(--accent);border-bottom-color:var(--accent)}',
      '[data-cs="auth-form"]{display:flex;flex-direction:column;gap:12px}',
      '[data-cs="auth-form"].cs-hidden{display:none}',
      '[data-cs="auth-label"]{font-size:12px;font-weight:600;color:var(--muted);margin-bottom:3px}',
      '[data-cs="auth-input"]{',
      '  width:100%;border:1.5px solid var(--border);border-radius:10px;',
      '  padding:10px 12px;font-size:13px;font-family:inherit;',
      '  background:var(--surface-2);color:var(--fg);outline:none;',
      '  box-sizing:border-box;transition:border-color .2s;',
      '}',
      '[data-cs="auth-input"]:focus{border-color:var(--accent)}',
      '[data-cs="auth-err"]{',
      '  font-size:12px;color:var(--err);min-height:16px;',
      '}',
      '[data-cs="auth-submit"]{',
      '  width:100%;padding:11px;background:var(--accent);color:#000;',
      '  border:none;border-radius:10px;font-size:14px;font-weight:700;',
      '  cursor:pointer;transition:filter .15s;',
      '}',
      '[data-cs="auth-submit"]:hover{filter:brightness(1.1)}',
      '[data-cs="auth-submit"]:disabled{opacity:.6;cursor:default}',
      '[data-cs="auth-guest"]{',
      '  text-align:center;font-size:12px;color:var(--muted);',
      '  margin-top:12px;cursor:pointer;background:none;border:none;width:100%;',
      '}',
      '[data-cs="auth-guest"]:hover{color:var(--fg)}',
      '[data-cs="auth-user"]{',
      '  display:flex;align-items:center;gap:6px;padding:4px 6px;',
      '  border-radius:8px;cursor:pointer;background:none;border:none;',
      '  color:var(--fg);font-size:12px;font-weight:600;transition:background .15s;',
      '}',
      '[data-cs="auth-user"]:hover{background:var(--surface-2)}',
      '[data-cs="auth-user"] svg{width:14px;height:14px;fill:currentColor;opacity:.6}',

      /* ── DM panel ── */
      '[data-cs="dm-panel"]{position:absolute;inset:0;z-index:5;background:var(--surface);display:flex;flex-direction:column;transform:translateX(105%);opacity:0;pointer-events:none;transition:transform .22s cubic-bezier(.4,0,.2,1),opacity .22s}',
      '[data-cs="dm-panel"].cs-open{transform:translateX(0);opacity:1;pointer-events:all}',
      '[data-cs="dm-hdr"]{display:flex;align-items:center;gap:8px;padding:0 14px;height:46px;border-bottom:1px solid var(--border);flex-shrink:0;background:linear-gradient(135deg,#1a1d28 0%,#0e1017 100%)}',
      '@media(prefers-color-scheme:light){[data-cs="dm-hdr"]{background:linear-gradient(135deg,#c47f0a 0%,#e09612 100%)}}',
      '[data-cs="dm-hdr-title"]{flex:1;font-weight:600;font-size:14px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--fg)}',
      '@media(prefers-color-scheme:light){[data-cs="dm-hdr-title"]{color:#fff}}',
      '[data-cs="dm-list"]{flex:1;overflow-y:auto}',
      '[data-cs="dm-list"].cs-hidden,[data-cs="dm-empty"].cs-hidden,[data-cs="dm-conv"].cs-hidden,[data-cs="dm-footer"].cs-hidden{display:none}',
      '[data-cs="dm-item"]{display:flex;align-items:center;gap:10px;padding:11px 14px;cursor:pointer;border-bottom:1px solid var(--border);transition:background .12s}',
      '[data-cs="dm-item"]:hover{background:var(--surface-2)}',
      '[data-cs="dm-item-av"]{width:34px;height:34px;border-radius:50%;display:flex;align-items:center;justify-content:center;font-weight:700;font-size:12px;color:#fff;flex-shrink:0}',
      '[data-cs="dm-item-info"]{flex:1;min-width:0}',
      '[data-cs="dm-item-name"]{font-weight:600;font-size:13px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}',
      '[data-cs="dm-item-preview"]{font-size:12px;color:var(--muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;margin-top:2px}',
      '[data-cs="dm-item-badge"]{min-width:16px;height:16px;border-radius:8px;background:var(--accent);color:#fff;font-size:10px;font-weight:700;display:flex;align-items:center;justify-content:center;padding:0 4px;flex-shrink:0}',
      '[data-cs="dm-empty"]{flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:8px;color:var(--muted);font-size:13px;padding:24px;text-align:center}',
      '[data-cs="dm-conv"]{flex:1;overflow-y:auto;padding:12px 14px;display:flex;flex-direction:column;gap:6px}',
      '[data-cs="dm-bbl-row"]{display:flex;flex-direction:column;max-width:78%}',
      '[data-cs="dm-bbl-row"][data-mine]{align-self:flex-end}',
      '[data-cs="dm-bbl"]{padding:8px 11px;border-radius:14px 14px 14px 3px;font-size:13px;line-height:1.5;background:var(--surface-2);word-break:break-word}',
      '[data-cs="dm-bbl-row"][data-mine] [data-cs="dm-bbl"]{background:var(--accent);color:#fff;border-radius:14px 14px 3px 14px}',
      '[data-cs="dm-bbl-time"]{font-size:10px;color:var(--muted);margin-top:2px;align-self:flex-end}',
      '[data-cs="dm-footer"]{border-top:1px solid var(--border);padding:8px 10px;display:flex;gap:8px;align-items:flex-end;flex-shrink:0;background:var(--surface)}',
      '[data-cs="dm-input"]{flex:1;border:1.5px solid var(--border);border-radius:22px;padding:10px 14px;font-size:13px;font-family:inherit;resize:none;outline:none;overflow-y:auto;min-height:44px;max-height:88px;background:var(--surface-2);color:var(--fg);box-sizing:border-box;transition:border-color .2s}',
      '[data-cs="dm-input"]:focus{border-color:var(--accent)}',
      '[data-cs="dm-send"]{width:44px;height:44px;min-width:44px;border-radius:50%;border:none;cursor:pointer;background:var(--accent);display:flex;align-items:center;justify-content:center;transition:filter .15s;flex-shrink:0}',
      '[data-cs="dm-send"]:hover{filter:brightness(1.1)}',
      '[data-cs="dm-send"] svg{width:18px;height:18px;fill:#000;transform:translateX(1px)}',
      '[data-cs="dm-unread"]{position:absolute;top:-3px;right:-3px;min-width:14px;height:14px;border-radius:7px;background:var(--err);color:#fff;font-size:9px;font-weight:700;display:none;align-items:center;justify-content:center;padding:0 3px;border:1.5px solid var(--bg)}',
      '[data-cs="dm-unread"].cs-vis{display:flex}',
      '[data-cs="dm-msg-btn"]:hover{color:var(--accent)}',

      /* ── Channel switcher panel ── */
      '[data-cs="ch-panel"]{position:absolute;inset:0;z-index:6;background:var(--surface);display:flex;flex-direction:column;transform:translateX(-105%);opacity:0;pointer-events:none;transition:transform .22s cubic-bezier(.4,0,.2,1),opacity .22s}',
      '[data-cs="ch-panel"].cs-open{transform:translateX(0);opacity:1;pointer-events:all}',
      '[data-cs="ch-hdr"]{display:flex;align-items:center;gap:8px;padding:0 14px;height:46px;border-bottom:1px solid var(--border);flex-shrink:0;background:linear-gradient(135deg,#1a1d28 0%,#0e1017 100%)}',
      '@media(prefers-color-scheme:light){[data-cs="ch-hdr"]{background:linear-gradient(135deg,#c47f0a 0%,#e09612 100%)}}',
      '[data-cs="ch-hdr-title"]{flex:1;font-weight:600;font-size:14px;color:var(--fg)}',
      '@media(prefers-color-scheme:light){[data-cs="ch-hdr-title"]{color:#fff}}',
      '[data-cs="ch-list"]{flex:1;overflow-y:auto}',
      '[data-cs="ch-item"]{display:flex;align-items:center;gap:10px;padding:12px 14px;cursor:pointer;border-bottom:1px solid var(--border);transition:background .12s}',
      '[data-cs="ch-item"]:hover{background:var(--surface-2)}',
      '[data-cs="ch-item"].cs-active{background:var(--accent-dim);border-left:3px solid var(--accent)}',
      '[data-cs="ch-av"]{width:36px;height:36px;border-radius:50%;background:var(--accent);display:flex;align-items:center;justify-content:center;font-weight:700;font-size:13px;color:#fff;flex-shrink:0}',
      '[data-cs="ch-info"]{flex:1;min-width:0}',
      '[data-cs="ch-name"]{font-weight:600;font-size:13px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}',
      '[data-cs="ch-desc"]{font-size:11px;color:var(--muted);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;margin-top:2px}',

      /* ── Bot card message ── */
      '[data-cs="bot-card"]{',
      '  display:flex;flex-direction:column;gap:6px;',
      '  background:var(--surface-2);border-radius:12px;padding:10px;margin:6px 0;',
      '  border:1px solid var(--border);overflow:hidden;',
      '  animation:cs-fadein .2s ease;',
      '}',
      '[data-cs="bot-card-hdr"]{display:flex;align-items:center;gap:8px;margin-bottom:4px}',
      '[data-cs="bot-card-av"]{width:28px;height:28px;border-radius:50%;background:var(--accent);display:flex;align-items:center;justify-content:center;font-size:11px;font-weight:700;color:#fff;flex-shrink:0}',
      '[data-cs="bot-card-name"]{font-size:12px;font-weight:700;color:var(--muted)}',
      '[data-cs="bot-cards-wrap"]{display:flex;gap:8px;overflow-x:auto;scrollbar-width:thin;padding-bottom:4px}',
      '[data-cs="bot-card-item"]{',
      '  flex-shrink:0;width:160px;border-radius:10px;overflow:hidden;',
      '  background:var(--surface);border:1px solid var(--border);',
      '  display:flex;flex-direction:column;cursor:pointer;',
      '  transition:transform .15s,box-shadow .15s;',
      '}',
      '[data-cs="bot-card-item"]:hover{transform:translateY(-2px);box-shadow:0 4px 16px rgba(0,0,0,.3)}',
      '[data-cs="bot-card-img"]{width:100%;height:100px;object-fit:cover;background:var(--surface-2);display:block}',
      '[data-cs="bot-card-body"]{padding:8px}',
      '[data-cs="bot-card-title"]{font-size:12px;font-weight:700;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;margin-bottom:4px}',
      '[data-cs="bot-card-desc"]{font-size:11px;color:var(--muted);display:-webkit-box;-webkit-line-clamp:2;-webkit-box-orient:vertical;overflow:hidden}',

      /* ── Responsive (mobile: full-width panel) ── */
      '@media(max-width:420px){',
      '  [data-cs="root"][data-pos="bottom-right"],',
      '  [data-cs="root"][data-pos="bottom-left"]{bottom:0;right:0;left:0}',
      '  [data-cs="root"][data-pos="top-right"],',
      '  [data-cs="root"][data-pos="top-left"]{top:0;right:0;left:0}',
      '  [data-cs="panel"]{',
      '    width:100vw;max-height:75vh;',
      '    border-radius:var(--radius) var(--radius) 0 0;',
      '    bottom:calc(var(--btn) + 10px);transform-origin:bottom center;',
      '  }',
      '  [data-cs="btn"]{margin:10px auto;max-width:calc(100vw - 20px)}',
      '}',
    ].join('\n');
    document.head.appendChild(s);
  }

  // ── Constants ────────────────────────────────────────────────────────────────

  // renderContent: applies @mention spans to HTML-escaped server content.
  // The content is already escaped (no XSS risk), newlines → <br>.
  function renderContent(content) {
    return (content || '')
      .replace(/@(\w+)/g, '<span data-cs="mention">@$1</span>')
      .replace(/\n/g, '<br>');
  }

  var EMOJI_CATS = [
    { key:'recent', icon:'🕐', label:'Recent',  emojis:[] },
    { key:'faces',  icon:'😊', label:'Faces',   emojis:['😀','😂','😍','🥰','😎','🤔','😮','😢','😡','🤩','🤣','😅','🥳','😴','🫡','😤','🤯','🥺','😇','🤗','😬','🤭','🫠','😑','😐','🙂','😶','😳','🫢','😲'] },
    { key:'hands',  icon:'👏', label:'Hands',   emojis:['👍','👎','👏','🙌','🤝','🤜','🤛','✊','👊','🫶','🤲','🙏','💪','✌️','🤞','🫰','🤟','🤘','🤙','👋','🤚','✋','🖐️','👌','🤌','🤏'] },
    { key:'love',   icon:'❤️', label:'Love',    emojis:['❤️','🧡','💛','💚','💙','💜','🖤','🤍','🤎','❤️‍🔥','💕','💞','💓','💗','💖','💘','💝','💟','♥️','🫀','💔','❣️'] },
    { key:'animals',icon:'🐼', label:'Animals', emojis:['🐼','🐶','🐱','🐭','🐹','🐰','🦊','🐻','🐨','🐯','🦁','🐮','🐷','🐸','🐵','🐔','🐧','🐦','🦆','🦅','🦉','🦇','🐺','🐗','🐴','🦄','🐝','🪱','🐛','🦋'] },
    { key:'food',   icon:'🍎', label:'Food',    emojis:['🍎','🍊','🍋','🍇','🍓','🫐','🍒','🍑','🥭','🍍','🥥','🥝','🍅','🥑','🫛','🥦','🥕','🌽','🫚','🧄','🧅','🥔','🫙','🧅','🍕','🍔','🌮','🌯','🥗','🍜'] },
    { key:'other',  icon:'🚀', label:'Other',   emojis:['🚀','🌍','🎯','💡','🔥','⭐','💯','🎉','✅','❌','⚠️','💬','🏆','🎮','🎵','🎸','🎤','📱','💻','⌨️','🖥️','📷','📸','🎥','📺','⚽','🏀','🏈','⚾','🎱'] },
  ];

  // ── Helpers ──────────────────────────────────────────────────────────────────

  function h(tag, attrs) {
    var el = document.createElement(tag);
    if (attrs) {
      Object.keys(attrs).forEach(function (k) {
        var v = attrs[k];
        if (k === 'class') {
          el.className = v;
        } else if (k === 'text') {
          el.textContent = v;
        } else {
          el.setAttribute(k, v);
        }
      });
    }
    return el;
  }

  function cs(name, attrs) {
    var merged = { 'data-cs': name };
    if (attrs) Object.keys(attrs).forEach(function (k) { merged[k] = attrs[k]; });
    return h('div', merged);
  }

  function timeStr(d) {
    return new Date(d).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }

  function initials(name) {
    return (name || '?').substring(0, 2).toUpperCase();
  }

  function flagEmoji(cc) {
    if (!cc || cc.length !== 2) return '';
    return String.fromCodePoint(cc.charCodeAt(0) + 127397, cc.charCodeAt(1) + 127397);
  }

  function hashColor(seed) {
    var palette = ['#6366f1','#8b5cf6','#ec4899','#f43f5e','#0ea5e9','#14b8a6','#f97316','#22c55e'];
    var n = 0;
    for (var i = 0; i < seed.length; i++) n = (Math.imul(31, n) + seed.charCodeAt(i)) | 0;
    return palette[Math.abs(n) % palette.length];
  }

  // Level names: index 0 = LV.1 … index 9 = LV.10
  var LEVEL_NAMES_ZH = ['初入','探索','进阶','精英','卓越','大师','宗师','传奇','至尊','永恒'];
  var LEVEL_NAMES_EN = ['Novice','Explorer','Advanced','Elite','Premium','Master','Grandmaster','Legend','Supreme','Eternal'];

  function _makeLevelBadge(badge, isAnon, lang) {
    var el = h('span', { 'data-cs': 'badge-rep' });
    var l = lang === 'en' ? 'en' : 'zh';
    if (isAnon) {
      el.classList.add('cs-guest');
      el.setAttribute('data-badge-raw', '__anon__');
      el.textContent = l === 'en' ? 'LV.0 Guest' : 'LV.0 访客';
      return el;
    }
    if (!badge) return null;
    el.setAttribute('data-badge-raw', badge);
    if (badge === 'LV.MAX') {
      el.classList.add('cs-admin');
      el.textContent = l === 'en' ? 'Admin Founder' : '管理员 创世者';
      return el;
    }
    var m = badge.match(/^LV\.(\d+)$/);
    if (!m) return null;
    var lv = parseInt(m[1], 10);
    el.classList.add('cs-lv' + lv);
    var names = l === 'en' ? LEVEL_NAMES_EN : LEVEL_NAMES_ZH;
    el.textContent = badge + ' ' + (names[lv - 1] || badge);
    return el;
  }

  // ── i18n ─────────────────────────────────────────────────────────────────────

  var WL = {
    zh: {
      placeholder:'说点什么…', account:'账号', minimise:'最小化', dm:'私信',
      report:'举报', dismiss:'关闭', send:'发送', online:' 在线',
      sign_in_up:'登录 / 注册', sign_in:'登录', sign_up:'注册',
      sign_in_ing:'登录中…', sign_up_ing:'注册中…',
      sign_in_fail:'登录失败', sign_up_fail:'注册失败',
      sign_out:'退出登录', net_err:'网络错误，请重试',
      fill_required:'请填写用户名和密码',
      username_short:'用户名至少3个字符', password_short:'密码至少6位',
      username:'用户名', password:'密码',
      email:'邮箱（选填）', reg_username_ph:'3-30个字符',
      email_ph:'you@example.com', reg_password_ph:'至少6位',
      new_msgs:function(n){return '↓ '+n+' 条新消息';},
      dm_title:'私信', dm_empty:'暂无私信',
      dm_empty_hint:'点击消息右下角的「私信」可以发起对话',
      dm_placeholder:'发私信…', dm_send:'发送',
      back:'返回', close:'关闭',
      guest:'以访客身份继续',
    },
    en: {
      placeholder:'Say something…', account:'Account', minimise:'Minimise', dm:'DMs',
      report:'Report', dismiss:'Dismiss', send:'Send', online:' online',
      sign_in_up:'Sign in / Sign up', sign_in:'Sign in', sign_up:'Sign up',
      sign_in_ing:'Signing in…', sign_up_ing:'Signing up…',
      sign_in_fail:'Sign in failed', sign_up_fail:'Sign up failed',
      sign_out:'Sign out', net_err:'Network error, please retry',
      fill_required:'Please enter username and password',
      username_short:'Username must be at least 3 characters',
      password_short:'Password must be at least 6 characters',
      username:'Username', password:'Password',
      email:'Email (optional)', reg_username_ph:'3-30 chars',
      email_ph:'you@example.com', reg_password_ph:'Min 6 chars',
      new_msgs:function(n){return '↓ '+n+' new messages';},
      dm_title:'Messages', dm_empty:'No messages yet',
      dm_empty_hint:'Click DM on a message to start a conversation',
      dm_placeholder:'Send a message…', dm_send:'Send',
      back:'Back', close:'Close',
      guest:'Continue as guest',
    }
  };

  // ── Widget ───────────────────────────────────────────────────────────────────

  function Widget(opts) {
    this.server      = (opts.server || '').replace(/\/$/, '');
    this.channel     = opts.channel  || 'general';
    this.position    = opts.position || 'bottom-right';
    this.title       = opts.title    || '';
    this.notice      = opts.notice   || '';
    this.telegramURL = opts.telegram || '';
    this.accent      = opts.accent   || '';
    this.offset      = Math.min(80, Math.max(8, parseInt(opts.offset, 10) || 20));

    // Language: user localStorage preference → embed attr → default zh
    this._langKey = 'cs:lang:' + (this.server || 'default');
    var savedLang; try { savedLang = localStorage.getItem(this._langKey); } catch (_) {}
    var defaultLang = opts.lang === 'en' ? 'en' : 'zh';
    this.lang = (savedLang === 'zh' || savedLang === 'en') ? savedLang : defaultLang;

    this.ws          = null;
    this.isOpen      = false;
    this.isConnected = false;
    this.retryDelay  = 1000;   // ms; doubles on each failure, capped at 30 s
    this.retryTimer  = null;

    this.myName        = null;   // assigned by server on 'joined' event
    this.myUserID      = null;   // assigned by server on 'joined' event (registered users only)
    this.myAvatarColor = null;   // assigned by server on 'joined' event

    // DM state
    this.dmOpen        = false;
    this.dmConvUserId  = null;
    this.dmConvs       = {};     // userId → { username, avatarColor, msgs:[], unread:0 }

    this.unread      = 0;
    this.maxLen      = 200;
    this.emojiOpen   = false;
    this.nearBottom  = true;
    this.pendingNew  = 0;

    // Auth state
    this._jwtKey     = 'cs:jwt:' + this.server;
    this.jwt         = null;
    this.currentUser = null;

    // Channel switcher state
    this.channels    = [];
    this.chPanelOpen = false;

    injectCSS();
    this._buildDOM();
    this._loadAuth();  // restore saved session, then connect
    this._loadChannels();
  }

  Widget.prototype = {

    wt: function(key) { var d = WL[this.lang] || WL.zh; return d[key] || key; },

    _toggleLang: function () {
      this.lang = this.lang === 'zh' ? 'en' : 'zh';
      try { localStorage.setItem(this._langKey, this.lang); } catch (_) {}
      this._applyLang();
    },

    _applyLang: function () {
      var self = this;
      // Header
      if (this.langBtn) this.langBtn.textContent = this.lang === 'en' ? '中' : 'EN';
      if (this.titleEl && !this.title) this.titleEl.textContent = this.lang === 'en' ? 'Chat' : '聊天';
      if (this.btnTitleEl && !this.title) this.btnTitleEl.textContent = this.lang === 'en' ? 'Chat' : '聊天';
      if (this.onlineSpan) this.onlineSpan.textContent = (this.onlineCount || 0) + this.wt('online');
      if (this.minBtn)  { this.minBtn.title = this.wt('minimise'); this.minBtn.setAttribute('aria-label', this.wt('minimise')); }
      if (this.dmBtnEl) { this.dmBtnEl.title = this.wt('dm'); this.dmBtnEl.setAttribute('aria-label', this.wt('dm')); }
      this._updateAuthButton();
      // Input row
      if (this.inputEl) this.inputEl.placeholder = this.wt('placeholder');
      if (this.sendBtn) { this.sendBtn.title = this.wt('send'); this.sendBtn.setAttribute('aria-label', this.wt('send')); }
      // Notice
      if (this.noticeCls) this.noticeCls.setAttribute('aria-label', this.wt('dismiss'));
      // New-message button
      if (this.newMsgBtn && this.pendingNew > 0) this.newMsgBtn.textContent = this.wt('new_msgs')(this.pendingNew);
      // Auth overlay
      if (this.authOverlay) this.authOverlay.setAttribute('aria-label', this.wt('sign_in_up'));
      if (this._authTabLogin) this._authTabLogin.textContent = this.wt('sign_in');
      if (this._authTabReg)   this._authTabReg.textContent   = this.wt('sign_up');
      if (this._loginBtn) this._loginBtn.textContent = this.wt('sign_in');
      if (this._regBtn)   this._regBtn.textContent   = this.wt('sign_up');
      if (this._guestBtn) this._guestBtn.textContent = this.wt('guest');
      // Auth field labels + placeholders
      var fieldDefs = [
        [this._loginUser, this.wt('username'), this.wt('username')],
        [this._loginPass, this.wt('password'), this.wt('password')],
        [this._regUser,   this.wt('username'), this.wt('reg_username_ph')],
        [this._regEmail,  this.wt('email'),    this.wt('email_ph')],
        [this._regPass,   this.wt('password'), this.wt('reg_password_ph')],
      ];
      fieldDefs.forEach(function (fd) {
        if (!fd[0]) return;
        var lbl = fd[0].previousElementSibling;
        if (lbl) lbl.textContent = fd[1];
        fd[0].placeholder = fd[2];
      });
      // DM panel
      if (this.dmBackBtn) { this.dmBackBtn.title = this.wt('back'); this.dmBackBtn.setAttribute('aria-label', this.wt('back')); }
      if (this.dmHdrTitle && !this.dmConvUserId) this.dmHdrTitle.textContent = this.wt('dm_title');
      if (this.dmEmptyEl) this.dmEmptyEl.innerHTML = '<svg viewBox="0 0 24 24" style="width:40px;height:40px;opacity:.3;fill:currentColor"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/></svg><div>' + this.wt('dm_empty') + '</div><div style="font-size:11px;margin-top:4px">' + this.wt('dm_empty_hint') + '</div>';
      if (this.dmInputEl) { this.dmInputEl.placeholder = this.wt('dm_placeholder'); this.dmInputEl.setAttribute('aria-label', this.wt('dm_title')); }
      // Update existing level badges, report buttons, DM buttons in message list
      var selfLang = this.lang;
      Array.prototype.forEach.call(this.msgs.querySelectorAll('[data-badge-raw]'), function(el) {
        var raw = el.getAttribute('data-badge-raw');
        if (raw === '__anon__') {
          el.textContent = selfLang === 'en' ? 'LV.0 Guest' : 'LV.0 访客';
        } else if (raw === 'LV.MAX') {
          el.textContent = selfLang === 'en' ? 'Admin Founder' : '管理员 创世者';
        } else {
          var m = raw.match(/^LV\.(\d+)$/);
          if (m) {
            var lv = parseInt(m[1], 10);
            var names = selfLang === 'en' ? LEVEL_NAMES_EN : LEVEL_NAMES_ZH;
            el.textContent = raw + ' ' + (names[lv - 1] || raw);
          }
        }
      });
      Array.prototype.forEach.call(self.msgs.querySelectorAll('[data-cs="report-btn"]'), function(el) {
        el.textContent = self.wt('report');
      });
      Array.prototype.forEach.call(self.msgs.querySelectorAll('[data-cs="dm-msg-btn"]'), function(el) {
        el.textContent = self.wt('dm');
      });
    },

    // ── Build DOM ──────────────────────────────────────────────────────────────

    _buildDOM: function () {
      var self = this;

      var root = this.root = h('div', { 'data-cs': 'root', 'data-pos': this.position });
      root.style.setProperty('--offset', this.offset + 'px');
      if (this.accent && /^#[0-9a-fA-F]{6}$/.test(this.accent)) {
        var ah = this.accent;
        var ar = parseInt(ah.slice(1,3),16), ag = parseInt(ah.slice(3,5),16), ab2 = parseInt(ah.slice(5,7),16);
        root.style.setProperty('--accent', ah);
        root.style.setProperty('--accent-dim', 'rgba('+ar+','+ag+','+ab2+',.15)');
        root.style.setProperty('--accent-glow', 'rgba('+ar+','+ag+','+ab2+',.35)');
      }

      // ── Trigger button (card shape)
      var btn = this.btn = h('button', {
        'data-cs':      'btn',
        'aria-label':   'Open chat',
        'aria-expanded':'false',
      });

      // Icon circle
      var btnIcon = h('div', { 'data-cs': 'btn-icon' });
      btnIcon.innerHTML = '<svg viewBox="0 0 24 24"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/></svg>';

      // Info column: title + preview
      var btnInfo = h('div', { 'data-cs': 'btn-info' });
      var btnTitleEl = this.btnTitleEl = h('div', { 'data-cs': 'btn-title', 'text': this.title || this.wt('placeholder') });
      var btnPreviewEl = this.btnPreviewEl = h('div', { 'data-cs': 'btn-preview', 'text': '' });
      btnInfo.appendChild(btnTitleEl);
      btnInfo.appendChild(btnPreviewEl);

      // Close icon (shows when panel is open)
      var btnCloseIcon = h('div', { 'data-cs': 'btn-close-icon' });
      btnCloseIcon.innerHTML = '<svg viewBox="0 0 24 24"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>';

      btn.appendChild(btnIcon);
      btn.appendChild(btnInfo);
      btn.appendChild(btnCloseIcon);
      btn.addEventListener('click', function () { self.toggle(); });

      // Unread badge (top-right)
      this.badge = cs('badge');
      btn.appendChild(this.badge);

      // DM @ badge (bottom-right, visible when DMs are unread)
      this.dmLauncherBadge = h('div', { 'data-cs': 'dm-badge', 'text': '@' });
      btn.appendChild(this.dmLauncherBadge);

      // ── Panel
      var panel = this.panel = cs('panel');
      panel.setAttribute('role', 'dialog');
      panel.setAttribute('aria-label', this.title);

      // Status bar (yellow = disconnected/reconnecting)
      this.statusBar = cs('status');
      panel.appendChild(this.statusBar);

      // Header
      panel.appendChild(this._buildHeader());

      // Notice
      this.noticeEl = this._buildNotice();
      panel.appendChild(this.noticeEl);

      // Messages wrapper
      var msgsWrap = cs('msgs-wrap');
      this.msgs = cs('msgs');
      this.msgs.addEventListener('scroll', function () { self._onScroll(); });

      this.newMsgBtn = h('button', { 'data-cs': 'new-msgs', 'text': '↓ 新消息' });
      this.newMsgBtn.addEventListener('click', function () {
        self._scrollBottom();
        self._hideNewMsgBtn();
      });

      msgsWrap.appendChild(this.msgs);
      msgsWrap.appendChild(this.newMsgBtn);
      panel.appendChild(msgsWrap);

      // Auth overlay (sits over messages area)
      this.authOverlay = this._buildAuthOverlay();
      msgsWrap.appendChild(this.authOverlay);

      // Footer (emoji grid + input)
      panel.appendChild(this._buildFooter());

      // DM panel (slides in over main content)
      panel.appendChild(this._buildDMPanel());

      // Channel switcher panel
      panel.appendChild(this._buildChPanel());

      root.appendChild(panel);
      root.appendChild(btn);
      document.body.appendChild(root);

      // Close emoji picker on click outside
      document.addEventListener('click', function () {
        if (self.emojiOpen) {
          self.emojiOpen = false;
          self.emojiPicker.classList.remove('cs-open');
        }
      });
    },

    _buildHeader: function () {
      var self = this;
      var hdr = cs('header');

      var wave = this.wave = cs('wave');
      wave.classList.add('cs-off');
      for (var i = 0; i < 5; i++) wave.appendChild(document.createElement('span'));

      var titleEl = this.titleEl = h('span', { 'data-cs': 'title', 'text': this.title || (this.lang === 'en' ? 'Chat' : '聊天') });

      var onlineEl = this.onlineEl = cs('online');
      onlineEl.appendChild(cs('dot'));
      this.onlineSpan = document.createTextNode('0' + this.wt('online'));
      onlineEl.appendChild(this.onlineSpan);

      if (this.telegramURL) {
        var tgBtn = h('button', { 'data-cs': 'hbtn', 'title': 'Telegram', 'aria-label': 'Telegram' });
        tgBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M11.944 0A12 12 0 0 0 0 12a12 12 0 0 0 12 12 12 12 0 0 0 12-12A12 12 0 0 0 12 0a12 12 0 0 0-.056 0zm4.962 7.224c.1-.002.321.023.465.14a.506.506 0 0 1 .171.325c.016.093.036.306.02.472-.18 1.898-.962 6.502-1.36 8.627-.168.9-.499 1.201-.82 1.23-.696.065-1.225-.46-1.9-.902-1.056-.693-1.653-1.124-2.678-1.8-1.185-.78-.417-1.21.258-1.91.177-.184 3.247-2.977 3.307-3.23.007-.032.014-.15-.056-.212s-.174-.041-.249-.024c-.106.024-1.793 1.14-5.061 3.345-.48.33-.913.49-1.302.48-.428-.008-1.252-.241-1.865-.44-.752-.245-1.349-.374-1.297-.789.027-.216.325-.437.893-.663 3.498-1.524 5.83-2.529 6.998-3.014 3.332-1.386 4.025-1.627 4.476-1.635z"/></svg>';
        tgBtn.addEventListener('click', function () { window.open(self.telegramURL, '_blank'); });
        hdr.appendChild(tgBtn);
      }

      // Auth button: shows username when logged in, person icon when logged out
      var authBtn = this.authBtn = h('button', { 'data-cs': 'auth-user', 'title': this.wt('account'), 'aria-label': this.wt('account') });
      authBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg>';
      authBtn.addEventListener('click', function () {
        if (self.currentUser) {
          self._logout();
        } else {
          self._showAuthOverlay();
        }
      });

      var minBtn = this.minBtn = h('button', { 'data-cs': 'hbtn', 'title': this.wt('minimise'), 'aria-label': this.wt('minimise') });
      minBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M19 13H5v-2h14v2z"/></svg>';
      minBtn.addEventListener('click', function () { self.toggle(); });

      // Language toggle button
      var langBtn = this.langBtn = h('button', { 'data-cs': 'hbtn', 'data-lang-btn': '', 'title': 'Language', 'aria-label': 'Switch language' });
      langBtn.textContent = this.lang === 'en' ? '中' : 'EN';
      langBtn.addEventListener('click', function (e) { e.stopPropagation(); self._toggleLang(); });

      // DM button (only shown for registered users, populated after 'joined')
      var dmBtn = this.dmBtnEl = h('button', { 'data-cs': 'hbtn', 'title': this.wt('dm'), 'aria-label': this.wt('dm'), 'style': 'position:relative;display:none' });
      dmBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/></svg>';
      this.dmUnreadBadge = cs('dm-unread');
      dmBtn.appendChild(this.dmUnreadBadge);
      dmBtn.addEventListener('click', function (e) { e.stopPropagation(); self._toggleDMPanel(); });

      // Channels button (shown when multiple channels exist)
      var chBtn = this.chBtnEl = h('button', { 'data-cs': 'hbtn', 'title': '切换频道', 'aria-label': '切换频道', 'style': 'display:none' });
      chBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M3 15h18v-2H3v2zm0 4h18v-2H3v2zm0-8h18V9H3v2zm0-6v2h18V5H3z"/></svg>';
      chBtn.addEventListener('click', function (e) { e.stopPropagation(); self._toggleChPanel(); });

      hdr.appendChild(wave);
      hdr.appendChild(titleEl);
      hdr.appendChild(onlineEl);
      hdr.appendChild(chBtn);
      hdr.appendChild(dmBtn);
      hdr.appendChild(authBtn);
      hdr.appendChild(langBtn);
      hdr.appendChild(minBtn);

      return hdr;
    },

    _buildNotice: function () {
      var self = this;
      var el = cs('notice');
      el.classList.add('cs-hidden');
      if (!this.notice) return el;

      el.classList.remove('cs-hidden');
      var txt = h('span', { 'data-cs': 'notice-text', 'text': this.notice });
      var cls = this.noticeCls = h('button', { 'data-cs': 'notice-close', 'aria-label': this.wt('dismiss'), 'text': '×' });
      cls.addEventListener('click', function () { el.classList.add('cs-hidden'); });
      el.appendChild(txt);
      el.appendChild(cls);
      return el;
    },

    _buildFooter: function () {
      var self = this;
      var footer = cs('footer');

      // Emoji picker with category tabs
      var picker = this.emojiPicker = cs('emoji');

      var tabsRow = h('div', { 'data-cs': 'emoji-tabs' });
      var emojiGrid = this.emojiGrid = h('div', { 'data-cs': 'emoji-grid' });
      var self2 = self;

      function renderEmojiCat(cat) {
        emojiGrid.innerHTML = '';
        cat.emojis.forEach(function (e) {
          var sp = document.createElement('span');
          sp.textContent = e;
          sp.title = e;
          sp.addEventListener('click', function () { self2._insertEmoji(e); });
          emojiGrid.appendChild(sp);
        });
      }

      EMOJI_CATS.forEach(function (cat, idx) {
        var tab = h('button', { 'data-cs': 'etab', 'title': cat.label, 'aria-label': cat.label });
        tab.textContent = cat.icon;
        tab.addEventListener('click', function (e) {
          e.stopPropagation();
          tabsRow.querySelectorAll('[data-cs="etab"]').forEach(function (t) { t.classList.remove('cs-active'); });
          tab.classList.add('cs-active');
          renderEmojiCat(cat);
        });
        if (idx === 1) tab.classList.add('cs-active'); // default: faces
        tabsRow.appendChild(tab);
      });

      picker.appendChild(tabsRow);
      picker.appendChild(emojiGrid);

      // Render default category on first open
      renderEmojiCat(EMOJI_CATS[1]);
      footer.appendChild(picker);

      // Input row
      var row = cs('input-row');

      var emojiBtn = h('button', { 'data-cs': 'ibtn', 'title': 'Emoji', 'aria-label': 'Emoji' });
      emojiBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M11.99 2C6.47 2 2 6.48 2 12s4.47 10 9.99 10C17.52 22 22 17.52 22 12S17.52 2 11.99 2zM12 20c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm3.5-9c.83 0 1.5-.67 1.5-1.5S16.33 8 15.5 8 14 8.67 14 9.5s.67 1.5 1.5 1.5zm-7 0c.83 0 1.5-.67 1.5-1.5S9.33 8 8.5 8 7 8.67 7 9.5 7.67 11 8.5 11zm3.5 6.5c2.33 0 4.31-1.46 5.11-3.5H6.89c.8 2.04 2.78 3.5 5.11 3.5z"/></svg>';
      emojiBtn.addEventListener('click', function (e) {
        e.stopPropagation();
        self._toggleEmoji();
      });

      var inputEl = this.inputEl = h('textarea', {
        'data-cs':    'input',
        'placeholder': this.wt('placeholder'),
        'rows':       '1',
        'aria-label': 'Message',
      });
      inputEl.addEventListener('input', function () { self._onInputChange(); });
      inputEl.addEventListener('keydown', function (e) {
        if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); self._send(); }
      });

      var sendBtn = this.sendBtn = h('button', { 'data-cs': 'send', 'title': this.wt('send'), 'aria-label': this.wt('send') });
      sendBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>';
      sendBtn.addEventListener('click', function () { self._send(); });

      row.appendChild(emojiBtn);
      row.appendChild(inputEl);
      row.appendChild(sendBtn);
      footer.appendChild(row);

      this.charCountEl = cs('char-count');
      footer.appendChild(this.charCountEl);

      return footer;
    },

    // ── WebSocket ──────────────────────────────────────────────────────────────

    _connect: function () {
      var self = this;
      if (this.retryTimer) { clearTimeout(this.retryTimer); this.retryTimer = null; }
      if (this.ws) {
        this.ws.onclose = null;
        this.ws.close();
        this.ws = null;
      }

      var proto = this.server.startsWith('https') ? 'wss' : 'ws';
      var base  = this.server.replace(/^https?:\/\//, '');
      var url   = proto + '://' + base + '/ws/' + this.channel;
      if (this.jwt) url += '?token=' + encodeURIComponent(this.jwt);

      try {
        var ws = this.ws = new WebSocket(url);
        ws.onopen    = function ()  { self._onOpen(); };
        ws.onmessage = function (e) { try { self._onMessage(JSON.parse(e.data)); } catch (_) {} };
        ws.onclose   = function ()  { self._onClose(); };
        ws.onerror   = function ()  {};
      } catch (_) {
        this._scheduleRetry();
      }
    },

    _onOpen: function () {
      this.isConnected = true;
      this.retryDelay  = 1000;
      this._setConnected(true);
    },

    _onClose: function () {
      this.isConnected = false;
      this._setConnected(false);
      this._scheduleRetry();
    },

    _scheduleRetry: function () {
      var self = this;
      this.retryTimer = setTimeout(function () { self._connect(); }, this.retryDelay);
      this.retryDelay = Math.min(this.retryDelay * 2, 30000);
    },

    _onMessage: function (evt) {
      switch (evt.type) {
        case 'joined':
          // Server tells us our own display name so we can mark own messages.
          this.myName = evt.content || null;
          if (evt.user_id) {
            this.myUserID = evt.user_id;
            this.myAvatarColor = evt.avatar_color || null;
            if (this.dmBtnEl) this.dmBtnEl.style.display = '';
          }
          // Apply channel accent color if provided by the server.
          if (evt.accent_color && /^#[0-9a-fA-F]{6}$/.test(evt.accent_color)) {
            var hex = evt.accent_color;
            var r = parseInt(hex.slice(1,3),16),
                g = parseInt(hex.slice(3,5),16),
                b = parseInt(hex.slice(5,7),16);
            this.root.style.setProperty('--accent', hex);
            this.root.style.setProperty('--accent-dim', 'rgba('+r+','+g+','+b+',.15)');
            this.root.style.setProperty('--accent-glow', 'rgba('+r+','+g+','+b+',.35)');
          }
          break;

        case 'dm':
          if (evt.dm) this._onDM(evt.dm);
          break;

        case 'dm_history':
          if (evt.with_user_id != null) this._onDMHistory(evt.with_user_id, evt.dms || []);
          break;

        case 'history':
          if (evt.messages && evt.messages.length) {
            var self = this;
            evt.messages.forEach(function (raw) {
              try {
                self._renderMsg(typeof raw === 'string' ? JSON.parse(raw) : raw);
              } catch (_) {}
            });
            this._scrollBottom();
            // Show last message as button preview when panel is closed
            if (!this.isOpen) {
              var last = evt.messages[evt.messages.length - 1];
              if (last && typeof last !== 'string' && last.content && this.btnPreviewEl) {
                var sender = last.display_name || last.username || '';
                this.btnPreviewEl.textContent = sender ? sender + ': ' + last.content : last.content;
              }
            }
          }
          break;

        case 'message':
          if (evt.message) {
            this._renderMsg(evt.message);
            if (!this.isOpen) {
              this._incUnread(evt.message);
            } else if (this.nearBottom) {
              this._scrollBottom();
            } else {
              this._showNewMsgBtn();
            }
          }
          break;

        case 'online_count':
          if (evt.count != null) this._setOnline(evt.count);
          break;

        case 'system':
          this._addSysMsg(evt.content || evt.error);
          break;

        case 'error':
          this._addSysMsg(evt.error || evt.message, true);
          break;

        case 'deleted':
          if (evt.id) {
            var el = this.msgs.querySelector('[data-msgid="' + evt.id + '"]');
            if (el) el.remove();
          }
          break;

        case 'mention':
          // Another user @mentioned this user in this channel.
          if (evt.message) {
            var sender = evt.message.display_name || '有人';
            this._addSysMsg('@ ' + sender + ' 提及了你');
            if (!this.isOpen) this._incUnread();
          }
          break;

        case 'bot_card':
          if (evt.bot_cards && evt.bot_cards.length) {
            this._renderBotCards(evt.content || 'Bot', evt.avatar_color || '', evt.bot_cards);
            if (!this.isOpen) {
              this._incUnread();
            } else if (this.nearBottom) {
              this._scrollBottom();
            } else {
              this._showNewMsgBtn();
            }
          }
          break;
      }
    },

    // ── Rendering ─────────────────────────────────────────────────────────────

    _renderMsg: function (msg) {
      var mine  = !!(this.myName && msg.display_name === this.myName);
      var name  = msg.display_name || 'Guest';
      var color = msg.avatar_color || hashColor(name);

      // Avatar
      var avatarEl = cs('avatar', { style: 'background:' + color });
      avatarEl.textContent = initials(name);

      // Meta (name + time)
      var metaEl = cs('msg-meta');
      if (msg.country) {
        var flagEl = cs('flag');
        flagEl.textContent = flagEmoji(msg.country);
        metaEl.appendChild(flagEl);
      }
      var nameEl = h('span', { 'data-cs': 'name', 'text': name });
      if (msg.is_anonymous) nameEl.classList.add('cs-anon');
      metaEl.appendChild(nameEl);
      // Level badge
      var lvBadgeEl = _makeLevelBadge(msg.badge, msg.is_anonymous, this.lang);
      if (lvBadgeEl) metaEl.appendChild(lvBadgeEl);
      var timeEl = h('span', { 'data-cs': 'time', 'text': timeStr(msg.created_at) });
      metaEl.appendChild(timeEl);

      // Bubble — HTML-escaped on server, safe for innerHTML; highlight @mentions
      var bubble = cs('bubble');
      bubble.innerHTML = renderContent(msg.content);

      // Bubble row: bubble + inline action buttons (report / DM)
      var bubbleRow = h('div', { 'data-cs': 'bubble-row' });
      bubbleRow.appendChild(bubble);
      if (!mine) {
        var self = this;
        var actionsEl = h('div', { 'data-cs': 'msg-actions' });
        var reportBtn = h('button', { 'data-cs': 'report-btn', 'text': self.wt('report') });
        reportBtn.addEventListener('click', function () { self._report(msg.id); });
        actionsEl.appendChild(reportBtn);
        // DM button — only for registered users (non-anonymous with a user_id)
        if (!msg.is_anonymous && msg.user_id) {
          var dmMsgBtn = h('button', { 'data-cs': 'dm-msg-btn', 'text': self.wt('dm') });
          var _uid = msg.user_id, _uname = msg.display_name, _ucolor = msg.avatar_color;
          dmMsgBtn.addEventListener('click', function () { self._openDMConv(_uid, _uname, _ucolor); });
          actionsEl.appendChild(dmMsgBtn);
        }
        bubbleRow.appendChild(actionsEl);
      }
      var msgWrap = cs('msg-wrap');
      msgWrap.appendChild(metaEl);
      msgWrap.appendChild(bubbleRow);

      // Wrapper row
      var wrapper = cs('msg');
      if (msg.id) wrapper.setAttribute('data-msgid', msg.id);
      if (mine)   wrapper.setAttribute('data-mine', '');
      wrapper.appendChild(avatarEl);
      wrapper.appendChild(msgWrap);

      this.msgs.appendChild(wrapper);
    },

    // ── Auth overlay ──────────────────────────────────────────────────────────

    _buildAuthOverlay: function () {
      var self = this;
      var overlay = cs('auth-overlay');
      overlay.setAttribute('role', 'dialog');
      overlay.setAttribute('aria-label', this.wt('sign_in_up'));

      // Tabs
      var tabs = cs('auth-tabs');
      var tabLogin = this._authTabLogin = h('button', { 'data-cs': 'auth-tab', 'text': this.wt('sign_in') });
      var tabReg   = this._authTabReg   = h('button', { 'data-cs': 'auth-tab', 'text': this.wt('sign_up') });
      tabLogin.classList.add('cs-active');
      tabs.appendChild(tabLogin);
      tabs.appendChild(tabReg);

      // Login form
      var loginForm = cs('auth-form');
      var loginUser = this._loginUser = this._authField(loginForm, this.wt('username'), 'text', 'username', this.wt('username'));
      var loginPass = this._loginPass = this._authField(loginForm, this.wt('password'), 'password', 'current-password', this.wt('password'));
      var loginErr  = cs('auth-err'); loginForm.appendChild(loginErr);
      var loginBtn  = this._loginBtn = h('button', { 'data-cs': 'auth-submit', 'text': this.wt('sign_in'), 'type': 'button' });
      loginForm.appendChild(loginBtn);

      // Register form
      var regForm  = cs('auth-form');
      regForm.classList.add('cs-hidden');
      var regUser  = this._regUser  = this._authField(regForm, this.wt('username'), 'text', 'username', this.wt('reg_username_ph'));
      var regEmail = this._regEmail = this._authField(regForm, this.wt('email'), 'email', 'email', this.wt('email_ph'));
      var regPass  = this._regPass  = this._authField(regForm, this.wt('password'), 'password', 'new-password', this.wt('reg_password_ph'));
      var regErr   = cs('auth-err'); regForm.appendChild(regErr);
      var regBtn   = this._regBtn = h('button', { 'data-cs': 'auth-submit', 'text': this.wt('sign_up'), 'type': 'button' });
      regForm.appendChild(regBtn);

      // Guest link
      var guestBtn = this._guestBtn = h('button', { 'data-cs': 'auth-guest', 'text': this.wt('guest'), 'type': 'button' });

      overlay.appendChild(tabs);
      overlay.appendChild(loginForm);
      overlay.appendChild(regForm);
      overlay.appendChild(guestBtn);

      // Tab switching
      tabLogin.addEventListener('click', function () {
        tabLogin.classList.add('cs-active');
        tabReg.classList.remove('cs-active');
        loginForm.classList.remove('cs-hidden');
        regForm.classList.add('cs-hidden');
        loginErr.textContent = '';
        loginUser.focus();
      });
      tabReg.addEventListener('click', function () {
        tabReg.classList.add('cs-active');
        tabLogin.classList.remove('cs-active');
        regForm.classList.remove('cs-hidden');
        loginForm.classList.add('cs-hidden');
        regErr.textContent = '';
        regUser.focus();
      });

      // Enter key submits
      [loginUser, loginPass].forEach(function (el) {
        el.addEventListener('keydown', function (e) { if (e.key === 'Enter') loginBtn.click(); });
      });
      [regUser, regEmail, regPass].forEach(function (el) {
        el.addEventListener('keydown', function (e) { if (e.key === 'Enter') regBtn.click(); });
      });

      loginBtn.addEventListener('click', function () {
        self._doLogin(loginUser.value.trim(), loginPass.value, loginErr, loginBtn);
      });
      regBtn.addEventListener('click', function () {
        self._doRegister(regUser.value.trim(), regEmail.value.trim(), regPass.value, regErr, regBtn);
      });
      guestBtn.addEventListener('click', function () { self._hideAuthOverlay(); });

      return overlay;
    },

    _authField: function (parent, label, type, autocomplete, placeholder) {
      var wrap = document.createElement('div');
      var lbl = h('div', { 'data-cs': 'auth-label', 'text': label });
      var inp = h('input', { 'data-cs': 'auth-input', type: type, autocomplete: autocomplete, placeholder: placeholder });
      wrap.appendChild(lbl);
      wrap.appendChild(inp);
      parent.appendChild(wrap);
      return inp;
    },

    _showAuthOverlay: function () {
      this.authOverlay.classList.add('cs-open');
      var firstInput = this.authOverlay.querySelector('[data-cs="auth-input"]');
      if (firstInput) setTimeout(function () { firstInput.focus(); }, 320);
    },

    _hideAuthOverlay: function () {
      this.authOverlay.classList.remove('cs-open');
    },

    _doLogin: function (username, password, errEl, btn) {
      var self = this;
      if (!username || !password) { errEl.textContent = self.wt('fill_required'); return; }
      errEl.textContent = '';
      btn.disabled = true;
      btn.textContent = self.wt('sign_in_ing');

      fetch(this.server + '/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: username, password: password }),
      })
        .then(function (r) { return r.json().then(function (d) { return { ok: r.ok, d: d }; }); })
        .then(function (res) {
          btn.disabled = false;
          btn.textContent = self.wt('sign_in');
          if (!res.ok) { errEl.textContent = res.d.error || self.wt('sign_in_fail'); return; }
          self._setAuthState(res.d.token, res.d.user);
          self._hideAuthOverlay();
        })
        .catch(function () {
          btn.disabled = false;
          btn.textContent = self.wt('sign_in');
          errEl.textContent = self.wt('net_err');
        });
    },

    _doRegister: function (username, email, password, errEl, btn) {
      var self = this;
      if (!username || !password) { errEl.textContent = self.wt('fill_required'); return; }
      if (username.length < 3) { errEl.textContent = self.wt('username_short'); return; }
      if (password.length < 6) { errEl.textContent = self.wt('password_short'); return; }
      errEl.textContent = '';
      btn.disabled = true;
      btn.textContent = self.wt('sign_up_ing');

      fetch(this.server + '/api/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username: username, email: email, password: password }),
      })
        .then(function (r) { return r.json().then(function (d) { return { ok: r.ok, d: d }; }); })
        .then(function (res) {
          btn.disabled = false;
          btn.textContent = self.wt('sign_up');
          if (!res.ok) { errEl.textContent = res.d.error || self.wt('sign_up_fail'); return; }
          self._setAuthState(res.d.token, res.d.user);
          self._hideAuthOverlay();
        })
        .catch(function () {
          btn.disabled = false;
          btn.textContent = self.wt('sign_up');
          errEl.textContent = self.wt('net_err');
        });
    },

    _setAuthState: function (jwt, user) {
      this.jwt = jwt;
      this.currentUser = user;
      try { localStorage.setItem(this._jwtKey, jwt); } catch (_) {}
      this._updateAuthButton();
      // Clear messages and reconnect as registered user
      this.msgs.innerHTML = '';
      this.myName = null;
      this.myUserID = null;
      this.myAvatarColor = null;
      if (this.dmBtnEl) this.dmBtnEl.style.display = 'none';
      this._connect();
    },

    _logout: function () {
      this.jwt = null;
      this.currentUser = null;
      try { localStorage.removeItem(this._jwtKey); } catch (_) {}
      this._updateAuthButton();
      this.msgs.innerHTML = '';
      this.myName = null;
      this.myUserID = null;
      this.myAvatarColor = null;
      this.dmConvs = {};
      if (this.dmBtnEl) this.dmBtnEl.style.display = 'none';
      if (this.dmOpen) { this.dmOpen = false; this.dmPanelEl.classList.remove('cs-open'); }
      this._connect();
    },

    _updateAuthButton: function () {
      if (!this.authBtn) return;
      if (this.currentUser) {
        this.authBtn.textContent = this.currentUser.username;
        this.authBtn.title = this.wt('sign_out');
        this.authBtn.setAttribute('aria-label', this.wt('sign_out'));
      } else {
        this.authBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M12 12c2.21 0 4-1.79 4-4s-1.79-4-4-4-4 1.79-4 4 1.79 4 4 4zm0 2c-2.67 0-8 1.34-8 4v2h16v-2c0-2.66-5.33-4-8-4z"/></svg>';
        this.authBtn.title = this.wt('sign_in_up');
        this.authBtn.setAttribute('aria-label', this.wt('sign_in_up'));
      }
    },

    // ── Auth restore ──────────────────────────────────────────────────────────

    _loadAuth: function () {
      var self = this;
      var saved;
      try { saved = localStorage.getItem(this._jwtKey); } catch (_) {}
      if (!saved) { this._connect(); return; }

      // Verify saved token is still valid
      fetch(this.server + '/api/auth/me', {
        headers: { 'Authorization': 'Bearer ' + saved },
      })
        .then(function (r) { return r.ok ? r.json() : null; })
        .then(function (data) {
          if (data && data.user) {
            self.jwt = saved;
            self.currentUser = data.user;
            self._updateAuthButton();
          } else {
            try { localStorage.removeItem(self._jwtKey); } catch (_) {}
          }
          self._connect();
        })
        .catch(function () { self._connect(); });
    },

    _addSysMsg: function (text, isError) {
      if (!text) return;
      var el = cs('sys-msg');
      el.textContent = text;
      if (isError) el.classList.add('cs-err');
      this.msgs.appendChild(el);
      if (this.nearBottom) this._scrollBottom();
    },

    // ── State helpers ─────────────────────────────────────────────────────────

    _setConnected: function (connected) {
      if (connected) {
        this.statusBar.classList.remove('cs-vis');
        this.wave.classList.remove('cs-off');
      } else {
        this.statusBar.classList.add('cs-vis');
        this.wave.classList.add('cs-off');
      }
    },

    _setOnline: function (n) {
      this.onlineCount = n;
      this.onlineSpan.textContent = n + this.wt('online');
    },

    _onScroll: function () {
      var el = this.msgs;
      this.nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight <= 200;
      if (this.nearBottom) this._hideNewMsgBtn();
    },

    _scrollBottom: function () {
      var el = this.msgs;
      requestAnimationFrame(function () { el.scrollTop = el.scrollHeight; });
    },

    _showNewMsgBtn: function () {
      this.pendingNew++;
      this.newMsgBtn.textContent = this.wt('new_msgs')(this.pendingNew);
      this.newMsgBtn.classList.add('cs-vis');
    },

    _hideNewMsgBtn: function () {
      this.pendingNew = 0;
      this.newMsgBtn.classList.remove('cs-vis');
    },

    // ── Actions ───────────────────────────────────────────────────────────────

    toggle: function () {
      this.isOpen = !this.isOpen;
      this.panel.classList.toggle('cs-open', this.isOpen);
      this.btn.classList.toggle('cs-open', this.isOpen);
      this.btn.setAttribute('aria-expanded', this.isOpen ? 'true' : 'false');
      if (this.isOpen) {
        this._clearUnread();
        this._scrollBottom();
        var el = this.inputEl;
        setTimeout(function () { el.focus(); }, 320);
      }
    },

    _send: function () {
      var content = this.inputEl.value.trim();
      if (!content || !this.ws || this.ws.readyState !== 1) return;
      if (content.length > this.maxLen) {
        this._addSysMsg('消息超过 ' + this.maxLen + ' 字符限制', true);
        return;
      }
      this.ws.send(JSON.stringify({ type: 'send', content: content }));
      this.inputEl.value = '';
      this._onInputChange();
      this.inputEl.focus();
    },

    _report: function (msgId) {
      if (!msgId || !this.ws || this.ws.readyState !== 1) return;
      this.ws.send(JSON.stringify({ type: 'report', message_id: msgId, reason: 'inappropriate' }));
      this._addSysMsg('举报已提交，感谢反馈。');
    },

    _onInputChange: function () {
      var len = this.inputEl.value.length;
      this.charCountEl.textContent = len > 0 ? len + ' / ' + this.maxLen : '';
      var over = len > this.maxLen;
      this.charCountEl.classList.toggle('cs-warn', over);
      this.inputEl.classList.toggle('cs-over', over);
      // Auto-resize: 1–4 lines
      this.inputEl.style.height = 'auto';
      var newH = Math.min(88, Math.max(44, this.inputEl.scrollHeight));
      this.inputEl.style.height = newH + 'px';
    },

    _toggleEmoji: function () {
      this.emojiOpen = !this.emojiOpen;
      this.emojiPicker.classList.toggle('cs-open', this.emojiOpen);
    },

    _insertEmoji: function (e) {
      var start = this.inputEl.selectionStart;
      var val   = this.inputEl.value;
      this.inputEl.value = val.slice(0, start) + e + val.slice(start);
      this.inputEl.focus();
      this._onInputChange();
    },

    // ── Unread ────────────────────────────────────────────────────────────────

    _incUnread: function (msg) {
      this.unread++;
      this.badge.style.display = 'flex';
      this.badge.textContent = this.unread > 9 ? '9+' : this.unread;
      if (msg && this.btnPreviewEl) {
        var sender = msg.display_name || '';
        var text = (msg.content || '').replace(/<[^>]*>/g, '').substring(0, 40);
        this.btnPreviewEl.textContent = sender ? sender + ': ' + text : text;
      }
      this._playNotifSound();
    },

    _playNotifSound: function () {
      try {
        var AudioCtx = window.AudioContext || window.webkitAudioContext;
        if (!AudioCtx) return;
        var ctx = new AudioCtx();
        var osc = ctx.createOscillator();
        var gain = ctx.createGain();
        osc.connect(gain);
        gain.connect(ctx.destination);
        osc.type = 'sine';
        osc.frequency.setValueAtTime(880, ctx.currentTime);
        osc.frequency.exponentialRampToValueAtTime(440, ctx.currentTime + 0.08);
        gain.gain.setValueAtTime(0.12, ctx.currentTime);
        gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.18);
        osc.start(ctx.currentTime);
        osc.stop(ctx.currentTime + 0.18);
        osc.onended = function () { ctx.close(); };
      } catch (_) {}
    },

    _clearUnread: function () {
      this.unread = 0;
      this.badge.style.display = 'none';
      if (this.btnPreviewEl) this.btnPreviewEl.textContent = '';
      this._hideNewMsgBtn();
    },

    _updateDMLauncherBadge: function () {
      var total = 0;
      var convs = this.dmConvs;
      Object.keys(convs).forEach(function (k) { total += convs[k].unread; });
      if (this.dmLauncherBadge) {
        this.dmLauncherBadge.classList.toggle('cs-vis', total > 0);
      }
    },

    // ── DM ────────────────────────────────────────────────────────────────────

    _buildDMPanel: function () {
      var self = this;
      var panel = this.dmPanelEl = cs('dm-panel');

      // Header
      var hdr = cs('dm-hdr');
      var backBtn = this.dmBackBtn = h('button', { 'data-cs': 'hbtn', 'title': self.wt('back'), 'aria-label': self.wt('back'), 'style': 'display:none' });
      backBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/></svg>';
      backBtn.addEventListener('click', function () { self._dmBackToList(); });
      var titleEl = this.dmHdrTitle = h('span', { 'data-cs': 'dm-hdr-title', 'text': self.wt('dm_title') });
      var closeBtn = h('button', { 'data-cs': 'hbtn', 'title': self.wt('close'), 'aria-label': self.wt('close') });
      closeBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>';
      closeBtn.addEventListener('click', function () { self._toggleDMPanel(); });
      hdr.appendChild(backBtn);
      hdr.appendChild(titleEl);
      hdr.appendChild(closeBtn);
      panel.appendChild(hdr);

      // List view
      var listEl = this.dmListEl = cs('dm-list');
      panel.appendChild(listEl);

      // Empty state
      var emptyEl = this.dmEmptyEl = cs('dm-empty');
      emptyEl.innerHTML = '<svg viewBox="0 0 24 24" style="width:40px;height:40px;opacity:.3;fill:currentColor"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2z"/></svg><div>' + self.wt('dm_empty') + '</div><div style="font-size:11px;margin-top:4px">' + self.wt('dm_empty_hint') + '</div>';
      emptyEl.classList.add('cs-hidden');
      panel.appendChild(emptyEl);

      // Conversation messages view
      var convEl = this.dmConvEl = cs('dm-conv');
      convEl.classList.add('cs-hidden');
      panel.appendChild(convEl);

      // DM input footer
      var footer = this.dmFooterEl = cs('dm-footer');
      footer.classList.add('cs-hidden');
      var inp = this.dmInputEl = h('textarea', { 'data-cs': 'dm-input', 'placeholder': self.wt('dm_placeholder'), 'rows': '1', 'aria-label': self.wt('dm_title') });
      inp.addEventListener('input', function () { self._dmInputResize(); });
      inp.addEventListener('keydown', function (e) {
        if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); self._sendDM(); }
      });
      var sendBtn = h('button', { 'data-cs': 'dm-send', 'title': self.wt('dm_send'), 'aria-label': self.wt('dm_send') });
      sendBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>';
      sendBtn.addEventListener('click', function () { self._sendDM(); });
      footer.appendChild(inp);
      footer.appendChild(sendBtn);
      panel.appendChild(footer);

      return panel;
    },

    _toggleDMPanel: function () {
      this.dmOpen = !this.dmOpen;
      this.dmPanelEl.classList.toggle('cs-open', this.dmOpen);
      if (this.dmOpen) {
        if (!this.dmConvUserId) this._dmShowList();
      }
    },

    _dmShowList: function () {
      this.dmConvUserId = null;
      this.dmHdrTitle.textContent = this.wt('dm_title');
      this.dmBackBtn.style.display = 'none';
      this.dmConvEl.classList.add('cs-hidden');
      this.dmFooterEl.classList.add('cs-hidden');
      this.dmListEl.classList.remove('cs-hidden');
      this._dmRenderList();
    },

    _dmBackToList: function () {
      this._dmShowList();
    },

    _openDMConv: function (userId, username, avatarColor) {
      if (!this.myUserID) { this._showAuthOverlay(); return; }
      userId = parseInt(userId) || userId;
      if (!this.dmConvs[userId]) {
        this.dmConvs[userId] = { username: username || '用户', avatarColor: avatarColor || hashColor(String(userId)), msgs: [], unread: 0 };
      }
      this.dmConvUserId = userId;
      this.dmConvs[userId].unread = 0;
      this._dmUpdateUnreadBadge();

      if (!this.dmOpen) {
        this.dmOpen = true;
        this.dmPanelEl.classList.add('cs-open');
      }

      this.dmHdrTitle.textContent = this.dmConvs[userId].username;
      this.dmBackBtn.style.display = '';
      this.dmListEl.classList.add('cs-hidden');
      this.dmEmptyEl.classList.add('cs-hidden');
      this.dmConvEl.classList.remove('cs-hidden');
      this.dmFooterEl.classList.remove('cs-hidden');
      this._dmRenderConv();

      if (this.dmConvs[userId].msgs.length === 0 && this.ws && this.ws.readyState === 1) {
        this.ws.send(JSON.stringify({ type: 'dm_history', with_user_id: userId }));
      }
      setTimeout(function () { }, 0);
    },

    _dmRenderList: function () {
      var self = this;
      this.dmListEl.innerHTML = '';
      var userIds = Object.keys(this.dmConvs);
      if (userIds.length === 0) {
        this.dmListEl.classList.add('cs-hidden');
        this.dmEmptyEl.classList.remove('cs-hidden');
        return;
      }
      this.dmEmptyEl.classList.add('cs-hidden');
      this.dmListEl.classList.remove('cs-hidden');
      userIds.forEach(function (uid) {
        var conv = self.dmConvs[uid];
        var item = cs('dm-item');
        var av = cs('dm-item-av', { style: 'background:' + (conv.avatarColor || hashColor(String(uid))) });
        av.textContent = initials(conv.username);
        var info = cs('dm-item-info');
        var nameEl = cs('dm-item-name'); nameEl.textContent = conv.username;
        var preview = cs('dm-item-preview');
        var last = conv.msgs[conv.msgs.length - 1];
        preview.textContent = last ? (last.from_user_id == self.myUserID ? '我: ' : '') + last.content : '点击开始对话';
        info.appendChild(nameEl); info.appendChild(preview);
        item.appendChild(av); item.appendChild(info);
        if (conv.unread > 0) {
          var badge = cs('dm-item-badge');
          badge.textContent = conv.unread > 9 ? '9+' : conv.unread;
          item.appendChild(badge);
        }
        item.addEventListener('click', function () { self._openDMConv(parseInt(uid), conv.username, conv.avatarColor); });
        self.dmListEl.appendChild(item);
      });
    },

    _dmRenderConv: function () {
      var self = this;
      this.dmConvEl.innerHTML = '';
      var uid = this.dmConvUserId;
      if (!uid || !this.dmConvs[uid]) return;
      this.dmConvs[uid].msgs.forEach(function (dm) { self._dmRenderBubble(dm); });
      var el = this.dmConvEl;
      requestAnimationFrame(function () { el.scrollTop = el.scrollHeight; });
    },

    _dmRenderBubble: function (dm) {
      var mine = dm.from_user_id == this.myUserID;
      var row = cs('dm-bbl-row');
      if (mine) row.setAttribute('data-mine', '');
      var bbl = cs('dm-bbl');
      bbl.innerHTML = renderContent(dm.content);
      var timeEl = cs('dm-bbl-time');
      timeEl.textContent = timeStr(dm.created_at || new Date().toISOString());
      row.appendChild(bbl); row.appendChild(timeEl);
      this.dmConvEl.appendChild(row);
    },

    _onDM: function (dm) {
      var partnerId = (dm.from_user_id == this.myUserID) ? dm.to_user_id : dm.from_user_id;
      var partnerName = (dm.from_user_id == this.myUserID) ? (dm.to_username || '用户') : (dm.from_username || '用户');
      var partnerColor = (dm.from_user_id == this.myUserID) ? (dm.to_avatar_color || hashColor(String(partnerId))) : (dm.from_avatar_color || hashColor(String(partnerId)));
      if (!this.dmConvs[partnerId]) {
        this.dmConvs[partnerId] = { username: partnerName, avatarColor: partnerColor, msgs: [], unread: 0 };
      }
      this.dmConvs[partnerId].msgs.push(dm);

      if (this.dmOpen && this.dmConvUserId == partnerId) {
        this._dmRenderBubble(dm);
        var el = this.dmConvEl;
        requestAnimationFrame(function () { el.scrollTop = el.scrollHeight; });
      } else if (dm.from_user_id != this.myUserID) {
        this.dmConvs[partnerId].unread++;
        this._dmUpdateUnreadBadge();
        if (!this.isOpen) {
          this._incUnread(); // also calls _playNotifSound
        } else {
          this._playNotifSound();
        }
      }

      if (this.dmOpen && !this.dmConvUserId) this._dmRenderList();
    },

    _onDMHistory: function (withUserId, dms) {
      withUserId = parseInt(withUserId) || withUserId;
      if (!this.dmConvs[withUserId]) return;
      this.dmConvs[withUserId].msgs = dms;
      if (this.dmOpen && this.dmConvUserId == withUserId) this._dmRenderConv();
    },

    _sendDM: function () {
      var content = this.dmInputEl.value.trim();
      if (!content || !this.ws || this.ws.readyState !== 1 || !this.dmConvUserId) return;
      this.ws.send(JSON.stringify({ type: 'dm', to_user_id: this.dmConvUserId, content: content }));
      this.dmInputEl.value = '';
      this._dmInputResize();
      this.dmInputEl.focus();
    },

    _dmInputResize: function () {
      this.dmInputEl.style.height = 'auto';
      this.dmInputEl.style.height = Math.min(88, Math.max(44, this.dmInputEl.scrollHeight)) + 'px';
    },

    _dmUpdateUnreadBadge: function () {
      var total = 0;
      var convs = this.dmConvs;
      Object.keys(convs).forEach(function (k) { total += convs[k].unread; });
      if (total > 0) {
        this.dmUnreadBadge.textContent = total > 9 ? '9+' : total;
        this.dmUnreadBadge.classList.add('cs-vis');
      } else {
        this.dmUnreadBadge.classList.remove('cs-vis');
      }
      this._updateDMLauncherBadge();
    },

    // ── Public API ────────────────────────────────────────────────────────────

    setTitle: function (t) {
      this.title = t;
      this.titleEl.textContent = t;
      if (this.btnTitleEl) this.btnTitleEl.textContent = t;
    },

    setNotice: function (t) {
      this.notice = t;
      var el = this.noticeEl.querySelector('[data-cs="notice-text"]');
      if (el) el.textContent = t;
      this.noticeEl.classList.toggle('cs-hidden', !t);
    },

    // ── Bot cards ─────────────────────────────────────────────────────────────

    _renderBotCards: function (botName, avatarColor, cards) {
      var wrap = h('div', { 'data-cs': 'bot-card' });

      var hdr = h('div', { 'data-cs': 'bot-card-hdr' });
      var av = h('div', { 'data-cs': 'bot-card-av' });
      av.style.background = avatarColor || '#6366f1';
      av.textContent = initials(botName);
      var nameEl = h('div', { 'data-cs': 'bot-card-name', 'text': botName });
      hdr.appendChild(av);
      hdr.appendChild(nameEl);
      wrap.appendChild(hdr);

      var cardsWrap = h('div', { 'data-cs': 'bot-cards-wrap' });
      cards.forEach(function (card) {
        var item = h('div', { 'data-cs': 'bot-card-item' });
        if (card.link) {
          item.addEventListener('click', function () { window.open(card.link, '_blank'); });
        }
        if (card.poster) {
          var img = h('img', { 'data-cs': 'bot-card-img', 'src': card.poster, 'alt': card.title || '' });
          item.appendChild(img);
        }
        var body = h('div', { 'data-cs': 'bot-card-body' });
        var titleEl = h('div', { 'data-cs': 'bot-card-title', 'text': card.title || '' });
        var descEl = h('div', { 'data-cs': 'bot-card-desc', 'text': card.desc || '' });
        body.appendChild(titleEl);
        body.appendChild(descEl);
        item.appendChild(body);
        cardsWrap.appendChild(item);
      });
      wrap.appendChild(cardsWrap);
      this.msgs.appendChild(wrap);
    },

    // ── Channel switcher ──────────────────────────────────────────────────────

    _buildChPanel: function () {
      var self = this;
      var panel = this.chPanelEl = h('div', { 'data-cs': 'ch-panel' });

      var hdr = h('div', { 'data-cs': 'ch-hdr' });
      var titleEl = h('span', { 'data-cs': 'ch-hdr-title', 'text': '切换频道' });
      var closeBtn = h('button', { 'data-cs': 'hbtn', 'title': '关闭', 'aria-label': '关闭' });
      closeBtn.innerHTML = '<svg viewBox="0 0 24 24"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>';
      closeBtn.addEventListener('click', function () { self._toggleChPanel(); });
      hdr.appendChild(titleEl);
      hdr.appendChild(closeBtn);
      panel.appendChild(hdr);

      this.chListEl = h('div', { 'data-cs': 'ch-list' });
      panel.appendChild(this.chListEl);
      return panel;
    },

    _toggleChPanel: function () {
      this.chPanelOpen = !this.chPanelOpen;
      this.chPanelEl.classList.toggle('cs-open', this.chPanelOpen);
      if (this.chPanelOpen) this._renderChList();
    },

    _renderChList: function () {
      var self = this;
      this.chListEl.innerHTML = '';
      this.channels.forEach(function (ch) {
        var item = h('div', { 'data-cs': 'ch-item' });
        if (ch.slug === self.channel) item.classList.add('cs-active');
        var color = ch.accent_color || '#6366f1';
        var av = h('div', { 'data-cs': 'ch-av', 'style': 'background:' + color });
        av.textContent = initials(ch.name);
        var info = h('div', { 'data-cs': 'ch-info' });
        var nameEl = h('div', { 'data-cs': 'ch-name', 'text': ch.name });
        info.appendChild(nameEl);
        if (ch.description) {
          var desc = h('div', { 'data-cs': 'ch-desc', 'text': ch.description });
          info.appendChild(desc);
        }
        item.appendChild(av);
        item.appendChild(info);
        item.addEventListener('click', function () { self._switchChannel(ch); });
        self.chListEl.appendChild(item);
      });
    },

    _switchChannel: function (ch) {
      if (ch.slug === this.channel) { this._toggleChPanel(); return; }
      this.channel = ch.slug;
      this.msgs.innerHTML = '';
      this.myName = null;
      if (this.titleEl) this.titleEl.textContent = ch.name || this.title || (this.lang === 'en' ? 'Chat' : '聊天');
      if (this.btnTitleEl) this.btnTitleEl.textContent = ch.name || this.title || (this.lang === 'en' ? 'Chat' : '聊天');
      if (ch.accent_color && /^#[0-9a-fA-F]{6}$/.test(ch.accent_color)) {
        var hex = ch.accent_color;
        var r = parseInt(hex.slice(1,3),16), g = parseInt(hex.slice(3,5),16), b2 = parseInt(hex.slice(5,7),16);
        this.root.style.setProperty('--accent', hex);
        this.root.style.setProperty('--accent-dim', 'rgba('+r+','+g+','+b2+',.15)');
        this.root.style.setProperty('--accent-glow', 'rgba('+r+','+g+','+b2+',.35)');
      }
      this._toggleChPanel();
      this._connect();
    },

    _loadChannels: function () {
      var self = this;
      fetch(this.server + '/api/channels')
        .then(function (r) { return r.ok ? r.json() : null; })
        .then(function (data) {
          if (!data || !data.channels) return;
          var active = (data.channels || []).filter(function (c) { return c.is_active; });
          self.channels = active;
          if (self.chBtnEl) self.chBtnEl.style.display = active.length > 1 ? '' : 'none';
        })
        .catch(function () {});
    },
  };

  // ── Global API ───────────────────────────────────────────────────────────────

  global.ChatSphereWidget = Widget;
  global.ChatSphere = {
    init: function (opts) {
      var instance = new Widget(opts);
      return instance;
    },
  };

  // ── Auto-init from script tag data-* attributes ──────────────────────────────

  var currentScript = document.currentScript ||
    document.querySelector('script[data-server],script[data-channel]');
  if (currentScript && (currentScript.dataset.server || currentScript.dataset.channel)) {
    var d = currentScript.dataset;
    var run = function () {
      global.ChatSphere = new Widget({
        server:   d.server   || location.origin,
        channel:  d.channel  || 'general',
        position: d.position || 'bottom-right',
        title:    d.title    || '',
        notice:   d.notice   || '',
        telegram: d.telegram || '',
        accent:   d.accent   || '',
        offset:   d.offset   || '',
        lang:     d.lang     || '',
      });
    };
    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', run);
    } else {
      run();
    }
  }

}(window));
