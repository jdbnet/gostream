<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { Plus, Trash2, X } from '@lucide/vue'

const playlists = ref<any[]>([])
const timetable = ref<any[]>([])
const timezone = ref('')
const nowDay = ref(new Date().getDay())
const nowMinute = ref(0)
const nowLineRef = ref<HTMLElement | null>(null)

const days = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday']
const hours = Array.from({length: 24}, (_, i) => i)
const weekdayIndex: Record<string, number> = {
  Sun: 0, Mon: 1, Tue: 2, Wed: 3, Thu: 4, Fri: 5, Sat: 6,
}

let nowTimer: ReturnType<typeof setInterval> | null = null

const setNowLineRef = (el: Element | null) => {
  nowLineRef.value = el as HTMLElement | null
}

const tickNow = () => {
  const date = new Date()
  try {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone.value || undefined,
      weekday: 'short',
      hour: 'numeric',
      minute: 'numeric',
      hourCycle: 'h23',
    }).formatToParts(date)
    const value = (type: Intl.DateTimeFormatPartTypes) =>
      parts.find(p => p.type === type)?.value || '0'
    nowDay.value = weekdayIndex[value('weekday')] ?? date.getDay()
    nowMinute.value = (Number(value('hour')) % 24) * 60 + Number(value('minute'))
  } catch {
    nowDay.value = date.getDay()
    nowMinute.value = date.getHours() * 60 + date.getMinutes()
  }
}

const showModal = ref(false)
const form = ref({
  id: 0,
  playlist_id: 0,
  day_of_week: 0,
  start_hour: 0,
  start_minute: 0,
  end_hour: 1,
  end_minute: 0,
})

const fetchData = async () => {
  const [plRes, ttRes, cfgRes] = await Promise.all([
    fetch('/api/playlists'),
    fetch('/api/timetable'),
    fetch('/api/config')
  ])
  if (plRes.ok) playlists.value = await plRes.json()
  if (ttRes.ok) timetable.value = await ttRes.json()
  if (cfgRes.ok) {
    const cfg = await cfgRes.json()
    timezone.value = cfg.server?.timezone || ''
  }
  tickNow()
}

const getEntriesForDay = (dayIndex: number) => {
  return timetable.value.filter(e => e.day_of_week === dayIndex)
}

const getPlaylistName = (id: number) => {
  return playlists.value.find(p => p.id === id)?.name || 'Unknown'
}

const openAddModal = (day: number, hour: number) => {
  form.value = {
    id: 0,
    playlist_id: playlists.value[0]?.id || 0,
    day_of_week: day,
    start_hour: hour,
    start_minute: 0,
    end_hour: Math.min(hour + 1, 23),
    end_minute: 59,
  }
  showModal.value = true
}

const editEntry = (entry: any) => {
  form.value = {
    id: entry.id,
    playlist_id: entry.playlist_id,
    day_of_week: entry.day_of_week,
    start_hour: Math.floor(entry.start_minute / 60),
    start_minute: entry.start_minute % 60,
    end_hour: Math.floor(entry.end_minute / 60),
    end_minute: entry.end_minute % 60,
  }
  showModal.value = true
}

const saveEntry = async () => {
  const startMin = form.value.start_hour * 60 + form.value.start_minute
  const endMin = form.value.end_hour * 60 + form.value.end_minute
  
  if (endMin <= startMin) {
    alert("End time must be after start time")
    return
  }

  const res = await fetch('/api/timetable', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      id: form.value.id,
      playlist_id: form.value.playlist_id,
      day_of_week: form.value.day_of_week,
      start_minute: startMin,
      end_minute: endMin
    })
  })
  if (res.ok) {
    showModal.value = false
    fetchData()
  } else {
    alert('Failed to save schedule')
  }
}

const deleteEntry = async (id: number) => {
  if (!confirm('Delete this scheduled block?')) return
  const res = await fetch(`/api/timetable/${id}`, { method: 'DELETE' })
  if (res.ok) {
    showModal.value = false
    fetchData()
  }
}

const formatTime = (minutes: number) => {
  const h = Math.floor(minutes / 60).toString().padStart(2, '0')
  const m = (minutes % 60).toString().padStart(2, '0')
  return `${h}:${m}`
}

onMounted(async () => {
  await fetchData()
  nowTimer = setInterval(tickNow, 15000)
  await nextTick()
  nowLineRef.value?.scrollIntoView({ block: 'center', behavior: 'smooth' })
})

onUnmounted(() => {
  if (nowTimer) clearInterval(nowTimer)
})
</script>

