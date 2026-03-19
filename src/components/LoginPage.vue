<script setup lang="ts">
import { ref } from 'vue'
import { API, setCookie } from '../utils/gd'

const emit = defineEmits<{
  (e: 'loggedIn', username: string, accountId: string): void
  (e: 'showToast', msg: string, type: 'success' | 'error' | 'info'): void
}>()

const authPhase = ref<'landing' | 'login' | 'signup'>('landing')
const username = ref('')
const password = ref('')
const accountId = ref('')
const validationCode = ref('')
const loading = ref(false)

function setPhase(p: 'landing' | 'login' | 'signup') {
  authPhase.value = p
  if (p === 'signup' && !validationCode.value) {
    validationCode.value = Math.random().toString(36).substring(2, 10).toUpperCase()
  }
}

function copyCode() {
  if (validationCode.value) {
    navigator.clipboard.writeText(validationCode.value)
      .then(() => emit('showToast', 'Verification code copied to clipboard!', 'info'))
      .catch(() => emit('showToast', 'Failed to copy', 'error'))
  }
}

async function doLogin() {
  if (!username.value || !password.value) {
    emit('showToast', 'Username and password are required.', 'error'); return
  }
  loading.value = true
  try {
    const res = await fetch(`${API}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.value, password: password.value }),
    })
    const json = await res.json()
    if (json.success) {
      setCookie('gdd_username',   json.data.username)
      setCookie('gdd_gjp2',       json.data.gjp2)
      setCookie('gdd_account_id', json.data.account_id)
      emit('loggedIn', json.data.username, json.data.account_id)
      emit('showToast', json.message, 'success')
    } else {
      emit('showToast', json.message ?? 'Login failed.', 'error')
    }
  } catch (err) {
    emit('showToast', 'Cannot reach server', 'error')
  } finally { loading.value = false }
}


async function doLoginValidate() {
  if (!username.value || !password.value || !validationCode.value) {
    emit('showToast', 'All fields are required.', 'error'); return
  }
  loading.value = true
  try {
    const res = await fetch(`${API}/login/validate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value,
        password: password.value,
        account_id: accountId.value,
        validation_code: validationCode.value
      }),
    })
    const json = await res.json()
    if (json.success) {
      setCookie('gdd_username',   json.data.username)
      setCookie('gdd_gjp2',       json.data.gjp2)
      setCookie('gdd_account_id', json.data.account_id)
      emit('loggedIn', json.data.username, json.data.account_id)
      emit('showToast', json.message, 'success')
    } else {
      emit('showToast', json.message ?? 'Validation failed.', 'error')
    }
  } catch (err) {
    emit('showToast', 'Cannot reach server', 'error')
  } finally { loading.value = false }
}
</script>

<template>
  <div class="login-wrapper">
    <div class="login-card">
      <div class="logo-block">
        <img class="brand-logo" src="../assets/gddrive-logo.png" alt="GD Drive" />
        <p class="brand-sub">Store files inside the Geometry Dash Server</p>
      </div>

      <!-- LANDING -->
      <div v-if="authPhase === 'landing'" class="landing-actions">
        <footer class="app-footer">
          <p>Watch this video by <a href="https://youtube.com/@SweepSweep2" target="_blank" rel="noopener noreferrer">@SweepSweep2</a> on how it works!</p>
          <div class="video-embed">
            <iframe width="640" height="360" src="https://www.youtube-nocookie.com/embed/oENqzFJ3TgI?si=NPOjFARPw3WitqJB" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>
          </div>
        </footer>
        <button class="btn-primary" @click="setPhase('login')">
          <span>Login</span>
        </button>
        <button class="btn-secondary" @click="setPhase('signup')">
          <span>Sign Up</span>
        </button>
      </div>

      <!-- LOGIN -->
      <form v-else-if="authPhase === 'login'" @submit.prevent="doLogin" id="login-form-direct">
        <div class="field-group">
          <label for="login-username">Username</label>
          <div class="input-wrap">
            <input id="login-username" v-model="username" type="text" placeholder="GD Username" required />
          </div>
        </div>
        <div class="field-group">
          <label for="login-password">Password</label>
          <div class="input-wrap">
            <input id="login-password" v-model="password" type="password" placeholder="••••••••" required />
          </div>
        </div>
        <div class="btn-row">
          <button type="submit" class="btn-primary" :class="{ loading }" :disabled="loading">
            <span v-if="!loading">Login</span>
            <span v-else class="spinner"></span>
          </button>
          <button type="button" class="btn-secondary" @click="authPhase = 'landing'">Back</button>
        </div>
      </form>

      <!-- SIGN UP -->
      <form v-else-if="authPhase === 'signup'" @submit.prevent="doLoginValidate" id="login-form-signup">
        <div class="verify-instructions">
          <p>To verify ownership of your account, set your profile's <strong>Custom</strong> field to the following one-time authentication token</p>
          <div class="verify-code" @click="copyCode" style="cursor: pointer;" title="Click to copy">{{ validationCode }}</div>
        </div>

        <div class="field-group">
          <label for="gd-account-id">Geometry Dash Account ID</label>
          <div class="input-wrap">
            <input id="gd-account-id" v-model="accountId" type="text" placeholder="Account ID" required />
          </div>
        </div>
        <div class="field-group">
          <label for="gd-username">Geometry Dash Username</label>
          <div class="input-wrap">
            <input id="gd-username" v-model="username" type="text" placeholder="Confirm Username" required />
          </div>
        </div>
        <div class="field-group">
          <label for="gd-password">Geometry Dash Password</label>
          <div class="input-wrap">
            <input id="gd-password" v-model="password" type="password" placeholder="••••••••" required />
          </div>
        </div>
        
        <div class="btn-row">
          <button id="signup-btn" type="submit" class="btn-primary" :class="{ loading }" :disabled="loading">
            <span v-if="!loading">Validate & Sign Up</span>
            <span v-else class="spinner"></span>
          </button>
          <button type="button" class="btn-secondary" @click="authPhase = 'landing'">Back</button>
        </div>
      </form>

      <p class="login-hint">The server will store your GJP2 for authentication purposes. If you wish not to use your GJP2 outside Geometry Dash, do not use this website.</p>
      <p class="login-hint">Frontend/Backend created by <a href="https://arcticwoof.xyz">@ArcticWoof</a></p>
    </div>
  </div>
</template>
