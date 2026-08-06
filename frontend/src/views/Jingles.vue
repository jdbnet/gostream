<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Trash2, Edit2, Save, X } from 'lucide-vue-next'
import TrackUploader from '../components/TrackUploader.vue'

const jingles = ref<any[]>([])

const editingJingleId = ref<number | null>(null)
const editForm = ref({ name: '' })

const fetchJingles = async () => {
  const res = await fetch('/api/jingles')
  if (res.ok) {
    jingles.value = await res.json()
  }
}

const deleteJingle = async (id: number) => {
  if (!confirm('Are you sure you want to delete this jingle?')) return
  await fetch(`/api/jingles/${id}`, { method: 'DELETE' })
  fetchJingles()
}

const startEdit = (jingle: any) => {
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
    body: JSON.stringify(editForm.value)
  })
  if (res.ok) {
    editingJingleId.value = null
    fetchJingles()
  } else {
    alert('Failed to update jingle')
  }
}

onMounted(fetchJingles)
</script>

<template>
  <div class="max-w-4xl mx-auto space-y-8">
    <div class="flex items-center justify-between">
      <h1 class="text-3xl font-bold">Jingles</h1>
    </div>

    <TrackUploader 
      endpoint="/api/jingles" 
      label="Upload Jingle" 
      @uploaded="fetchJingles"
    />

    <div class="glass overflow-hidden rounded-xl border border-border">
      <div class="overflow-x-auto">
      <table class="w-full min-w-[480px] text-left">
        <thead class="bg-white/5 border-b border-border text-textSecondary text-sm uppercase tracking-wider">
          <tr>
            <th class="px-6 py-4 font-medium">Name</th>
            <th class="px-6 py-4 font-medium">Uploaded</th>
            <th class="px-6 py-4 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-if="jingles.length === 0">
            <td colspan="3" class="px-6 py-8 text-center text-textSecondary">
              No jingles uploaded yet.
            </td>
          </tr>
          <tr v-for="jingle in jingles" :key="jingle.id" class="hover:bg-white/[0.02] transition-colors group">
            <template v-if="editingJingleId === jingle.id">
              <td class="px-6 py-4">
                <input v-model="editForm.name" type="text" class="w-full bg-surface border border-border rounded px-3 py-1 text-white outline-none focus:border-accent/50" />
              </td>
              <td class="px-6 py-4 text-textSecondary">{{ new Date(jingle.uploaded_at).toLocaleDateString() }}</td>
              <td class="px-6 py-4 text-right">
                <div class="flex items-center justify-end space-x-2 opacity-100">
                  <button @click="saveJingle(jingle.id)" class="text-green-400 hover:text-green-300 p-2 rounded-lg hover:bg-green-400/10 transition-colors" title="Save">
                    <Save class="w-4 h-4" />
                  </button>
                  <button @click="cancelEdit" class="text-textSecondary hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors" title="Cancel">
                    <X class="w-4 h-4" />
                  </button>
                </div>
              </td>
            </template>
            <template v-else>
              <td class="px-6 py-4 font-medium text-white">{{ jingle.name }}</td>
              <td class="px-6 py-4 text-textSecondary">{{ new Date(jingle.uploaded_at).toLocaleDateString() }}</td>
              <td class="px-6 py-4 text-right opacity-100 transition-opacity lg:opacity-0 lg:group-hover:opacity-100">
                <div class="flex items-center justify-end space-x-2">
                  <button @click="startEdit(jingle)" class="text-accent hover:text-accent/80 p-2 rounded-lg hover:bg-accent/10 transition-colors" title="Edit">
                    <Edit2 class="w-4 h-4" />
                  </button>
                  <button @click="deleteJingle(jingle.id)" class="text-red-400 hover:text-red-300 p-2 rounded-lg hover:bg-red-400/10 transition-colors">
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
  </div>
</template>
