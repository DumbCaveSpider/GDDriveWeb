<script setup lang="ts">
import { ref, onMounted } from 'vue'
import LoginPage from './LoginPage.vue'
import Dashboard from './Dashboard.vue'
import { API, getCookie, credHeaders } from '../utils/gd'

// --- State ---
const view = ref<'login' | 'dashboard'>('login')
const loggedInAs = ref('')
const loggedInAccountID = ref('')
const toast = ref<{ msg: string; type: 'success' | 'error' | 'info' } | null>(null)

function showToast(msg: string, type: 'success' | 'error' | 'info' = 'info') {
  toast.value = { msg, type }
  setTimeout(() => (toast.value = null), 3500)
}

function handleLoggedIn(username: string, accountId: string) {
  loggedInAs.value = username
  loggedInAccountID.value = accountId
  view.value = 'dashboard'
}

function handleLoggedOut() {
  loggedInAs.value = ''
  view.value = 'login'
}

onMounted(async () => {
  const savedUser = getCookie('gdd_username')
  const savedGJP2  = getCookie('gdd_gjp2')
  if (savedUser && savedGJP2) {
    loggedInAs.value = savedUser
    loggedInAccountID.value = getCookie('gdd_account_id') ?? ''
    view.value = 'dashboard'
  } else {
    // Verify Go server is reachable
    const res = await fetch(`${API}/status`, { headers: credHeaders() }).catch(() => null)
    if (!res) showToast('Cannot reach the Go server. Start it with: go run . in server/', 'error')
  }
})
</script>

<template>
  <div class="gd-root">
    <!-- Static BG -->
    <div class="main-bg"></div>

    <!-- Toast -->
    <Transition name="toast-slide">
      <div v-if="toast" :class="['toast', toast.type]">
        <img v-if="toast.type === 'success'" class="toast-icon" src="../assets/check.png" alt="✓" />
        <img v-else-if="toast.type === 'error'" class="toast-icon" src="../assets/cross.png" alt="✕" />
        <img v-else-if="toast.type === 'info'" class="toast-icon" src="../assets/info.png" alt="ℹ" />
        {{ toast.msg }}
      </div>
    </Transition>

    <div class="view-container">
      <Transition name="fade-scale" mode="out-in">
        <LoginPage v-if="view === 'login'" @loggedIn="handleLoggedIn" @showToast="showToast" />
        <Dashboard v-else :username="loggedInAs" :accountId="loggedInAccountID" @loggedOut="handleLoggedOut" @showToast="showToast" />
      </Transition>
    </div>
  </div>
</template>

<style>
/* Global-ish styles for the app. Not scoped so they hit sub-components. */
@import url('https://fonts.googleapis.com/css2?family=Chakra+Petch:wght@400;700&display=swap');

@font-face {
  font-family: 'Pusab';
  src: url('../assets/Pusab.ttf') format('truetype');
  font-weight: normal;
  font-style: normal;
}

.gd-root {
  font-family: 'Pusab', 'Chakra Petch', sans-serif;
  min-height: 100vh;
  position: relative;
  overflow-x: hidden;
  color: #fff;
}

/* Base Pusab font for titles, buttons, tabs */
.brand, .btn-primary, .btn-ghost, .tab, h1, h2, .verify-code, .stat-num, .field-group label, .btn-secondary {
  font-family: 'Pusab', 'Chakra Petch', sans-serif;
  letter-spacing: 0.5px;
  text-shadow: 2px 2px 0 #000, -1px -1px 0 #000, 1px -1px 0 #000, -1px 1px 0 #000, 1px 1px 0 #000;
}

/* Helvetica for smaller/helper text */
.brand-sub, .login-hint, .stat-label, .file-meta, .drop-sub, .panel-desc, .small, .verify-instructions p {
  font-family: 'Helvetica', Arial, sans-serif;
  font-weight: 400;
  color: #ffffff;
}

