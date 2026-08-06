<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { SkipForward, Activity, Music, Play, Square } from 'lucide-vue-next'

const status = ref({
  is_connected: false,
  is_reconnecting: false,
  current_track: null as any,
  current_jingle: null as any,
  upcoming_tracks: [] as any[]
})

const history = ref<any[]>([])
let pollInterval: any = null

const isPlaying = ref(false)
let audio: HTMLAudioElement | null = null

const togglePlay = () => {
  if (isPlaying.value) {
    if (audio) {
      audio.pause()
      audio.src = ''
      audio = null
    }
    isPlaying.value = false
  } else {
    audio = new Audio('https://icecast.jdb143.uk/gostream')
    audio.play().catch(e => console.error("Audio playback failed", e))
    isPlaying.value = true
  }
}

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
  if (audio) {
    audio.pause()
    audio.src = ''
    audio = null
  }
})
</script>

<template>
  <div class="max-w-4xl mx-auto space-y-8">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <h1 class="text-2xl font-bold sm:text-3xl">Now Playing</h1>
      <div class="flex w-fit items-center space-x-3 rounded-full border border-border bg-surface px-4 py-2">
        <Activity class="w-4 h-4" :class="status.is_connected ? 'text-green-400' : 'text-red-400'" />
        <span class="text-sm font-medium">
          {{ status.is_connected ? 'Stream Live' : status.is_reconnecting ? 'Reconnecting...' : 'Offline' }}
        </span>
      </div>
    </div>

    <!-- Main Player Card -->
    <div class="glass group relative overflow-hidden rounded-2xl p-5 sm:p-8">
      <!-- Decorative background blur -->
      <div class="absolute -inset-20 bg-accent/10 blur-3xl rounded-full opacity-0 group-hover:opacity-100 transition-opacity duration-700 pointer-events-none"></div>
      
      <div class="relative flex flex-col items-center gap-6 sm:flex-row sm:items-center sm:gap-8">
        <div class="flex h-40 w-40 shrink-0 items-center justify-center overflow-hidden rounded-xl border border-white/5 bg-gradient-to-br from-surface to-border shadow-2xl sm:h-48 sm:w-48">
          <img 
            v-if="status.current_track?.artwork_s3_key" 
            :src="'/api/artwork?key=' + status.current_track.artwork_s3_key" 
            alt="Album Art" 
            class="w-full h-full object-cover"
          />
          <Music v-else class="w-16 h-16 text-textSecondary opacity-50" />
        </div>
        
        <div class="min-w-0 flex-1 text-center sm:text-left">
          <div v-if="status.current_track" class="space-y-2">
            <h2 class="truncate text-2xl font-bold text-white drop-shadow-md sm:text-4xl">
              {{ status.current_track.title }}
            </h2>
            <p class="truncate text-xl text-accent sm:text-2xl">
              {{ status.current_track.artist || 'Unknown Artist' }}
            </p>
          </div>
          <div v-else-if="status.current_jingle" class="space-y-2">
            <h2 class="text-2xl font-bold text-white drop-shadow-md sm:text-4xl">
              GoStream Jingle
            </h2>
            <p class="truncate text-xl text-accent sm:text-2xl">
              {{ status.current_jingle.name }}
            </p>
          </div>
          <div v-else class="space-y-2">
            <h2 class="text-2xl font-bold text-textSecondary sm:text-4xl">Not Playing</h2>
            <p class="text-lg text-textSecondary/60 sm:text-xl">Stream is idle or buffering</p>
          </div>

          <div class="mt-6 flex flex-col gap-3 sm:mt-8 sm:flex-row sm:items-center sm:gap-4">
            <button 
              @click="togglePlay"
              class="flex items-center justify-center space-x-2 rounded-xl bg-accent px-6 py-3 text-white shadow-lg shadow-accent/20 transition-all hover:bg-accent/90 active:scale-95"
            >
              <Square v-if="isPlaying" class="w-5 h-5 fill-current" />
              <Play v-else class="w-5 h-5 fill-current" />
              <span class="font-medium">{{ isPlaying ? 'Stop stream' : 'Listen Live' }}</span>
            </button>

            <button  
              @click="skipTrack"
              class="flex items-center justify-center space-x-2 rounded-xl border border-white/10 bg-white/5 px-6 py-3 text-white transition-all hover:border-accent/50 hover:bg-white/10 active:scale-95"
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
          <div class="w-10 h-10 rounded bg-white/5 flex items-center justify-center shrink-0 border border-border overflow-hidden">
            <img 
              v-if="track.artwork_s3_key" 
              :src="'/api/artwork?key=' + track.artwork_s3_key" 
              alt="Album Art" 
              class="w-full h-full object-cover"
            />
            <span v-else class="text-textSecondary text-sm font-medium">{{ index + 1 }}</span>
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
        <div v-else class="overflow-x-auto">
        <table class="w-full min-w-[320px] text-left">
          <thead class="bg-white/5 border-b border-border">
            <tr>
              <th class="px-4 py-3 font-medium text-textSecondary sm:px-6 sm:py-4">Time</th>
              <th class="px-4 py-3 font-medium text-textSecondary sm:px-6 sm:py-4">Track / Jingle</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="entry in history" :key="entry.id" class="hover:bg-white/[0.02] transition-colors">
              <td class="whitespace-nowrap px-4 py-3 text-textSecondary sm:px-6 sm:py-4">
                {{ new Date(entry.played_at).toLocaleTimeString() }}
              </td>
              <td class="px-4 py-3 font-medium text-white sm:px-6 sm:py-4">
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
  </div>
</template>
