<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Trash2 } from 'lucide-vue-next'
import TrackUploader from '../components/TrackUploader.vue'

const jingles = ref<any[]>([])

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

    <div class="glass rounded-xl overflow-hidden border border-border">
      <table class="w-full text-left">
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
            <td class="px-6 py-4 font-medium text-white">{{ jingle.name }}</td>
            <td class="px-6 py-4 text-textSecondary">{{ new Date(jingle.uploaded_at).toLocaleDateString() }}</td>
            <td class="px-6 py-4 text-right opacity-0 group-hover:opacity-100 transition-opacity">
              <button @click="deleteJingle(jingle.id)" class="text-red-400 hover:text-red-300 p-2 rounded-lg hover:bg-red-400/10 transition-colors">
                <Trash2 class="w-4 h-4" />
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
