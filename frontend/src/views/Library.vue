<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useVirtualizer } from '@tanstack/vue-virtual'
import { ListPlus, Loader2, Music, Search, SlidersHorizontal } from '@lucide/vue'
import { toast } from 'vue-sonner'
import AddToPlaylistDialog from '../components/AddToPlaylistDialog.vue'
import TrackUploader from '../components/TrackUploader.vue'
import { fetchAlbums, fetchArtists, fetchGenres, fetchTracks, requestTrack } from '../lib/api'
import { artworkUrl, formatDuration, type AlbumRow, type ArtistRow, type GenreRow, type Track, type TrackQuery } from '../types'

const route = useRoute()
const router = useRouter()

const mode = computed(() => {
  if (route.path.endsWith('/artists')) return 'artists'
  if (route.path.endsWith('/albums')) return 'albums'
  if (route.path.endsWith('/genres')) return 'genres'
  return 'tracks'
})

const search = ref('')
const showFilters = ref(false)
const yearMin = ref('')
const yearMax = ref('')
const durationMin = ref('')
const durationMax = ref('')
const addedFrom = ref('')
const addedTo = ref('')
const sort = ref('date_added')
const dir = ref<'asc' | 'desc'>('desc')

const tracks = ref<Track[]>([])
const artists = ref<ArtistRow[]>([])
const albums = ref<AlbumRow[]>([])
const genres = ref<GenreRow[]>([])
const total = ref(0)
const loading = ref(false)
const scroller = ref<HTMLElement | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)
const selected = ref<Set<number>>(new Set())
const allMatching = ref(false)
const anchor = ref<number | null>(null)
const cursor = ref(0)
const dialogOpen = ref(false)

const pageSize = 100
let offset = 0
let requestId = 0

const numberOrNull = (value: string) => {
  if (value.trim() === '') return null
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : null
}

const filter = computed<TrackQuery>(() => {
  const query: TrackQuery = {
    q: search.value.trim(),
    artist: typeof route.query.artist === 'string' ? route.query.artist : '',
    album: typeof route.query.album === 'string' ? route.query.album : '',
    genre: typeof route.query.genre === 'string' ? route.query.genre : '',
    year_min: numberOrNull(yearMin.value),
    year_max: numberOrNull(yearMax.value),
    duration_min: numberOrNull(durationMin.value),
    duration_max: numberOrNull(durationMax.value),
    added_from: addedFrom.value,
    added_to: addedTo.value,
    sort: sort.value,
    dir: dir.value,
  }
  return query
})

const chips = computed(() => {
  const items: { key: string; label: string }[] = []
  if (filter.value.artist) items.push({ key: 'artist', label: filter.value.artist })
  if (filter.value.album) items.push({ key: 'album', label: filter.value.album })
  if (filter.value.genre) items.push({ key: 'genre', label: filter.value.genre })
  if (yearMin.value || yearMax.value) items.push({ key: 'year', label: `${yearMin.value || '…'}–${yearMax.value || '…'}` })
  if (durationMin.value || durationMax.value) items.push({ key: 'duration', label: `${durationMin.value || '0'}–${durationMax.value || '∞'}s` })
  if (addedFrom.value || addedTo.value) items.push({ key: 'added', label: 'Date added' })
  return items
})

const selectedCount = computed(() => allMatching.value ? total.value : selected.value.size)
const bulkFilter = computed(() => allMatching.value ? filter.value : null)

const columns = [
  { key: 'title', label: 'Title' },
  { key: 'artist', label: 'Artist' },
  { key: 'album', label: 'Album' },
  { key: 'genre', label: 'Genre' },
  { key: 'year', label: 'Year' },
  { key: 'duration', label: 'Time' },
]

const virtualizer = useVirtualizer(computed(() => ({
  count: mode.value === 'tracks' ? tracks.value.length : 0,
  getScrollElement: () => scroller.value,
  estimateSize: () => 56,
  overscan: 12,
})))

const virtualRows = computed(() => virtualizer.value.getVirtualItems())

async function reload() {
  offset = 0
  tracks.value = []
  artists.value = []
  albums.value = []
  genres.value = []
  selected.value = new Set()
  allMatching.value = false
  anchor.value = null
  await loadMore()
}

