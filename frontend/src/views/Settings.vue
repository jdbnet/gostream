<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Save, AlertCircle, CheckCircle2, FolderSearch } from '@lucide/vue'

const config = ref<any>(null)
const saved = ref(false)
const error = ref('')
const scanResult = ref<any>(null)
const scanning = ref(false)

const fetchConfig = async () => {
  const res = await fetch('/api/config')
  if (res.ok) {
    config.value = await res.json()
  }
}

const saveConfig = async () => {
  try {
    const res = await fetch('/api/config', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config.value)
    })
    if (!res.ok) throw new Error(await res.text())

    saved.value = true
    error.value = ''
    setTimeout(() => saved.value = false, 3000)
  } catch (e: any) {
    error.value = e.message
  }
}

const scanLibrary = async () => {
  scanning.value = true
  scanResult.value = null
  error.value = ''
  try {
    const res = await fetch('/api/storage/scan', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'Scan failed')
    scanResult.value = data
  } catch (e: any) {
    error.value = e.message
  } finally {
    scanning.value = false
  }
}

onMounted(fetchConfig)
</script>

<template>
  <div class="max-w-4xl mx-auto space-y-8 pb-12">
    <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <h1 class="text-2xl font-bold sm:text-3xl">Settings</h1>
      <button
        @click="saveConfig"
        class="flex w-full items-center justify-center space-x-2 rounded-xl bg-accent px-6 py-2.5 font-medium text-background transition-colors hover:bg-accent/90 sm:w-auto"
      >
        <Save class="w-5 h-5" />
        <span>Save Configuration</span>
      </button>
    </div>

    <div v-if="saved" class="bg-green-400/10 text-green-400 p-4 rounded-lg border border-green-400/20 flex items-center">
      <CheckCircle2 class="w-5 h-5 mr-3" />
      Settings saved and applied successfully.
    </div>

    <div v-if="error" class="bg-red-400/10 text-red-400 p-4 rounded-lg border border-red-400/20 flex items-center">
      <AlertCircle class="w-5 h-5 mr-3" />
      {{ error }}
    </div>

    <div v-if="config" class="space-y-8">

      <!-- General Settings -->
      <section class="glass rounded-xl p-6 border border-border">
        <h2 class="text-xl font-bold mb-6 text-white border-b border-border pb-4">General</h2>
        <div class="space-y-2">
          <label class="text-sm text-textSecondary">Timezone</label>
          <input
            v-model="config.server.timezone"
            type="text"
            placeholder="Europe/London"
            class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50"
          />
          <p class="text-xs text-textSecondary/60">IANA timezone name used for timetable scheduling. Can also be set via the GOSTREAM_TIMEZONE environment variable.</p>
        </div>
      </section>

      <!-- Database Settings -->
      <section class="glass rounded-xl p-6 border border-border">
        <h2 class="text-xl font-bold mb-6 text-white border-b border-border pb-4">Database</h2>
        <div class="space-y-6">
          <div class="flex flex-wrap gap-6">
            <label class="flex items-center space-x-2 cursor-pointer">
              <input v-model="config.database.driver" type="radio" value="sqlite" class="accent-accent" />
              <span class="text-sm text-textSecondary">SQLite</span>
            </label>
            <label class="flex items-center space-x-2 cursor-pointer">
              <input v-model="config.database.driver" type="radio" value="mariadb" class="accent-accent" />
              <span class="text-sm text-textSecondary">MariaDB</span>
            </label>
          </div>

          <div v-if="config.database.driver === 'sqlite'" class="space-y-2">
            <label class="text-sm text-textSecondary">Database File Path</label>
            <input v-model="config.database.path" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
          </div>

          <div v-else class="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Host</label>
              <input v-model="config.database.host" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Port</label>
              <input v-model.number="config.database.port" type="number" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Database Name</label>
              <input v-model="config.database.name" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">User</label>
              <input v-model="config.database.user" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2 col-span-2">
              <label class="text-sm text-textSecondary">Password</label>
              <input v-model="config.database.password" type="password" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
          </div>
        </div>
      </section>

      <!-- Storage Settings -->
      <section class="glass rounded-xl p-6 border border-border">
        <h2 class="text-xl font-bold mb-6 text-white border-b border-border pb-4">Storage</h2>
        <div class="space-y-6">
          <div class="flex flex-wrap gap-6">
            <label class="flex items-center space-x-2 cursor-pointer">
              <input v-model="config.storage.type" type="radio" value="local" class="accent-accent" />
              <span class="text-sm text-textSecondary">Local storage</span>
            </label>
            <label class="flex items-center space-x-2 cursor-pointer">
              <input v-model="config.storage.type" type="radio" value="s3" class="accent-accent" />
              <span class="text-sm text-textSecondary">S3 storage</span>
            </label>
          </div>

          <div v-if="config.storage.type === 'local'" class="space-y-6">
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Local Path</label>
              <input v-model="config.storage.local_path" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
              <p class="text-xs text-textSecondary/60">Audio files are stored in tracks/, jingles/, and artworks/ subdirectories.</p>
            </div>
            <div class="space-y-2 flex items-center">
              <input v-model="config.storage.auto_scan_on_startup" type="checkbox" id="autoScan" class="mr-2 accent-accent" />
              <label for="autoScan" class="text-sm text-textSecondary">Auto-scan library on startup</label>
            </div>
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
              <button
                @click="scanLibrary"
                :disabled="scanning"
                class="flex items-center justify-center space-x-2 rounded-xl border border-border px-5 py-2.5 text-white transition-colors hover:bg-surface disabled:opacity-50"
              >
                <FolderSearch class="w-5 h-5" />
                <span>{{ scanning ? 'Scanning...' : 'Scan Library' }}</span>
              </button>
              <p v-if="scanResult" class="text-sm text-textSecondary">
                Imported {{ scanResult.imported }}, skipped {{ scanResult.skipped }}
                <span v-if="scanResult.errors?.length">, {{ scanResult.errors.length }} errors</span>
              </p>
            </div>
          </div>

          <div v-else class="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div class="space-y-2 col-span-2">
              <label class="text-sm text-textSecondary">Endpoint URL (Optional)</label>
              <input v-model="config.storage.s3.endpoint" type="text" placeholder="https://..." class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Bucket Name</label>
              <input v-model="config.storage.s3.bucket" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Region</label>
              <input v-model="config.storage.s3.region" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Access Key</label>
              <input v-model="config.storage.s3.access_key" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Secret Key</label>
              <input v-model="config.storage.s3.secret_key" type="password" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2 col-span-2 flex items-center">
              <input v-model="config.storage.s3.path_style" type="checkbox" id="pathStyle" class="mr-2 accent-accent" />
              <label for="pathStyle" class="text-sm text-textSecondary">Force Path Style (Required for MinIO/Wasabi/etc)</label>
            </div>
          </div>
        </div>
      </section>

      <!-- Icecast Settings -->
      <section class="glass rounded-xl p-6 border border-border">
        <h2 class="text-xl font-bold mb-6 text-white border-b border-border pb-4">Icecast Source</h2>
        <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
          <div class="space-y-2">
            <label class="text-sm text-textSecondary">Protocol</label>
            <select v-model="config.icecast.protocol" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50 appearance-none">
              <option value="http">HTTP</option>
              <option value="https">HTTPS</option>
            </select>
          </div>
          <div class="space-y-2">
            <label class="text-sm text-textSecondary">Host</label>
            <input v-model="config.icecast.host" type="text" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
          </div>
          <div class="space-y-2">
            <label class="text-sm text-textSecondary">Port</label>
            <input v-model.number="config.icecast.port" type="number" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
          </div>
          <div class="space-y-2">
            <label class="text-sm text-textSecondary">Mount Point</label>
            <input v-model="config.icecast.mount" type="text" placeholder="/stream" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
          </div>
          <div class="space-y-2">
            <label class="text-sm text-textSecondary">Source Password</label>
            <input v-model="config.icecast.password" type="password" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
          </div>
          <div class="space-y-2">
            <label class="text-sm text-textSecondary">Admin Password</label>
            <input v-model="config.icecast.admin_password" type="password" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
          </div>

          <div class="space-y-2">
            <label class="text-sm text-textSecondary">Target Bitrate (kbps) - Note: Must match ffmpeg target</label>
            <input v-model.number="config.icecast.bitrate" type="number" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
          </div>
        </div>
      </section>

      <!-- Stream Engine Settings -->
      <section class="glass rounded-xl p-6 border border-border">
        <h2 class="text-xl font-bold mb-6 text-white border-b border-border pb-4">Stream Engine</h2>
        <div class="space-y-8">
          <div class="space-y-4">
            <label class="text-sm text-textSecondary flex justify-between">
              <span>Jingle Interval</span>
              <span class="text-accent font-mono bg-accent/10 px-2 py-0.5 rounded">Every {{ config.stream.jingle_interval }} tracks</span>
            </label>
            <input
              v-model.number="config.stream.jingle_interval"
              type="range"
              min="0"
              max="20"
              class="w-full accent-accent"
            />
            <p class="text-xs text-textSecondary/60 mt-1">Set to 0 to disable jingles.</p>
          </div>

          <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Reconnect Delay (Seconds)</label>
              <input v-model.number="config.stream.reconnect_delay_seconds" type="number" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
            <div class="space-y-2">
              <label class="text-sm text-textSecondary">Buffer Amount (Seconds)</label>
              <input v-model.number="config.stream.buffer_seconds" type="number" class="w-full bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50" />
            </div>
          </div>
        </div>
      </section>

    </div>
  </div>
</template>
