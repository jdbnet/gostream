<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { DialogClose, DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { ListPlus, Plus, X } from '@lucide/vue'
import { toast } from 'vue-sonner'
import { addTracksToPlaylist, createPlaylist, fetchPlaylists } from '../lib/api'
import type { Playlist, TrackQuery } from '../types'

const props = defineProps<{
  open: boolean
  trackIds: number[]
  filter: TrackQuery | null
  count: number
}>()

const emit = defineEmits<{
  'update:open': [boolean]
}>()

const playlists = ref<Playlist[]>([])
const name = ref('')
const saving = ref(false)

const label = computed(() => props.count === 1 ? '1 track' : `${props.count} tracks`)

watch(() => props.open, async (open) => {
  if (!open) return
  name.value = ''
  playlists.value = await fetchPlaylists()
})

async function addTo(playlistId: number) {
  saving.value = true
  try {
    const body = props.filter ? { filter: props.filter } : { track_ids: props.trackIds }
    const result = await addTracksToPlaylist(playlistId, body)
    toast.success(`Added ${result.added} tracks`)
    emit('update:open', false)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not add tracks')
  } finally {
    saving.value = false
  }
}

async function createAndAdd() {
  if (!name.value.trim()) return
  saving.value = true
  try {
    const playlist = await createPlaylist({ name: name.value.trim() })
    await addTo(playlist.id)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not create playlist')
    saving.value = false
  }
}
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-50 bg-black/60" />
      <DialogContent class="fixed left-1/2 top-1/2 z-50 w-[min(100%-2rem,28rem)] -translate-x-1/2 -translate-y-1/2 rounded-2xl border border-border bg-surface p-5 shadow-2xl">
        <div class="mb-4 flex items-start justify-between gap-4">
          <div>
            <DialogTitle class="text-lg font-bold">Add to playlist</DialogTitle>
            <DialogDescription class="mt-1 text-sm text-textSecondary">{{ label }}</DialogDescription>
          </div>
          <DialogClose class="rounded-lg p-2 text-textSecondary hover:bg-white/5 hover:text-textPrimary" aria-label="Close">
            <X class="h-4 w-4" />
          </DialogClose>
        </div>

        <form class="mb-4 flex gap-2" @submit.prevent="createAndAdd">
          <input
            v-model="name"
            type="text"
            placeholder="New playlist name"
            class="min-w-0 flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm outline-none focus:border-accent/60"
          />
          <button type="submit" class="rounded-lg bg-accent px-3 py-2 text-background" :disabled="saving" aria-label="Create playlist">
            <Plus class="h-4 w-4" />
          </button>
        </form>

        <div class="max-h-72 space-y-1 overflow-y-auto">
          <button
            v-for="playlist in playlists"
            :key="playlist.id"
            type="button"
            class="flex w-full items-center gap-3 rounded-lg px-3 py-2 text-left hover:bg-white/5"
            :disabled="saving"
            @click="addTo(playlist.id)"
          >
            <ListPlus class="h-4 w-4 text-accent" />
            <span class="truncate">{{ playlist.name }}</span>
          </button>
          <p v-if="playlists.length === 0" class="px-3 py-6 text-center text-sm text-textSecondary">No playlists yet.</p>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
