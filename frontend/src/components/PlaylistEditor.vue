<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { GripVertical, Music, Search, Trash2 } from '@lucide/vue'
import { toast } from 'vue-sonner'
import { artworkUrl, emptyRules, formatDuration, parseRules, type Playlist, type SmartRules, type Track } from '../types'

const props = defineProps<{
  playlist: Playlist
}>()

const emit = defineEmits<{
  changed: []
}>()

const tracks = ref<Track[]>([])
const total = ref(0)
const duration = ref(0)
const artwork = ref<string[]>([])
const search = ref('')
const loading = ref(false)
const selected = ref<Set<number>>(new Set())
const anchor = ref<number | null>(null)
const dragId = ref<number | null>(null)
const rules = ref<SmartRules>(emptyRules())
const artistText = ref('')
const albumText = ref('')
const genreText = ref('')

const isSmart = computed(() => props.playlist.kind === 'smart')

function splitList(value: string) {
  return value.split(',').map((part) => part.trim()).filter(Boolean)
}

async function load() {
  loading.value = true
  try {
    const params = new URLSearchParams({ limit: '500', offset: '0' })
    if (search.value.trim()) params.set('q', search.value.trim())
    const res = await fetch(`/api/playlists/${props.playlist.id}/tracks?${params}`)
    if (!res.ok) throw new Error('Could not load playlist')
    const data = await res.json()
    tracks.value = data.tracks || []
    total.value = data.total || 0
    duration.value = data.total_duration || 0
    artwork.value = data.artwork || []
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not load playlist')
  } finally {
    loading.value = false
  }
}

