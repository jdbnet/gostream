<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, CheckCircle2, Trash2, ListMusic } from 'lucide-vue-next'
import PlaylistEditor from '../components/PlaylistEditor.vue'

const playlists = ref<any[]>([])
const newPlaylistName = ref('')
const selectedPlaylist = ref<any>(null)

const fetchPlaylists = async () => {
  const res = await fetch('/api/playlists')
  if (res.ok) {
    playlists.value = await res.json()
  }
}

const createPlaylist = async () => {
  if (!newPlaylistName.value) return
  await fetch('/api/playlists', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name: newPlaylistName.value })
  })
  newPlaylistName.value = ''
  fetchPlaylists()
}

const deletePlaylist = async (id: number) => {
  if (!confirm('Are you sure you want to delete this playlist?')) return
  await fetch(`/api/playlists/${id}`, { method: 'DELETE' })
  if (selectedPlaylist.value?.id === id) {
    selectedPlaylist.value = null
  }
  fetchPlaylists()
}

const setActive = async (id: number) => {
  await fetch(`/api/playlists/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ is_active: true })
  })
  fetchPlaylists()
}

onMounted(fetchPlaylists)
</script>

<template>
  <div class="max-w-6xl mx-auto flex gap-8 h-[calc(100vh-4rem)]">
    <!-- Playlists Sidebar -->
    <div class="w-1/3 flex flex-col space-y-4">
      <h1 class="text-3xl font-bold mb-4">Playlists</h1>
      
      <form @submit.prevent="createPlaylist" class="flex items-center space-x-2">
        <input 
          v-model="newPlaylistName" 
          type="text" 
          placeholder="New playlist name..."
          class="flex-1 bg-surface border border-border rounded-lg px-4 py-2 text-white outline-none focus:border-accent/50"
        />
        <button type="submit" class="bg-accent text-background p-2 rounded-lg hover:bg-accent/80 transition-colors">
          <Plus class="w-5 h-5" />
        </button>
      </form>

      <div class="flex-1 overflow-y-auto space-y-2 pr-2">
        <div 
          v-for="p in playlists" 
          :key="p.id"
          class="glass p-4 rounded-xl cursor-pointer transition-all border border-border group flex flex-col"
          :class="selectedPlaylist?.id === p.id ? 'border-accent ring-1 ring-accent/50' : 'hover:border-accent/30'"
          @click="selectedPlaylist = p"
        >
          <div class="flex items-center justify-between mb-2">
            <h3 class="font-bold text-lg text-white">{{ p.name }}</h3>
            <div class="flex items-center space-x-2">
              <button 
                @click.stop="setActive(p.id)"
                class="p-1.5 rounded-md transition-colors"
                :class="p.is_active ? 'text-accent bg-accent/10' : 'text-textSecondary hover:text-white hover:bg-white/10'"
                :title="p.is_active ? 'Active Playlist' : 'Set Active'"
              >
                <CheckCircle2 class="w-5 h-5" />
              </button>
              <button 
                @click.stop="deletePlaylist(p.id)"
                class="p-1.5 rounded-md text-red-400 hover:text-red-300 hover:bg-red-400/10 transition-colors opacity-0 group-hover:opacity-100"
              >
                <Trash2 class="w-5 h-5" />
              </button>
            </div>
          </div>
          <div class="text-xs text-textSecondary">
            {{ p.is_active ? 'Currently active for stream' : 'Inactive' }}
          </div>
        </div>
      </div>
    </div>

    <!-- Playlist Editor -->
    <div class="flex-1 glass rounded-2xl border border-border overflow-hidden flex flex-col relative">
      <PlaylistEditor 
        v-if="selectedPlaylist" 
        :playlist="selectedPlaylist" 
      />
      <div v-else class="absolute inset-0 flex items-center justify-center text-textSecondary flex-col">
        <ListMusic class="w-16 h-16 mb-4 opacity-20" />
        <p>Select a playlist to edit</p>
      </div>
    </div>
  </div>
</template>