.brand-sub {
  padding: 10px;
}

.main-bg {
  position: fixed;
  inset: 0;
  z-index: 0;
  background: linear-gradient(180deg, rgb(0, 75, 100) 0%, rgb(0, 40, 60) 100%);
  pointer-events: none;
}


.toast {
  position: fixed; top: 24px; right: 24px; z-index: 9999;
  padding: 16px 24px;
  display: flex; align-items: center; gap: 12px;
  position: fixed;
  font-family: 'Pusab', 'Chakra Petch', sans-serif;
  font-size: 20px;
  color: #fff;
  text-shadow: 2px 2px 0 #000, -1px -1px 0 #000, 1px -1px 0 #000, -1px 1px 0 #000, 1px 1px 0 #000;
  max-width: 440px;
}

.toast::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  border: 10px solid transparent;
  border-image-source: url('../assets/toast1BG.png');
  border-image-slice: 40% fill;
  border-image-width: 25px;
  z-index: -1;
}

.toast-icon { width: 28px; height: 28px; object-fit: contain; flex-shrink: 0; }

.toast-slide-enter-active, .toast-slide-leave-active { transition: all 0.3s ease; }
.toast-slide-enter-from { opacity: 0; transform: translateX(40px); }
.toast-slide-leave-to   { opacity: 0; transform: translateX(40px); }

.fade-scale-enter-active, .fade-scale-leave-active { transition: all 0.35s cubic-bezier(0.34, 1.56, 0.64, 1); }
.fade-scale-enter-from   { opacity: 0; transform: scale(0.96); }
.fade-scale-leave-to     { opacity: 0; transform: scale(1.02); }

.view-container { position: relative; z-index: 1; }

/* SHARED UI ELEMENTS */
.login-wrapper { min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 24px; }
.login-card, .panel {
  position: relative;
  border: 12px solid transparent;
  border-image-source: url('../assets/containerBG.png');
  border-image-slice: 33% fill;
  border-image-width: 50px;
  background: transparent;
  width: 100%;
  max-width: 720px;
  box-shadow: 0 24px 80px rgba(0,0,0,0.5);
  z-index: 1;
  padding: 10px;
}

.panel-content {
  position: relative;
  z-index: 1;
}

.panel-content::after {
  content: '';
  position: absolute;
  top: -10px; left: -10px; right: -10px; bottom: -10px;
  border-image-source: url('../assets/square1BG.png');
  border-image-slice: 33% fill;
  border-image-width: 50px;
  opacity: 0.5;
  z-index: -1;
  pointer-events: none;
}

.panel-content > * { position: relative; z-index: 2; }

.login-card {
  padding: 30px;
}

.panel {
  max-width: 100%;
  padding: 20px;
}

.btn-row { display: flex; flex-direction: column; gap: 12px; margin-top: 24px; }
.logo-block { text-align: center; margin-bottom: 24px; display: flex; flex-direction: column; align-items: center; gap: 8px; }
.brand-logo { width: 100%; max-width: 320px; height: auto; object-fit: contain; }