async function loadMore() {
  if (loading.value) return
  if (offset > 0 && offset >= total.value) return
  const current = ++requestId
  loading.value = true
  try {
    if (mode.value === 'artists') {
      const data = await fetchArtists(search.value.trim(), offset)
      if (current !== requestId) return
      artists.value = offset === 0 ? data.artists : artists.value.concat(data.artists)
      total.value = data.total
      offset += data.artists.length
    } else if (mode.value === 'albums') {
      const data = await fetchAlbums(search.value.trim(), offset)
      if (current !== requestId) return
      albums.value = offset === 0 ? data.albums : albums.value.concat(data.albums)
      total.value = data.total
      offset += data.albums.length
    } else if (mode.value === 'genres') {
      const data = await fetchGenres(search.value.trim(), offset)
      if (current !== requestId) return
      genres.value = offset === 0 ? data.genres : genres.value.concat(data.genres)
      total.value = data.total
      offset += data.genres.length
    } else {
      const data = await fetchTracks({ ...filter.value, limit: pageSize, offset })
      if (current !== requestId) return
      tracks.value = offset === 0 ? (data.tracks || []) : tracks.value.concat(data.tracks || [])
      total.value = data.total || 0
      offset += (data.tracks || []).length
    }
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not load the library')
  } finally {
    if (current === requestId) loading.value = false
  }
}

function clearChip(key: string) {
  const query = { ...route.query }
  if (key === 'artist' || key === 'album' || key === 'genre') delete query[key]
  if (key === 'year') {
    yearMin.value = ''
    yearMax.value = ''
  }
  if (key === 'duration') {
    durationMin.value = ''
    durationMax.value = ''
  }
  if (key === 'added') {
    addedFrom.value = ''
    addedTo.value = ''
  }
  router.replace({ path: '/library', query })
}

function toggleSort(key: string) {
  if (sort.value === key) {
    dir.value = dir.value === 'asc' ? 'desc' : 'asc'
  } else {
    sort.value = key
    dir.value = key === 'date_added' ? 'desc' : 'asc'
  }
}

function onRowClick(index: number, event: MouseEvent) {
  const id = tracks.value[index]?.id
  if (!id) return
  cursor.value = index
  if (event.shiftKey && anchor.value !== null) {
    const start = Math.min(anchor.value, index)
    const end = Math.max(anchor.value, index)
    const next = new Set(selected.value)
    for (let i = start; i <= end; i++) {
      const row = tracks.value[i]
      if (row) next.add(row.id)
    }
    selected.value = next
    allMatching.value = false
    return
  }
  if (event.metaKey || event.ctrlKey) {
    const next = new Set(selected.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    selected.value = next
    anchor.value = index
    allMatching.value = false
    return
  }
  selected.value = new Set([id])
  anchor.value = index
  allMatching.value = false
}

function rowSelected(id: number) {
  return allMatching.value || selected.value.has(id)
}

async function playNext(id: number) {
  try {
    await requestTrack(id)
    toast.success('Track will play next')
  } catch (error) {
    toast.error(error instanceof Error ? error.message : 'Could not queue track')
  }
}

function onScroll() {
  const last = virtualRows.value[virtualRows.value.length - 1]
  if (last && last.index >= tracks.value.length - 12) loadMore()
}

function onBrowseScroll(event: Event) {
  const el = event.target as HTMLElement
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 80) loadMore()
}

function onKey(event: KeyboardEvent) {
  if (!route.path.startsWith('/library') || mode.value !== 'tracks') return
  const tag = (event.target as HTMLElement | null)?.tagName
  const typing = tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT'
  if (event.key === '/' && !typing) {
    event.preventDefault()
    searchInput.value?.focus()
    return
  }
  if (typing) return
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    cursor.value = Math.min(tracks.value.length - 1, cursor.value + 1)
    virtualizer.value.scrollToIndex(cursor.value)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    cursor.value = Math.max(0, cursor.value - 1)
    virtualizer.value.scrollToIndex(cursor.value)
  } else if (event.key === ' ') {
    event.preventDefault()
    const id = tracks.value[cursor.value]?.id
    if (!id) return
    const next = new Set(selected.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    selected.value = next
    allMatching.value = false
    anchor.value = cursor.value
  } else if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'a') {
    event.preventDefault()
    allMatching.value = true
  }
}

let searchTimer: ReturnType<typeof setTimeout> | null = null
function scheduleReload() {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(reload, 200)
}

watch([filter, mode], scheduleReload)

onMounted(() => {
  window.addEventListener('keydown', onKey)
  reload()
})
onUnmounted(() => window.removeEventListener('keydown', onKey))
</script>

