<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { CheckCircle2, Plus, Trash2 } from '@lucide/vue'
import { toast } from 'vue-sonner'
import PlaylistEditor from '../components/PlaylistEditor.vue'
import { createPlaylist, fetchPlaylists } from '../lib/api'
import { formatDuration, type Playlist } from '../types'

const playlists = ref<Playlist[]>([])
const name = ref('')
const kind = ref('manual')
const selected = ref<Playlist | null>(null)

async function load() {
  playlists.value = await fetchPlaylists()
  if (selected.value) {
    selected.value = playlists.value.find((playlist) => playlist.id === selected.value?.id) || null
  }
}

async function create() {
  if (!name.value.trim()) return
  try {
    const playlist = await createPlaylist({
      name: name.value.trim(),
      kind: kind.value,
      playback_mode: 'shuffle',
    })
    name.value = ''
    await load()
    selected.value = playlists.value.find((item) => item.id === playlist.id) || playlist
    toast.success('Playlist created')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not create playlist')
  }
}

async function remove(id: number) {
  if (!confirm('Delete this playlist?')) return
  await fetch(`/api/playlists/${id}`, { method: 'DELETE' })
  if (selected.value?.id === id) selected.value = null
  toast.success('Playlist deleted')
  load()
}

async function setActive(id: number, active: boolean) {
  await fetch(`/api/playlists/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ is_active: active }),
  })
  load()
}

onMounted(load)
</script>

<template>
  <div class="flex h-[calc(100vh-7.5rem)] flex-col gap-6 lg:h-[calc(100vh-4rem)] lg:flex-row">
    <div class="flex w-full shrink-0 flex-col gap-4 lg:w-80">
      <h1 class="text-2xl font-bold sm:text-3xl">Playlists</h1>
      <form class="flex gap-2" @submit.prevent="create">
        <input v-model="name" type="text" placeholder="New playlist" class="min-w-0 flex-1 rounded-xl border border-border bg-surface px-3 py-2 outline-none focus:border-accent/60" />
        <select v-model="kind" class="rounded-xl border border-border bg-surface px-2 text-sm">
          <option value="manual">Manual</option>
          <option value="smart">Smart</option>
        </select>
        <button type="submit" class="rounded-xl bg-accent p-2 text-background" aria-label="Create playlist"><Plus class="h-5 w-5" /></button>
      </form>
      <div class="min-h-0 flex-1 space-y-2 overflow-y-auto">
        <button
          v-for="playlist in playlists"
          :key="playlist.id"
          type="button"
          class="w-full rounded-xl border p-4 text-left"
          :class="selected?.id === playlist.id ? 'border-accent bg-accent/10' : 'border-border bg-surface hover:border-accent/30'"
          @click="selected = playlist"
        >
          <div class="flex items-start justify-between gap-2">
            <div>
              <p class="font-medium">{{ playlist.name }}</p>
              <p class="text-xs text-textSecondary">{{ playlist.kind }} · {{ playlist.track_count || 0 }} tracks · {{ formatDuration(playlist.total_duration || 0) }}</p>
            </div>
            <span class="flex gap-1">
              <button type="button" class="rounded-md p-1" :class="playlist.is_active ? 'text-accent' : 'text-textSecondary'" aria-label="Set active" @click.stop="setActive(playlist.id, !playlist.is_active)">
                <CheckCircle2 class="h-4 w-4" />
              </button>
              <button type="button" class="rounded-md p-1 text-textSecondary hover:text-red-300" aria-label="Delete playlist" @click.stop="remove(playlist.id)">
                <Trash2 class="h-4 w-4" />
              </button>
            </span>
          </div>
        </button>
        <p v-if="playlists.length === 0" class="py-8 text-center text-sm text-textSecondary">No playlists yet.</p>
      </div>
    </div>
    <div class="flex min-h-0 min-w-0 flex-1 flex-col">
      <PlaylistEditor v-if="selected" :playlist="selected" @changed="load" />
      <div v-else class="flex flex-1 items-center justify-center rounded-xl border border-dashed border-border text-textSecondary">Select a playlist</div>
    </div>
  </div>
</template>
