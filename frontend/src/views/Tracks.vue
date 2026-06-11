<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Search, Trash2, Edit2, Save, X } from 'lucide-vue-next'
import TrackUploader from '../components/TrackUploader.vue'

const tracks = ref<any[]>([])
const playlists = ref<any[]>([])
const search = ref('')

const editingTrackId = ref<number | null>(null)
const editForm = ref({ title: '', artist: '' })

const fetchTracks = async () => {
  const res = await fetch('/api/tracks')
  if (res.ok) {
    tracks.value = await res.json()
  }
}

const fetchPlaylists = async () => {
  const res = await fetch('/api/playlists')
  if (res.ok) {
    playlists.value = await res.json()
  }
}

const deleteTrack = async (id: number) => {
  if (!confirm('Are you sure you want to delete this track?')) return
  await fetch(`/api/tracks/${id}`, { method: 'DELETE' })
  fetchTracks()
}

const startEdit = (track: any) => {
  editingTrackId.value = track.id
  editForm.value = { title: track.title, artist: track.artist || '' }
}

const cancelEdit = () => {
  editingTrackId.value = null
}

const saveTrack = async (id: number) => {
  const res = await fetch(`/api/tracks/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(editForm.value)
  })
  if (res.ok) {
    editingTrackId.value = null
    fetchTracks()
  } else {
    alert('Failed to update track')
  }
}

const filteredTracks = computed(() => {
  if (!search.value) return tracks.value
  const s = search.value.toLowerCase()
  return tracks.value.filter(t => 
    t.title.toLowerCase().includes(s) || 
    (t.artist && t.artist.toLowerCase().includes(s))
  )
})

onMounted(() => {
  fetchTracks()
  fetchPlaylists()
})
</script>

<template>
  <div class="max-w-6xl mx-auto space-y-8">
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold">Tracks</h1>
    </div>

    <TrackUploader 
      endpoint="/api/tracks" 
      label="Upload Track" 
      :playlists="playlists"
      @uploaded="fetchTracks"
    />

    <div class="glass rounded-xl overflow-hidden border border-border">
      <div class="p-4 border-b border-border flex items-center bg-white/[0.02]">
        <Search class="w-5 h-5 text-textSecondary mr-3" />
        <input 
          type="text" 
          v-model="search"
          placeholder="Search tracks by title or artist..."
          class="bg-transparent border-none outline-none text-white w-full placeholder-textSecondary/50"
        />
      </div>

      <table class="w-full text-left">
        <thead class="bg-white/5 border-b border-border text-textSecondary text-sm uppercase tracking-wider">
          <tr>
            <th class="px-6 py-4 font-medium">Title</th>
            <th class="px-6 py-4 font-medium">Artist</th>
            <th class="px-6 py-4 font-medium">Plays</th>
            <th class="px-6 py-4 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="filteredTracks.length === 0">
            <td colspan="4" class="px-6 py-8 text-center text-textSecondary">
              No tracks found.
            </td>
          </tr>
          <tr v-for="track in filteredTracks" :key="track.id" class="hover:bg-white/[0.02] transition-colors group">
            <template v-if="editingTrackId === track.id">
              <td class="px-6 py-4">
                <input v-model="editForm.title" type="text" class="w-full bg-surface border border-border rounded px-3 py-1 text-white outline-none focus:border-accent/50" />
              </td>
              <td class="px-6 py-4">
                <input v-model="editForm.artist" type="text" placeholder="Unknown Artist" class="w-full bg-surface border border-border rounded px-3 py-1 text-white outline-none focus:border-accent/50" />
              </td>
              <td class="px-6 py-4 text-textSecondary">{{ track.play_count }}</td>
              <td class="px-6 py-4 text-right">
                <div class="flex items-center justify-end space-x-2 opacity-100">
                  <button @click="saveTrack(track.id)" class="text-green-400 hover:text-green-300 p-2 rounded-lg hover:bg-green-400/10 transition-colors" title="Save">
                    <Save class="w-4 h-4" />
                  </button>
                  <button @click="cancelEdit" class="text-textSecondary hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors" title="Cancel">
                    <X class="w-4 h-4" />
                  </button>
                </div>
              </td>
            </template>
            <template v-else>
              <td class="px-6 py-4 font-medium text-white">{{ track.title }}</td>
              <td class="px-6 py-4 text-textSecondary">{{ track.artist || 'Unknown' }}</td>
              <td class="px-6 py-4 text-textSecondary">{{ track.play_count }}</td>
              <td class="px-6 py-4 text-right opacity-0 group-hover:opacity-100 transition-opacity">
                <div class="flex items-center justify-end space-x-2">
                  <button @click="startEdit(track)" class="text-accent hover:text-accent/80 p-2 rounded-lg hover:bg-accent/10 transition-colors" title="Edit">
                    <Edit2 class="w-4 h-4" />
                  </button>
                  <button @click="deleteTrack(track.id)" class="text-red-400 hover:text-red-300 p-2 rounded-lg hover:bg-red-400/10 transition-colors" title="Delete">
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </td>
            </template>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
