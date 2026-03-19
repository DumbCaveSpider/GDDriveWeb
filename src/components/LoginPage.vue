<script setup lang="ts">
import { ref } from 'vue'
import { API, setCookie } from '../utils/gd'

const emit = defineEmits<{
  (e: 'loggedIn', username: string, accountId: string): void
  (e: 'showToast', msg: string, type: 'success' | 'error' | 'info'): void
}>()

const loginPhase = ref<'init' | 'verify'>('init')
const requiresVerification = ref(true)
const username = ref('')
const password = ref('')
const accountId = ref('')
const validationCode = ref('')
const loading = ref(false)

async function doLoginInit() {
  if (!accountId.value) {
    emit('showToast', 'Account ID is required to start validation.', 'error'); return
  }
  loading.value = true
  try {
    const res = await fetch(`${API}/login/init`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ account_id: accountId.value }),
    })
    const json = await res.json()
    if (json.success) {
      validationCode.value = json.data.validation_code || ''
      requiresVerification.value = json.data.requires_verification
      loginPhase.value = 'verify'
      if (!requiresVerification.value) {
        emit('showToast', 'Welcome back! Please enter your credentials.', 'info')
      } else {
        emit('showToast', 'Validation code generated!', 'success')
      }
    } else {
      emit('showToast', json.message ?? 'Failed to initialize login.', 'error')
    }
  } catch (err) {
    emit('showToast', 'Cannot reach server', 'error')
  } finally { loading.value = false }
}

async function doLoginValidate() {
  if (!username.value || !password.value || (requiresVerification.value && !validationCode.value)) {
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
        <p class="brand-sub">Store files inside a Geometry Dash level</p>
        <footer class="app-footer">
          <p>Watch this video by <a href="https://youtube.com/@SweepSweep2" target="_blank" rel="noopener noreferrer">@SweepSweep2</a> on how it works!</p>
          <div class="video-embed">
            <iframe width="640" height="360" src="https://www.youtube-nocookie.com/embed/oENqzFJ3TgI?si=NPOjFARPw3WitqJB" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" referrerpolicy="strict-origin-when-cross-origin" allowfullscreen></iframe>
          </div>
        </footer>
      </div>

      <!-- init the web -->
      <form v-if="loginPhase === 'init'" @submit.prevent="doLoginInit" id="login-form-init">
        <div class="field-group">
          <label for="gd-account-id">Geometry Dash Account ID</label>
          <div class="input-wrap">
            <input id="gd-account-id" v-model="accountId" type="text" placeholder="Account ID" required />
          </div>
        </div>
        <div class="btn-row">
          <button id="login-init-btn" type="submit" class="btn-primary" :class="{ loading }" :disabled="loading">
            <span v-if="!loading">Continue</span>
            <span v-else class="spinner"></span>
          </button>
        </div>
      </form>

      <!-- verify user -->
      <form v-else @submit.prevent="doLoginValidate" id="login-form-verify">
        <div v-if="requiresVerification" class="verify-instructions">
          <p>To verify ownership, set your profile's <strong>Custom</strong> field to the following token</p>
          <div class="verify-code">{{ validationCode }}</div>
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
          <button id="login-verify-btn" type="submit" class="btn-primary" :class="{ loading }" :disabled="loading">
            <span v-if="!loading">Validate & Login</span>
            <span v-else class="spinner"></span>
          </button>
          <button type="button" class="btn-secondary" @click="loginPhase = 'init'" :disabled="loading">Back</button>
        </div>
      </form>

      <p class="login-hint">The server will store your GJP2 for authentication purposes. If you wish not to use your GJP2 outside Geometry Dash, do not use this website.</p>
    </div>
  </div>
</template>
