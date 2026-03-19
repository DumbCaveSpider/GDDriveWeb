<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { API, credHeaders, deleteCookie } from '../utils/gd'

const props = defineProps<{
  username: string
  accountId: string
}>()

const emit = defineEmits<{
  (e: 'loggedOut'): void
  (e: 'showToast', msg: string, type: 'success' | 'error' | 'info'): void
}>()

const files = ref<{ name: string; level_id: number; level_name: string }[]>([])
const activeTab = ref<'view' | 'upload' | 'download' | 'delete'>('view')

const uploadFile = ref<File | null>(null)
const uploadDragging = ref(false)
const downloadingFiles = ref<Set<string>>(new Set())
const loading = ref(false)

async function fetchFiles() {
  try {
    const res = await fetch(`${API}/files`, { headers: credHeaders() })
    const json = await res.json()
    if (json.success) files.value = json.data ?? []
  } catch (err) {
    emit('showToast', 'Failed to fetch files.', 'error')
  }
}

async function doLogout() {
  try {
    await fetch(`${API}/logout`, { method: 'POST' })
  } catch (err) {}
  deleteCookie('gdd_username')
  deleteCookie('gdd_gjp2')
  deleteCookie('gdd_account_id')
  emit('loggedOut')
  emit('showToast', 'Logged out.', 'info')
}

async function doUpload() {
  if (!uploadFile.value) { emit('showToast', 'Please select a file first.', 'error'); return }
  loading.value = true
  try {
    const form = new FormData()
    form.append('file', uploadFile.value)
    const res = await fetch(`${API}/files/upload`, {
      method: 'POST',
      headers: credHeaders(),
      body: form,
    })
    const json = await res.json()
    emit('showToast', json.message ?? (json.success ? 'Uploaded!' : 'Upload failed.'), json.success ? 'success' : 'error')
    if (json.success) { uploadFile.value = null; fetchFiles() }
  } finally { loading.value = false }
}

async function doDownload(name: string) {
  const target = name
  if (!target) return

  if (name) downloadingFiles.value.add(name)
  else loading.value = true

  try {
    const res = await fetch(`${API}/files/download`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', ...credHeaders() },
      body: JSON.stringify({ file_name: target }),
    })
    if (!res.ok) {
      const j = await res.json(); emit('showToast', j.message ?? 'Download failed.', 'error'); return
    }
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = target; a.click()
    URL.revokeObjectURL(url)
    emit('showToast', 'Download started!', 'success')
  } finally {
    if (name) downloadingFiles.value.delete(name)
    else loading.value = false
  }
}

async function doDelete(name: string) {
  if (!confirm(`Delete "${name}"? This is permanent.`)) return
  loading.value = true
  try {
    const res = await fetch(`${API}/files`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json', ...credHeaders() },
      body: JSON.stringify({ file_name: name }),
    })
    const json = await res.json()
    emit('showToast', json.message ?? (json.success ? 'Deleted.' : 'Delete failed.'), json.success ? 'success' : 'error')
    if (json.success) fetchFiles()
  } finally { loading.value = false }
}

function onDrop(e: DragEvent) {
  uploadDragging.value = false
  const f = e.dataTransfer?.files[0]
  if (f) uploadFile.value = f
}

function onFileInput(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) uploadFile.value = f
}