<template>
  <div class="flex h-[calc(100vh-7.5rem)] flex-col gap-4 lg:h-[calc(100vh-4rem)]">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-2xl font-bold sm:text-3xl">Library</h1>
        <p class="text-sm text-textSecondary">{{ total.toLocaleString() }} {{ mode === 'tracks' ? 'tracks' : mode }}</p>
      </div>
      <div class="flex flex-wrap items-center gap-2">
        <button
          v-if="mode === 'tracks' && total > 0 && (chips.length > 0 || search)"
          type="button"
          class="rounded-xl border border-border px-4 py-2 text-sm hover:bg-white/5"
          @click="allMatching = true; dialogOpen = true"
        >
          Add all to playlist
        </button>
        <TrackUploader v-if="mode === 'tracks'" endpoint="/api/tracks" label="Upload" @uploaded="reload" />
      </div>
    </div>

    <div class="flex flex-col gap-3 sm:flex-row">
      <div class="relative min-w-0 flex-1">
        <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-textSecondary" />
        <input
          ref="searchInput"
          v-model="search"
          type="search"
          placeholder="Search title, artist, album, genre"
          class="w-full rounded-xl border border-border bg-surface py-2.5 pl-10 pr-3 outline-none focus:border-accent/60"
        />
      </div>
      <button
        v-if="mode === 'tracks'"
        type="button"
        class="inline-flex items-center justify-center gap-2 rounded-xl border border-border px-4 py-2.5 text-sm hover:bg-white/5"
        @click="showFilters = !showFilters"
      >
        <SlidersHorizontal class="h-4 w-4" />
        Filters
      </button>
    </div>

    <div v-if="chips.length" class="flex flex-wrap gap-2">
      <button
        v-for="chip in chips"
        :key="chip.key"
        type="button"
        class="rounded-full bg-accent/15 px-3 py-1 text-sm text-accent"
        @click="clearChip(chip.key)"
      >
        {{ chip.label }} ×
      </button>
    </div>

    <div v-if="showFilters && mode === 'tracks'" class="grid grid-cols-2 gap-3 rounded-xl border border-border bg-surface p-4 md:grid-cols-3">
      <label class="text-xs text-textSecondary">Year from<input v-model="yearMin" type="number" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary" /></label>
      <label class="text-xs text-textSecondary">Year to<input v-model="yearMax" type="number" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary" /></label>
      <label class="text-xs text-textSecondary">Min seconds<input v-model="durationMin" type="number" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary" /></label>
      <label class="text-xs text-textSecondary">Max seconds<input v-model="durationMax" type="number" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary" /></label>
      <label class="text-xs text-textSecondary">Added from<input v-model="addedFrom" type="date" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary" /></label>
      <label class="text-xs text-textSecondary">Added to<input v-model="addedTo" type="date" class="mt-1 w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-textPrimary" /></label>
    </div>

    <div v-if="mode === 'tracks' && selectedCount > 0" class="flex flex-wrap items-center gap-3 rounded-xl border border-accent/30 bg-accent/10 px-4 py-3">
      <span class="text-sm">{{ allMatching ? `All ${total.toLocaleString()} matching` : `${selectedCount} selected` }}</span>
      <button type="button" class="text-sm text-accent" @click="allMatching = true">Select all {{ total.toLocaleString() }} matching</button>
      <button type="button" class="ml-auto inline-flex items-center gap-2 rounded-lg bg-accent px-3 py-1.5 text-sm font-medium text-background" @click="dialogOpen = true">
        <ListPlus class="h-4 w-4" />
        Add to playlist
      </button>
    </div>

    <div v-if="mode !== 'tracks'" class="min-h-0 flex-1 overflow-y-auto" @scroll="onBrowseScroll">
      <div v-if="!loading && total === 0" class="rounded-xl border border-dashed border-border px-6 py-16 text-center text-textSecondary">Nothing matches.</div>
      <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <router-link
          v-for="artist in artists"
          :key="artist.name"
          :to="{ path: '/library', query: { artist: artist.name } }"
          class="rounded-xl border border-border bg-surface p-4 hover:border-accent/40"
        >
          <p class="font-medium">{{ artist.name }}</p>
          <p class="text-sm text-textSecondary">{{ artist.track_count }} tracks</p>
        </router-link>
        <router-link
          v-for="album in albums"
          :key="album.album + album.album_artist"
          :to="{ path: '/library', query: { album: album.album } }"
          class="flex gap-3 rounded-xl border border-border bg-surface p-3 hover:border-accent/40"
        >
          <div class="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-white/5">
            <img v-if="album.artwork_s3_key" :src="artworkUrl(album.artwork_s3_key)" alt="" class="h-full w-full object-cover" />
            <Music v-else class="h-6 w-6 text-textSecondary" />
          </div>
          <div class="min-w-0">
            <p class="truncate font-medium">{{ album.album }}</p>
            <p class="truncate text-sm text-textSecondary">{{ album.album_artist }}</p>
            <p class="text-xs text-textSecondary">{{ album.year || '' }} · {{ album.track_count }} tracks</p>
          </div>
        </router-link>
        <router-link
          v-for="genre in genres"
          :key="genre.name"
          :to="{ path: '/library', query: { genre: genre.name } }"
          class="rounded-xl border border-border bg-surface p-4 hover:border-accent/40"
        >
          <p class="font-medium">{{ genre.name }}</p>
          <p class="text-sm text-textSecondary">{{ genre.track_count }} tracks</p>
        </router-link>
      </div>
    </div>

    <div v-else class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-border bg-surface">
      <div class="grid grid-cols-[2.5rem_minmax(0,1fr)] gap-2 border-b border-border px-3 py-2 text-xs uppercase tracking-wide text-textSecondary lg:grid-cols-[2.5rem_minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,0.8fr)_4.5rem_4rem]">
        <span></span>
        <button
          v-for="(column, index) in columns"
          :key="column.key"
          type="button"
          class="min-w-0 truncate text-left hover:text-textPrimary"
          :class="index === 0 ? '' : 'hidden lg:block'"
          @click="toggleSort(column.key)"
        >
          {{ column.label }}
          <span v-if="sort === column.key">{{ dir === 'asc' ? '↑' : '↓' }}</span>
        </button>
      </div>
      <div ref="scroller" class="min-h-0 flex-1 overflow-auto" @scroll="onScroll">
        <div v-if="!loading && tracks.length === 0" class="px-6 py-16 text-center text-textSecondary">No tracks match this library view.</div>
        <div v-else :style="{ height: `${virtualizer.getTotalSize()}px`, position: 'relative' }">
          <div
            v-for="row in virtualRows"
            :key="String(row.key)"
            class="absolute left-0 top-0 grid w-full grid-cols-[2.5rem_minmax(0,1fr)] items-center gap-2 border-b border-border/70 px-3 lg:grid-cols-[2.5rem_minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)_minmax(0,0.8fr)_4.5rem_4rem]"
            :class="[rowSelected(tracks[row.index].id) ? 'bg-accent/10' : 'hover:bg-white/[0.03]', cursor === row.index ? 'outline outline-1 outline-accent/40' : '']"
            :style="{ height: `${row.size}px`, transform: `translateY(${row.start}px)` }"
            @click="onRowClick(row.index, $event)"
          >
            <img v-if="tracks[row.index].artwork_s3_key" :src="artworkUrl(tracks[row.index].artwork_s3_key)" alt="" class="h-9 w-9 rounded object-cover" />
            <span v-else class="flex h-9 w-9 items-center justify-center rounded bg-white/5"><Music class="h-4 w-4 text-textSecondary" /></span>
            <div class="min-w-0 lg:contents">
              <div class="flex min-w-0 items-center gap-2">
                <p class="min-w-0 flex-1 truncate text-sm font-medium">{{ tracks[row.index].title }}</p>
                <button type="button" class="shrink-0 text-xs text-accent" @click.stop="playNext(tracks[row.index].id)">Play next</button>
              </div>
              <p class="min-w-0 truncate text-sm text-textSecondary lg:text-textPrimary">{{ tracks[row.index].artist || 'Unknown artist' }}</p>
              <p class="hidden min-w-0 truncate text-sm text-textSecondary lg:block">{{ tracks[row.index].album }}</p>
              <p class="hidden min-w-0 truncate text-sm text-textSecondary lg:block">{{ tracks[row.index].genre }}</p>
              <p class="hidden text-sm text-textSecondary lg:block">{{ tracks[row.index].year || '' }}</p>
              <p class="hidden text-sm text-textSecondary lg:block">{{ formatDuration(tracks[row.index].duration_seconds) }}</p>
            </div>
          </div>
        </div>
      </div>
      <div v-if="loading" class="flex items-center justify-center gap-2 border-t border-border py-2 text-sm text-textSecondary">
        <Loader2 class="h-4 w-4 animate-spin" /> Loading
      </div>
    </div>

    <AddToPlaylistDialog
      v-model:open="dialogOpen"
      :track-ids="Array.from(selected)"
      :filter="bulkFilter"
      :count="selectedCount"
    />
  </div>
</template>