<template>
  <div class="max-w-7xl mx-auto h-full flex flex-col space-y-6 pb-12">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between shrink-0">
      <h1 class="text-2xl font-bold sm:text-3xl">Timetable</h1>
      <button @click="openAddModal(0, 12)" class="btn-primary flex items-center px-4 py-2 rounded-lg font-medium text-sm">
        <Plus class="w-4 h-4 mr-2" />
        Add Schedule
      </button>
    </div>

    <div class="glass flex-1 rounded-xl border border-border overflow-auto relative">
      <div class="min-w-[800px]">
        <!-- Header -->
        <div class="flex border-b border-border sticky top-0 bg-surface z-20">
          <div class="w-16 shrink-0 border-r border-border"></div>
          <div
            v-for="(day, dayIdx) in days"
            :key="day"
            class="flex-1 text-center py-3 font-medium border-r border-border last:border-0"
            :class="dayIdx === nowDay ? 'text-accent' : 'text-textSecondary'"
          >
            {{ day }}
          </div>
        </div>

        <!-- Grid -->
        <div class="flex relative bg-white/[0.01]">
          <!-- Time Labels -->
          <div class="w-16 shrink-0 border-r border-border bg-surface relative z-10">
            <div v-for="hour in hours" :key="hour" class="h-[60px] relative">
              <span class="absolute -top-2 right-2 text-xs text-textSecondary text-right bg-surface px-1">
                {{ hour === 0 ? '12 AM' : hour < 12 ? hour + ' AM' : hour === 12 ? '12 PM' : (hour - 12) + ' PM' }}
              </span>
            </div>
          </div>

          <!-- Day Columns -->
          <div
            v-for="(day, dayIdx) in days"
            :key="day"
            class="flex-1 relative border-r border-border last:border-0"
            :class="dayIdx === nowDay ? 'bg-accent/[0.06]' : ''"
          >
            <!-- Hour slots (for clicking) -->
            <div 
              v-for="hour in hours" 
              :key="hour" 
              class="h-[60px] border-b border-border/50 hover:bg-white/[0.02] cursor-pointer transition-colors"
              @click="openAddModal(dayIdx, hour)"
            ></div>

            <!-- Current time -->
            <div
              v-if="dayIdx === nowDay"
              :ref="setNowLineRef"
              class="absolute left-0 right-0 z-30 pointer-events-none"
              :style="{ top: nowMinute + 'px' }"
            >
              <div class="relative flex items-center">
                <div class="absolute -left-1.5 h-3 w-3 rounded-full bg-rose-500 ring-2 ring-background"></div>
                <div class="h-0.5 w-full bg-rose-500"></div>
                <div class="absolute right-1 top-1/2 -translate-y-1/2 rounded bg-background/90 px-1 font-mono text-[10px] font-medium text-rose-400">
                  {{ formatTime(nowMinute) }}
                </div>
              </div>
            </div>

            <!-- Scheduled Entries -->
            <div 
              v-for="entry in getEntriesForDay(dayIdx)" 
              :key="entry.id"
              class="absolute left-1 right-1 rounded-md bg-accent/20 border border-accent/50 hover:bg-accent/30 cursor-pointer overflow-hidden transition-colors shadow-lg group backdrop-blur-sm"
              :style="{ top: entry.start_minute + 'px', height: (entry.end_minute - entry.start_minute) + 'px' }"
              @click.stop="editEntry(entry)"
            >
              <div class="p-2 h-full flex flex-col">
                <div class="text-xs font-semibold text-accent leading-tight truncate">
                  {{ getPlaylistName(entry.playlist_id) }}
                </div>
                <div class="text-[10px] text-white/70 mt-0.5">
                  {{ formatTime(entry.start_minute) }} - {{ formatTime(entry.end_minute) }}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-background/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div class="glass w-full max-w-md rounded-xl border border-border shadow-2xl p-6">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-xl font-bold">{{ form.id ? 'Edit Schedule' : 'Add Schedule' }}</h2>
          <button @click="showModal = false" class="text-textSecondary hover:text-white">
            <X class="w-5 h-5" />
          </button>
        </div>

        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-textSecondary mb-1">Playlist</label>
            <select v-model="form.playlist_id" class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-white outline-none focus:border-accent">
              <option v-for="p in playlists" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-textSecondary mb-1">Day</label>
            <select v-model="form.day_of_week" class="w-full bg-surface border border-border rounded-lg px-4 py-2.5 text-white outline-none focus:border-accent">
              <option v-for="(day, idx) in days" :key="idx" :value="idx">{{ day }}</option>
            </select>
          </div>

          <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label class="block text-sm font-medium text-textSecondary mb-1">Start Time</label>
              <div class="flex space-x-2">
                <input v-model="form.start_hour" type="number" min="0" max="23" class="w-full bg-surface border border-border rounded-lg px-3 py-2 text-white outline-none focus:border-accent text-center" />
                <span class="flex items-center">:</span>
                <input v-model="form.start_minute" type="number" min="0" max="59" class="w-full bg-surface border border-border rounded-lg px-3 py-2 text-white outline-none focus:border-accent text-center" />
              </div>
            </div>
            <div>
              <label class="block text-sm font-medium text-textSecondary mb-1">End Time</label>
              <div class="flex space-x-2">
                <input v-model="form.end_hour" type="number" min="0" max="23" class="w-full bg-surface border border-border rounded-lg px-3 py-2 text-white outline-none focus:border-accent text-center" />
                <span class="flex items-center">:</span>
                <input v-model="form.end_minute" type="number" min="0" max="59" class="w-full bg-surface border border-border rounded-lg px-3 py-2 text-white outline-none focus:border-accent text-center" />
              </div>
            </div>
          </div>
        </div>

        <div class="flex justify-between items-center mt-8">
          <button v-if="form.id" @click="deleteEntry(form.id)" class="text-red-400 hover:text-red-300 p-2 hover:bg-red-400/10 rounded-lg transition-colors flex items-center">
            <Trash2 class="w-4 h-4 mr-2" /> Delete
          </button>
          <div v-else></div>
          <div class="flex space-x-3">
            <button @click="showModal = false" class="px-4 py-2 rounded-lg font-medium text-textSecondary hover:text-white hover:bg-white/5 transition-colors">
              Cancel
            </button>
            <button @click="saveEntry" class="btn-primary px-4 py-2 rounded-lg font-medium text-sm">
              Save Schedule
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
