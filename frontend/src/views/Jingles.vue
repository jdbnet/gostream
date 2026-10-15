<script setup lang="ts">
import { onUnmounted, ref, onMounted } from 'vue'
import { Music, Pause, Pencil, Play, Save, Trash2, X } from '@lucide/vue'
import TrackUploader from '../components/TrackUploader.vue'

interface Jingle {
  id: number
  name: string
  uploaded_at: string
}

const jingles = ref<Jingle[]>([])
const editingJingleId = ref<number | null>(null)
const editForm = ref({ name: '' })
const playingId = ref<number | null>(null)
const audio = new Audio()

const fetchJingles = async () => {
  const res = await fetch('/api/jingles')
  if (res.ok) {
    jingles.value = await res.json()
  }
}

const deleteJingle = async (id: number) => {
  if (!confirm('Are you sure you want to delete this jingle?')) return
  if (playingId.value === id) stopPreview()
  await fetch(`/api/jingles/${id}`, { method: 'DELETE' })
  fetchJingles()
}

const startEdit = (jingle: Jingle) => {
  editingJingleId.value = jingle.id
  editForm.value = { name: jingle.name }
}

const cancelEdit = () => {
  editingJingleId.value = null
}

const saveJingle = async (id: number) => {
  const res = await fetch(`/api/jingles/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(editForm.value),
  })
  if (res.ok) {
    editingJingleId.value = null
    fetchJingles()
  } else {
    alert('Failed to update jingle')
  }
}

const stopPreview = () => {
  audio.pause()
  audio.removeAttribute('src')
  playingId.value = null
}

const togglePreview = (id: number) => {
  if (playingId.value === id) {
    stopPreview()
    return
  }
  audio.src = `/api/jingles/${id}/audio`
  playingId.value = id
  void audio.play()
}

audio.addEventListener('ended', stopPreview)

onMounted(fetchJingles)
onUnmounted(stopPreview)
</script>

<template>
  <div class="flex h-[calc(100vh-7.5rem)] flex-col gap-4 lg:h-[calc(100vh-4rem)]">
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold">Jingles</h1>
    </div>

    <TrackUploader
      endpoint="/api/jingles"
      label="Upload Jingle"
      @uploaded="fetchJingles"
    />

    <div class="flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl border border-border bg-surface">
      <div class="grid grid-cols-[2.5rem_minmax(0,1fr)_8rem_7.5rem] gap-2 border-b border-border px-3 py-2 text-xs uppercase tracking-wide text-textSecondary">
        <span></span>
        <span>Name</span>
        <span class="hidden sm:block">Uploaded</span>
        <span class="text-right">Actions</span>
      </div>
      <div v-if="jingles.length === 0" class="px-6 py-16 text-center text-textSecondary">No jingles uploaded yet.</div>
      <div v-else class="min-h-0 flex-1 overflow-auto">
        <div
          v-for="jingle in jingles"
          :key="jingle.id"
          class="grid grid-cols-[2.5rem_minmax(0,1fr)_auto] items-center gap-2 border-b border-border/70 px-3 py-2 sm:grid-cols-[2.5rem_minmax(0,1fr)_8rem_7.5rem]"
        >
          <span class="flex h-9 w-9 items-center justify-center rounded bg-white/5">
            <Music class="h-4 w-4 text-textSecondary" />
          </span>
          <template v-if="editingJingleId === jingle.id">
            <input v-model="editForm.name" type="text" class="min-w-0 rounded border border-border bg-background px-3 py-1.5 text-sm text-textPrimary outline-none focus:border-accent/50" />
            <span class="hidden text-sm text-textSecondary sm:block">{{ new Date(jingle.uploaded_at).toLocaleDateString() }}</span>
            <div class="flex items-center justify-end gap-1">
              <button type="button" class="rounded-lg p-2 text-accent hover:bg-accent/10" title="Save" @click="saveJingle(jingle.id)">
                <Save class="h-4 w-4" />
              </button>
              <button type="button" class="rounded-lg p-2 text-textSecondary hover:bg-white/10 hover:text-textPrimary" title="Cancel" @click="cancelEdit">
                <X class="h-4 w-4" />
              </button>
            </div>
          </template>
          <template v-else>
            <p class="min-w-0 truncate text-sm font-medium">{{ jingle.name }}</p>
            <p class="hidden truncate text-sm text-textSecondary sm:block">{{ new Date(jingle.uploaded_at).toLocaleDateString() }}</p>
            <div class="flex items-center justify-end gap-1">
              <button type="button" class="rounded-lg p-2 text-textSecondary hover:bg-white/10 hover:text-textPrimary" :title="playingId === jingle.id ? 'Stop' : 'Preview'" @click="togglePreview(jingle.id)">
                <Pause v-if="playingId === jingle.id" class="h-4 w-4" />
                <Play v-else class="h-4 w-4" />
              </button>
              <button type="button" class="rounded-lg p-2 text-accent hover:bg-accent/10" title="Edit" @click="startEdit(jingle)">
                <Pencil class="h-4 w-4" />
              </button>
              <button type="button" class="rounded-lg p-2 text-red-400 hover:bg-red-400/10" title="Delete" @click="deleteJingle(jingle.id)">
                <Trash2 class="h-4 w-4" />
              </button>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>
