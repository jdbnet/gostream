<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { SkipForward, Activity, Music } from 'lucide-vue-next'

const status = ref({
  is_connected: false,
  is_reconnecting: false,
  current_track: null as any,
  current_jingle: null as any,
  upcoming_tracks: [] as any[]
})

const history = ref<any[]>([])
let pollInterval: any = null

const fetchStatus = async () => {
  try {
    const res = await fetch('/api/status')
    if (res.ok) {
      status.value = await res.json()
    }
  } catch (e) {
    console.error(e)
  }
}

const fetchHistory = async () => {
  try {
    const res = await fetch('/api/history')
    if (res.ok) {
      history.value = await res.json()
    }
  } catch (e) {
    console.error(e)
  }
}

const skipTrack = async () => {
  await fetch('/api/stream/skip', { method: 'POST' })
  setTimeout(() => {
    fetchStatus()
    fetchHistory()
  }, 1000)
}

onMounted(() => {
  fetchStatus()
  fetchHistory()
  pollInterval = setInterval(() => {
    fetchStatus()
  }, 3000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<template>
  <div class="max-w-4xl mx-auto space-y-8">
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold">Now Playing</h1>
      <div class="flex items-center space-x-3 bg-surface px-4 py-2 rounded-full border border-border">
        <Activity class="w-4 h-4" :class="status.is_connected ? 'text-green-400' : 'text-red-400'" />
        <span class="text-sm font-medium">
          {{ status.is_connected ? 'Stream Live' : status.is_reconnecting ? 'Reconnecting...' : 'Offline' }}
        </span>
      </div>
    </div>

    <!-- Main Player Card -->
    <div class="glass rounded-2xl p-8 relative overflow-hidden group">
      <!-- Decorative background blur -->
      <div class="absolute -inset-20 bg-accent/10 blur-3xl rounded-full opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none"></div>
      
      <div class="relative flex items-center space-x-8">
        <div class="w-48 h-48 rounded-xl bg-gradient-to-br from-surface to-border flex items-center justify-center shadow-2xl shrink-0 border border-white/5">
          <Music class="w-16 h-16 text-textSecondary opacity-50" />
        </div>
        
        <div class="flex-1 min-w-0">
          <div v-if="status.current_track" class="space-y-2">
            <h2 class="text-4xl font-bold truncate text-white drop-shadow-md">
              {{ status.current_track.title }}
            </h2>
            <p class="text-2xl text-accent truncate">
              {{ status.current_track.artist || 'Unknown Artist' }}
            </p>
          </div>
          <div v-else-if="status.current_jingle" class="space-y-2">
            <h2 class="text-4xl font-bold text-white drop-shadow-md">
              GoStream Jingle
            </h2>
            <p class="text-2xl text-accent truncate">
              {{ status.current_jingle.name }}
            </p>
          </div>
          <div v-else class="space-y-2">
            <h2 class="text-4xl font-bold text-textSecondary">Not Playing</h2>
            <p class="text-xl text-textSecondary/60">Stream is idle or buffering</p>
          </div>

          <div class="mt-8 flex items-center space-x-4">
            <button 
              @click="skipTrack"
              class="flex items-center space-x-2 bg-white/5 hover:bg-white/10 text-white px-6 py-3 rounded-xl transition-all active:scale-95 border border-white/10 hover:border-accent/50"
            >
              <SkipForward class="w-5 h-5" />
              <span class="font-medium">Skip Track</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Upcoming Tracks -->
    <div v-if="status.upcoming_tracks && status.upcoming_tracks.length > 0" class="space-y-4">
      <h3 class="text-xl font-bold flex items-center">
        Upcoming Next
      </h3>
      <div class="bg-surface rounded-xl border border-border overflow-hidden p-2">
        <div 
          v-for="(track, index) in status.upcoming_tracks" 
          :key="index"
          class="flex items-center space-x-4 p-3 rounded-lg bg-white/[0.01] hover:bg-white/[0.03] transition-colors mb-2 last:mb-0"
        >
          <div class="w-10 h-10 rounded bg-white/5 flex items-center justify-center shrink-0 border border-border">
            <span class="text-textSecondary text-sm font-medium">{{ index + 1 }}</span>
          </div>
          <div class="flex-1 min-w-0">
            <h4 class="text-white font-medium truncate">{{ track.title }}</h4>
            <p class="text-textSecondary text-sm truncate">{{ track.artist || 'Unknown Artist' }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- History -->
    <div class="space-y-4">
      <h3 class="text-xl font-bold">Recent History</h3>
      <div class="bg-surface rounded-xl border border-border overflow-hidden">
        <div v-if="history.length === 0" class="p-8 text-center text-textSecondary">
          No play history yet.
        </div>
        <table v-else class="w-full text-left">
          <thead class="bg-white/5 border-b border-border">
            <tr>
              <th class="px-6 py-4 font-medium text-textSecondary">Time</th>
              <th class="px-6 py-4 font-medium text-textSecondary">Track / Jingle</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="entry in history" :key="entry.id" class="hover:bg-white/[0.02] transition-colors">
              <td class="px-6 py-4 text-textSecondary whitespace-nowrap">
                {{ new Date(entry.played_at).toLocaleTimeString() }}
              </td>
              <td class="px-6 py-4 font-medium text-white">
                <template v-if="entry.was_jingle">
                  <span class="text-accent">Jingle Played</span>
                </template>
                <template v-else-if="entry.track">
                  {{ entry.track.title }} <span class="text-textSecondary mx-2">•</span> <span class="text-textSecondary font-normal">{{ entry.track.artist }}</span>
                </template>
                <template v-else>
                  Unknown Track
                </template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