async function saveSettings() {
  rules.value.artist_in = splitList(artistText.value)
  rules.value.album_in = splitList(albumText.value)
  rules.value.genre_in = splitList(genreText.value)
  const res = await fetch(`/api/playlists/${props.playlist.id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: props.playlist.name,
      kind: props.playlist.kind,
      rules: JSON.stringify(rules.value),
      playback_mode: props.playlist.playback_mode,
      artist_separation: props.playlist.artist_separation,
      track_repeat_hours: props.playlist.track_repeat_hours,
    }),
  })
  if (!res.ok) {
    toast.error('Could not save playlist')
    return
  }
  toast.success('Playlist saved')
  emit('changed')
  load()
}

function onClick(index: number, event: MouseEvent) {
  const id = tracks.value[index].id
  if (event.shiftKey && anchor.value !== null) {
    const start = Math.min(anchor.value, index)
    const end = Math.max(anchor.value, index)
    const next = new Set(selected.value)
    for (let i = start; i <= end; i++) next.add(tracks.value[i].id)
    selected.value = next
    return
  }
  if (event.metaKey || event.ctrlKey) {
    const next = new Set(selected.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    selected.value = next
    anchor.value = index
    return
  }
  selected.value = new Set([id])
  anchor.value = index
}

async function removeSelected() {
  if (selected.value.size === 0 || isSmart.value) return
  const res = await fetch(`/api/playlists/${props.playlist.id}/tracks`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ track_ids: Array.from(selected.value) }),
  })
  if (!res.ok) {
    toast.error('Could not remove tracks')
    return
  }
  selected.value = new Set()
  toast.success('Removed from playlist')
  emit('changed')
  load()
}

async function onDrop(index: number) {
  if (isSmart.value || dragId.value === null) return
  const res = await fetch(`/api/playlists/${props.playlist.id}/tracks/order`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ track_id: dragId.value, to_index: index }),
  })
  dragId.value = null
  if (!res.ok) {
    toast.error('Could not reorder')
    return
  }
  load()
}

watch(() => props.playlist.id, () => {
  rules.value = parseRules(props.playlist.rules)
  artistText.value = rules.value.artist_in.join(', ')
  albumText.value = rules.value.album_in.join(', ')
  genreText.value = rules.value.genre_in.join(', ')
  selected.value = new Set()
  load()
}, { immediate: true })

let timer: ReturnType<typeof setTimeout> | null = null
watch(search, () => {
  if (timer) clearTimeout(timer)
  timer = setTimeout(load, 200)
})
</script>

<template>
  <div class="flex min-h-0 flex-1 flex-col gap-4">
    <div class="flex items-center gap-4">
      <div class="grid h-24 w-24 grid-cols-2 overflow-hidden rounded-xl bg-white/5">
        <img v-for="key in artwork.slice(0, 4)" :key="key" :src="artworkUrl(key)" alt="" class="h-12 w-12 object-cover" />
        <div v-if="artwork.length === 0" class="col-span-2 flex h-24 w-24 items-center justify-center">
          <Music class="h-8 w-8 text-textSecondary" />
        </div>
      </div>
      <div>
        <h2 class="text-2xl font-bold">{{ playlist.name }}</h2>
        <p class="text-sm text-textSecondary">{{ total }} tracks · {{ formatDuration(duration) }}</p>
      </div>
    </div>

    <div class="grid gap-3 rounded-xl border border-border bg-surface p-4 md:grid-cols-3">
      <label class="text-xs text-textSecondary">Playback
        <select v-model="playlist.playback_mode" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary">
          <option value="order">In order</option>
          <option value="shuffle">Shuffle</option>
          <option value="weighted">Weighted rotation</option>
        </select>
      </label>
      <label class="text-xs text-textSecondary">Artist separation
        <input v-model.number="playlist.artist_separation" type="number" min="0" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
      </label>
      <label class="text-xs text-textSecondary">Track repeat hours
        <input v-model.number="playlist.track_repeat_hours" type="number" min="0" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
      </label>
      <template v-if="isSmart">
        <label class="text-xs text-textSecondary md:col-span-3">Artists (comma separated)
          <input v-model="artistText" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
        </label>
        <label class="text-xs text-textSecondary">Albums
          <input v-model="albumText" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
        </label>
        <label class="text-xs text-textSecondary">Genres
          <input v-model="genreText" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
        </label>
        <label class="text-xs text-textSecondary">Added within days
          <input v-model.number="rules.added_within_days" type="number" min="0" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
        </label>
        <label class="text-xs text-textSecondary">Year from
          <input v-model.number="rules.year_min" type="number" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
        </label>
        <label class="text-xs text-textSecondary">Year to
          <input v-model.number="rules.year_max" type="number" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm" />
        </label>
      </template>
      <div class="md:col-span-3">
        <button type="button" class="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-background" @click="saveSettings">Save playlist</button>
      </div>
    </div>

    <div class="flex items-center gap-2">
      <div class="relative min-w-0 flex-1">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-textSecondary" />
        <input v-model="search" type="search" placeholder="Search in playlist" class="w-full rounded-xl border border-border bg-surface py-2 pl-10 pr-3 outline-none focus:border-accent/60" />
      </div>
      <button v-if="!isSmart && selected.size" type="button" class="inline-flex items-center gap-2 rounded-xl border border-red-400/40 px-3 py-2 text-sm text-red-300" @click="removeSelected">
        <Trash2 class="h-4 w-4" /> Remove {{ selected.size }}
      </button>
    </div>

    <div class="min-h-0 flex-1 space-y-1 overflow-y-auto">
      <p v-if="!loading && tracks.length === 0" class="py-10 text-center text-sm text-textSecondary">This playlist is empty.</p>
      <div
        v-for="(track, index) in tracks"
        :key="track.id"
        class="flex items-center gap-3 rounded-lg px-2 py-2"
        :class="selected.has(track.id) ? 'bg-accent/10' : 'hover:bg-white/5'"
        :draggable="!isSmart"
        @click="onClick(index, $event)"
        @dragstart="dragId = track.id"
        @dragover.prevent
        @drop="onDrop(index)"
      >
        <GripVertical v-if="!isSmart" class="h-4 w-4 shrink-0 text-textSecondary" />
        <img v-if="track.artwork_s3_key" :src="artworkUrl(track.artwork_s3_key)" alt="" class="h-10 w-10 rounded object-cover" />
        <span v-else class="flex h-10 w-10 items-center justify-center rounded bg-white/5"><Music class="h-4 w-4 text-textSecondary" /></span>
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium">{{ track.title }}</p>
          <p class="truncate text-xs text-textSecondary">{{ track.artist }} · {{ track.album }}</p>
        </div>
        <span class="text-xs text-textSecondary">{{ formatDuration(track.duration_seconds) }}</span>
      </div>
    </div>
  </div>
</template>
