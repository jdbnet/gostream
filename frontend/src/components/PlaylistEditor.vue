<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { Plus, Trash2, Search, GripVertical } from '@lucide/vue'

const props = defineProps<{
  playlist: any
}>()

const playlistTracks = ref<any[]>([])
const allTracks = ref<any[]>([])
const search = ref('')

const fetchPlaylistTracks = async () => {
  const res = await fetch(`/api/playlists/${props.playlist.id}/tracks`)
  if (res.ok) {
    playlistTracks.value = await res.json()
  }
}

const fetchAllTracks = async () => {
  const res = await fetch('/api/tracks?limit=1000')
  if (res.ok) {
    allTracks.value = await res.json()
  }
}

const addTrack = async (track: any) => {
  const position = playlistTracks.value.length
  await fetch(`/api/playlists/${props.playlist.id}/tracks`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ track_id: track.id, position })
  })
  fetchPlaylistTracks()
}

const removeTrack = async (trackId: number) => {
  await fetch(`/api/playlists/${props.playlist.id}/tracks/${trackId}`, { method: 'DELETE' })
  fetchPlaylistTracks()
}

// Drag and drop state
let draggedIndex: number | null = null

const onDragStart = (index: number) => {
  draggedIndex = index
}

const onDragOver = (e: DragEvent) => {
  e.preventDefault()
}

const onDrop = async (index: number) => {
  if (draggedIndex === null || draggedIndex === index) return
  
  // Reorder locally
  const item = playlistTracks.value.splice(draggedIndex, 1)[0]
  playlistTracks.value.splice(index, 0, item)
  draggedIndex = null

  // In a real app we'd save the new order to the backend, 
  // but playback is shuffled so position is mostly informational right now.
  // We'd probably need a batch update endpoint to save positions.
}

watch(() => props.playlist, () => {
  fetchPlaylistTracks()
}, { immediate: true })

onMounted(() => {
  fetchAllTracks()
})

const filteredTracks = computed(() => {
  if (!search.value) return allTracks.value
  const s = search.value.toLowerCase()
  return allTracks.value.filter(t => 
    t.title.toLowerCase().includes(s) || 
    t.artist.toLowerCase().includes(s)
  )
})

import { computed } from 'vue'
</script>

<template>
  <div class="h-full flex flex-col">
    <div class="p-6 border-b border-border bg-surface/50">
      <h2 class="text-2xl font-bold text-white">{{ playlist.name }}</h2>
      <p class="text-sm text-textSecondary mt-1">
        {{ playlistTracks.length }} tracks
      </p>
    </div>

    <div class="flex-1 flex overflow-hidden">
      <!-- Tracks in playlist -->
      <div class="w-1/2 border-r border-border flex flex-col">
        <div class="p-4 bg-white/[0.02] border-b border-border text-sm font-medium text-textSecondary">
          Current Tracks (Drag to reorder)
        </div>
        <div class="flex-1 overflow-y-auto p-4 space-y-2">
          <div v-if="playlistTracks.length === 0" class="text-center text-textSecondary py-8">
            Playlist is empty. Add tracks from the right.
          </div>
          <div 
            v-for="(track, idx) in playlistTracks" 
            :key="track.id + '-' + idx"
            draggable="true"
            @dragstart="onDragStart(idx)"
            @dragover="onDragOver"
            @drop="onDrop(idx)"
            class="flex items-center justify-between p-3 bg-surface border border-border rounded-lg group hover:border-accent/30 cursor-move"
          >
            <div class="flex items-center min-w-0">
              <GripVertical class="w-4 h-4 text-textSecondary/50 mr-3 shrink-0" />
              <div class="min-w-0">
                <div class="text-white truncate font-medium">{{ track.title }}</div>
                <div class="text-xs text-textSecondary truncate">{{ track.artist || 'Unknown' }}</div>
              </div>
            </div>
            <button 
              @click.stop="removeTrack(track.id)"
              class="text-red-400 hover:text-red-300 p-2 opacity-0 group-hover:opacity-100 transition-opacity"
            >
              <Trash2 class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      <!-- Track Picker -->
      <div class="w-1/2 flex flex-col bg-surface/30">
        <div class="p-4 border-b border-border">
          <div class="relative">
            <Search class="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-textSecondary" />
            <input 
              v-model="search"
              type="text"
              placeholder="Search library..."
              class="w-full bg-surface border border-border rounded-lg pl-9 pr-4 py-2 text-sm text-white focus:border-accent/50 outline-none"
            />
          </div>
        </div>
        <div class="flex-1 overflow-y-auto p-4 space-y-2">
          <div 
            v-for="track in filteredTracks" 
            :key="track.id"
            class="flex items-center justify-between p-3 bg-surface border border-border rounded-lg group hover:border-accent/30"
          >
            <div class="min-w-0">
              <div class="text-white truncate font-medium">{{ track.title }}</div>
              <div class="text-xs text-textSecondary truncate">{{ track.artist || 'Unknown' }}</div>
            </div>
            <button 
              @click="addTrack(track)"
              class="bg-accent/10 text-accent hover:bg-accent/20 p-2 rounded-lg transition-colors"
            >
              <Plus class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