onMounted(fetchFiles)
async function doDeleteAccount() {
  if (!confirm('DELETE ACCOUNT? This will permanently remove all your file indexes from GDDrive. This cannot be undone.')) return
  loading.value = true
  try {
    const res = await fetch(`${API}/account`, {
      method: 'DELETE',
      headers: credHeaders(),
    })
    const json = await res.json()
    if (json.success) {
      emit('showToast', 'Account deleted successfully.', 'success')
      doLogout()
    } else {
      emit('showToast', json.message ?? 'Failed to delete account.', 'error')
    }
  } catch (err) {
    emit('showToast', 'Cannot reach server.', 'error')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="dashboard">
    <header class="dash-header">
      <img class="dash-logo" src="../assets/gddrive-logo.png" alt="GD Drive" />
      <div class="dash-user">
        <div class="user-pill">
          <span>Logged in as: {{ username }}</span>
        </div>
        <button id="delete-acc-btn" class="btn-sm secondary" title="Delete Account" :disabled="loading" @click="doDeleteAccount">
          <img src="../assets/delete-white.png" class="btn-icon" alt="Delete" />
        </button>
        <button id="logout-btn" class="btn-sm secondary" title="Sign Out" @click="doLogout">
          <img src="../assets/logout.png" class="btn-icon" alt="Logout" />
        </button>
      </div>
    </header>

    <nav class="tabs" role="tablist">
      <button :class="['tab', { active: activeTab==='view' }]"     @click="activeTab='view'; fetchFiles()"><img class="tab-icon" src="../assets/folder.png" />Files</button>
      <button :class="['tab', { active: activeTab==='upload' }]"   @click="activeTab='upload'"><img class="tab-icon" src="../assets/upload.png" />Upload</button>
    </nav>

    <div class="tab-content">
      <div v-if="activeTab === 'view'" class="panel">
        <div class="panel-header">
          <div class="header-title-group">
            <h2>Your Files</h2>
            <span class="stat-num">{{ files.length }} Files Stored</span>
          </div>
          <button class="btn-sm" title="Refresh" @click="fetchFiles">
            <img class="btn-icon" src="../assets/reload.png" alt="Refresh" />
          </button>
        </div>
        <div class="panel-content">
          <div v-if="files.length === 0" class="empty-state">
            <p>No files yet. Upload your first file!</p>
          </div>
          <ul v-else class="file-list">
            <li v-for="f in files" :key="f.name" class="file-item">
              <img class="file-icon-img" src="../assets/file.png" alt="File" />
              <div class="file-info">
                <span class="file-name">{{ f.name }}</span>
                <span class="file-meta">Level ID: {{ f.level_id }} · Level Name: {{ f.level_name }}</span>
              </div>
              <div class="file-actions">
                <button class="btn-sm" title="Download" :disabled="downloadingFiles.has(f.name)" @click="doDownload(f.name)">
                  <img v-if="!downloadingFiles.has(f.name)" class="btn-icon" src="../assets/download.png" alt="Download" />
                  <span v-else class="spinner"></span>
                </button>
                <button class="btn-sm danger" title="Delete" @click="doDelete(f.name)">
                  <img class="btn-icon" src="../assets/delete.png" alt="Delete" />
                </button>
              </div>
            </li>
          </ul>
        </div>
      </div>

      <div v-if="activeTab === 'upload'" class="panel">
        <div class="panel-header"><h2>Upload a File</h2></div>
        <div class="panel-content">
          <div
            :class="['drop-zone', { dragging: uploadDragging, 'has-file': !!uploadFile }]"
            @dragover.prevent="uploadDragging = true"
            @dragleave="uploadDragging = false"
            @drop.prevent="onDrop"
            @click="($refs.fileInput as any).click()"
          >
            <input ref="fileInput" type="file" style="display:none" @change="onFileInput" />
            <div v-if="!uploadFile" class="drop-inner">
              <img class="btn-icon" src="../assets/upload.png" />
              <p class="drop-title">Drop a file here or click to browse</p>
            </div>
            <div v-else class="drop-inner chosen">
              <img class="btn-icon" src="../assets/file.png" />
              <p class="drop-title">{{ uploadFile.name }}</p>
            </div>
          </div>
          <button class="btn-primary" :class="{ loading }" :disabled="loading || !uploadFile" @click="doUpload">
            <span v-if="!loading"><img class="btn-icon" src="../assets/upload.png" /> Upload to GD</span>
            <span v-else class="spinner"></span>
          </button>
        </div>
      </div>

    </div>
    <footer class="app-footer">
      <p>Credits to <a href="https://www.youtube.com/@SweepSweep2" target="_blank" rel="noopener noreferrer">@SweepSweep2 on YouTube</a></p>
    </footer>
  </div>
</template>
