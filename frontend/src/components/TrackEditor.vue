<script setup lang="ts">
import { ref, watch } from 'vue'
import { DialogClose, DialogContent, DialogDescription, DialogOverlay, DialogPortal, DialogRoot, DialogTitle } from 'reka-ui'
import { ImagePlus, Music, Trash2, X } from '@lucide/vue'
import { toast } from 'vue-sonner'
import { artworkUrl, formatDuration, type Track } from '../types'

const props = defineProps<{
  open: boolean
  track: Track | null
}>()

const emit = defineEmits<{
  'update:open': [boolean]
  saved: [Track]
  deleted: [number]
}>()

const form = ref({
  title: '',
  artist: '',
  album: '',
  album_artist: '',
  genre: '',
  year: 0,
  track_number: 0,
  disc_number: 0,
})
const artworkKey = ref('')
const saving = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

watch(() => [props.open, props.track] as const, ([open, track]) => {
  if (!open || !track) return
  form.value = {
    title: track.title,
    artist: track.artist,
    album: track.album,
    album_artist: track.album_artist,
    genre: track.genre,
    year: track.year,
    track_number: track.track_number,
    disc_number: track.disc_number,
  }
  artworkKey.value = track.artwork_s3_key
})

async function save() {
  if (!props.track || !form.value.title.trim()) return
  saving.value = true
  try {
    const res = await fetch(`/api/tracks/${props.track.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(form.value),
    })
    if (!res.ok) throw new Error('Could not save track')
    const updated = await res.json() as Track
    toast.success('Track updated')
    emit('saved', updated)
    emit('update:open', false)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not save track')
  } finally {
    saving.value = false
  }
}

async function replaceArtwork(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !props.track) return
  saving.value = true
  try {
    const body = new FormData()
    body.append('file', file)
    const res = await fetch(`/api/tracks/${props.track.id}/artwork`, { method: 'POST', body })
    if (!res.ok) {
      const data = await res.json().catch(() => null) as { error?: string } | null
      throw new Error(data?.error || 'Could not replace artwork')
    }
    const updated = await res.json() as Track
    artworkKey.value = updated.artwork_s3_key
    toast.success('Artwork updated')
    emit('saved', updated)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not replace artwork')
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!props.track) return
  if (!confirm(`Delete "${props.track.title}"? This removes the file from storage.`)) return
  saving.value = true
  try {
    const res = await fetch(`/api/tracks/${props.track.id}`, { method: 'DELETE' })
    if (!res.ok) throw new Error('Could not delete track')
    toast.success('Track deleted')
    emit('deleted', props.track.id)
    emit('update:open', false)
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not delete track')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <DialogRoot :open="open" @update:open="emit('update:open', $event)">
    <DialogPortal>
      <DialogOverlay class="fixed inset-0 z-50 bg-black/60" />
      <DialogContent class="fixed left-1/2 top-1/2 z-50 max-h-[min(100%-2rem,44rem)] w-[min(100%-2rem,36rem)] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl border border-border bg-surface p-5 shadow-2xl">
        <div class="mb-4 flex items-start justify-between gap-4">
          <div class="min-w-0">
            <DialogTitle class="truncate text-lg font-bold">{{ track?.title || 'Track' }}</DialogTitle>
            <DialogDescription class="mt-1 text-sm text-textSecondary">
              {{ track ? formatDuration(track.duration_seconds) : '' }}
              <span v-if="track?.bitrate"> · {{ track.bitrate }} kbps</span>
            </DialogDescription>
          </div>
          <DialogClose class="rounded-lg p-2 text-textSecondary hover:bg-white/5 hover:text-textPrimary" aria-label="Close">
            <X class="h-4 w-4" />
          </DialogClose>
        </div>

        <div v-if="track" class="space-y-4">
          <button
            type="button"
            class="group relative mx-auto block h-48 w-48 overflow-hidden rounded-xl bg-white/5"
            :disabled="saving"
            @click="fileInput?.click()"
          >
            <img v-if="artworkKey" :src="artworkUrl(artworkKey)" alt="" class="h-full w-full object-cover" />
            <span v-else class="flex h-full w-full items-center justify-center">
              <Music class="h-10 w-10 text-textSecondary" />
            </span>
            <span class="absolute inset-0 flex items-end justify-center bg-black/0 p-3 text-sm opacity-0 transition group-hover:bg-black/50 group-hover:opacity-100">
              <span class="inline-flex items-center gap-1.5"><ImagePlus class="h-4 w-4" /> Replace artwork</span>
            </span>
          </button>
          <input ref="fileInput" type="file" accept="image/jpeg,image/png,image/webp" class="hidden" @change="replaceArtwork" />

          <form class="grid grid-cols-2 gap-3" @submit.prevent="save">
            <label class="col-span-2 text-xs text-textSecondary">Title
              <input v-model="form.title" required class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary outline-none focus:border-accent/60" />
            </label>
            <label class="text-xs text-textSecondary">Artist
              <input v-model="form.artist" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary outline-none focus:border-accent/60" />
            </label>
            <label class="text-xs text-textSecondary">Album artist
              <input v-model="form.album_artist" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary outline-none focus:border-accent/60" />
            </label>
            <label class="col-span-2 text-xs text-textSecondary">Album
              <input v-model="form.album" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary outline-none focus:border-accent/60" />
            </label>
            <label class="text-xs text-textSecondary">Genre
              <input v-model="form.genre" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary outline-none focus:border-accent/60" />
            </label>
            <label class="text-xs text-textSecondary">Year
              <input v-model.number="form.year" type="number" min="0" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary outline-none focus:border-accent/60" />
            </label>
            <label class="text-xs text-textSecondary">Track number
              <input v-model.number="form.track_number" type="number" min="0" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary outline-none focus:border-accent/60" />
            </label>
            <label class="text-xs text-textSecondary">Disc number
              <input v-model.number="form.disc_number" type="number" min="0" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary outline-none focus:border-accent/60" />
            </label>

            <div class="col-span-2 mt-2 flex items-center justify-between gap-3">
              <button type="button" class="inline-flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-red-400 hover:bg-red-400/10" :disabled="saving" @click="remove">
                <Trash2 class="h-4 w-4" /> Delete
              </button>
              <button type="submit" class="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-background disabled:opacity-50" :disabled="saving">
                Save
              </button>
            </div>
          </form>
        </div>
      </DialogContent>
    </DialogPortal>
  </DialogRoot>
</template>