.field-group { margin-bottom: 18px; }
.field-group label { display: block; font-size: 20px; font-weight: 500; margin-bottom: 7px; }
.input-wrap { position: relative; }
.input-icon { position: absolute; left: 14px; top: 50%; transform: translateY(-50%); font-size: 14px; color: #64748b; }
.input-wrap input, .styled-select {
  width: 100%; padding: 13px 14px 13px 40px; background: rgba(255,255,255,0.06); border: 1px solid rgba(255,255,255,0.1);
  border-radius: 12px; color: #e2e8f0; font-size: 15px; outline: none; transition: transform 0.1s;
}
.styled-select { padding-left: 14px; cursor: pointer; }
.input-wrap input:focus, .styled-select:focus { border-color: #7c3aed; box-shadow: 0 0 0 3px rgba(124, 58, 237, 0.2); }

.btn-primary, .btn-ghost, .tab, .btn-secondary {
  border: 10px solid transparent;
  border-image-source: url('../assets/button1BG.png');
  border-image-slice: 40% fill;
  border-image-width: 50px;
  background: transparent;
  color: #fff;
  font-size: 25px;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.1s;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 8px 16px;
}
.btn-primary span, .btn-secondary span, .btn-ghost span { display: flex; align-items: center; gap: 10px; }

.btn-secondary {
  border-image-source: url('../assets/button2BG.png');
}

.btn-primary:hover:not(:disabled), .btn-ghost:hover:not(:disabled), .btn-secondary:hover:not(:disabled) { 
  transform: scale(1.05); 
}

.btn-primary:active, .btn-ghost:active, .btn-secondary:active {
  transform: scale(0.95);
}

.panel-icon {
  width: 32px;
  height: 32px;
  object-fit: contain;
  align-self: center;
  flex-shrink: 0;
}

.verify-instructions, .panel-desc {
  position: relative;
  padding: 24px;
  margin-bottom: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  z-index: 1;
}

.verify-instructions::before, .panel-desc::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  border: 12px solid transparent; /* default for nine-slice */
  border-image-source: url('../assets/square1BG.png');
  border-image-slice: 33% fill;
  border-image-width: 50px;
  opacity: 0.5;
  z-index: -1;
  pointer-events: none;
}
.verify-code { font-family: monospace; font-size: 20px; font-weight: 700; color: #c4b5fd; background: rgba(0,0,0,0.3); padding: 8px; border-radius: 6px; }

.dashboard { max-width: 900px; margin: 0 auto; padding: 0 16px 60px; }
.dash-header { display: flex; align-items: center; justify-content: space-between; padding: 24px 0; border-bottom: 1px solid rgba(255,255,255,0.07); }
.dash-logo { height: 40px; width: auto; object-fit: contain; }
.dash-user { display: flex; align-items: center; gap: 8px; }
.user-pill {
  position: relative;
  display: flex;
  align-items: center;
  padding: 6px 14px;
  background: transparent;
  z-index: 1;
}
.user-pill::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  border: 4px solid transparent;
  border-image-source: url('../assets/square1BG.png');
  border-image-slice: 33% fill;
  border-image-width: 20px;
  opacity: 0.5;
  z-index: -1;
  pointer-events: none;
}
.user-avatar-img { width: 32px; height: 32px; border-radius: 8px; object-fit: contain; }

.stats-bar { display: flex; align-items: center; gap: 24px; padding: 20px 0; }
.stat-item { display: flex; flex-direction: column; }
.stat-num { font-size: 22px; font-weight: 700; color: #a855f7; }
.stat-label { font-size: 12px; color: #64748b; }
.stat-sep { width: 1px; height: 32px; background: rgba(255,255,255,0.07); }

.tabs { display: flex; gap: 8px; background: transparent; padding: 0; margin-bottom: 24px; border: none; }
.tab { flex: 1; min-height: 48px; display: flex; align-items: center; justify-content: center; gap: 8px; }
.tab.active { transform: scale(1.05); filter: brightness(1.2); }
.tab-icon { width: 32px; height: 32px; object-fit: contain; flex-shrink: 0; }

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
  padding-bottom: 16px;
  border-bottom: 2px solid rgba(255,255,255,0.07);
}
.header-title-group { display: flex; flex-direction: row; align-items: baseline; gap: 12px; }
.header-title-group .stat-num { font-size: 14px; color: #94a3b8; font-weight: 400; }
.panel-header h1, .panel-header h2 { font-size: 24px; margin: 0; }

.file-list { list-style: none; display: flex; flex-direction: column; gap: 10px; }
.file-item { display: flex; align-items: center; gap: 14px; background: rgba(255,255,255,0.04); border: 1px solid rgba(255,255,255,0.07); border-radius: 12px; padding: 14px 16px; }
.file-icon-img { width: 32px; height: 32px; object-fit: contain; flex-shrink: 0; }
.file-info { flex: 1; min-width: 0; }
.file-name { display: block; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.file-meta { font-size: 12px }
.file-actions { display: flex; gap: 6px; }

.drop-zone { border: 2px dashed rgba(255,255,255,0.12); border-radius: 16px; padding: 44px; text-align: center; cursor: pointer; transition: all 0.2s; margin-bottom: 20px; }
.drop-zone:hover { border-color: #7c3aed; background: rgba(124,58,237,0.06); }
.drop-inner { display: flex; flex-direction: column; align-items: center; gap: 8px; }
.drop-icon { font-size: 40px; }

.btn-sm {
  display: flex;
  align-items: center;
  justify-content: center;
  border: 6px solid transparent;
  border-image-source: url('../assets/button1BG.png');
  border-image-slice: 40% fill;
  border-image-width: 20px;
  background: transparent;
  color: #fff;
  cursor: pointer;
  font-family: 'Pusab', 'Chakra Petch', sans-serif;
  font-size: 14px;
  padding: 4px 10px;
  transition: transform 0.1s;
  min-width: 36px;
  text-align: center;
}
.btn-sm.secondary { border-image-source: url('../assets/button2BG.png'); }
.btn-sm:hover { transform: scale(1.1); }
.btn-sm:active { transform: scale(0.92); }
.btn-icon { height: 32px; width: 24px; object-fit: contain; flex-shrink: 0; }
.btn-danger { padding: 8px 18px; border-radius: 10px; background: rgba(239,68,68,0.12); border: 1px solid rgba(239,68,68,0.25); color: #f87171; cursor: pointer; }

.spinner { width: 18px; height: 18px; border: 2px solid rgba(255,255,255,0.3); border-top-color: #fff; border-radius: 50%; animation: spin 0.7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.login-hint { text-align: center; font-size: 12px; margin-top: 20px; line-height: 1.6; }
.or-divider { display: flex; align-items: center; gap: 12px; margin: 16px 0; font-size: 13px; }
.or-divider::before, .or-divider::after { content: ''; flex: 1; height: 1px; background: rgba(255,255,255,0.07); }

.app-footer {
  text-align: center;
  padding: 10px;
  font-size: 20px;
  text-shadow: 2px 2px 0 #000, -1px -1px 0 #000, 1px -1px 0 #000, -1px 1px 0 #000, 1px 1px 0 #000;
}
.app-footer a {
  color: #c17fff;
  text-decoration: none;
  font-weight: 600;
  transition: color 0.2s;
}
.app-footer a:hover {
  color: #c084fc;
  text-decoration: underline;
}
.video-embed {
  width: 100%;
  max-width: 720px;
  margin: 15px auto 0;
  aspect-ratio: 16 / 9;
}
.video-embed iframe {
  width: 100%;
  height: 100%;
  border-radius: 12px;
  border: 4px solid rgba(0,0,0,0.5);
}

.dash-footer {
  margin-top: 24px;
  padding: 20px;
  text-align: center;
  position: relative;
  z-index: 1;
}
.dash-footer::before {
  content: '';
  position: absolute;
  top: 0; left: 0; right: 0; bottom: 0;
  border-image-source: url('../assets/square1BG.png');
  border-image-slice: 33% fill;
  border-image-width: 30px;
  opacity: 0.5;
  z-index: -1;
  pointer-events: none;
}
.dash-hint { font-family: 'Pusab', 'Chakra Petch', sans-serif; font-size: 20px; margin: 10px; }
.dash-hint a { color: #7fc1ff; text-decoration: none; font-weight: 700; transition: color 0.1s; }
.dash-hint a:hover { color: #7fc1ff; text-decoration: underline; }

@media (max-width: 600px) {
  .login-card { padding: 32px 24px; }
  .stats-bar { flex-wrap: wrap; }
  .tabs { flex-wrap: wrap; }
}
</style>
