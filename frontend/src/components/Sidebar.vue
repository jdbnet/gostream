<script setup lang="ts">
import { watch } from 'vue'
import { useRoute } from 'vue-router'
import { Radio, Music, Mic2, ListMusic, Settings, Calendar, X } from 'lucide-vue-next'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const route = useRoute()

const navItems = [
  { name: 'Now Playing', path: '/', icon: Radio },
  { name: 'Timetable', path: '/timetable', icon: Calendar },
  { name: 'Tracks', path: '/tracks', icon: Music },
  { name: 'Jingles', path: '/jingles', icon: Mic2 },
  { name: 'Playlists', path: '/playlists', icon: ListMusic },
  { name: 'Settings', path: '/settings', icon: Settings },
]

const isActive = (path: string) => route.path === path

watch(() => route.path, () => {
  emit('close')
})

const onNavigate = () => {
  if (window.matchMedia('(max-width: 1023px)').matches) {
    emit('close')
  }
}
</script>

<template>
  <aside
    class="fixed inset-y-0 left-0 z-50 flex h-full w-64 shrink-0 flex-col border-r border-border bg-surface transition-transform duration-300 ease-in-out lg:static lg:translate-x-0"
    :class="open ? 'translate-x-0' : '-translate-x-full'"
  >
    <div class="flex h-16 items-center justify-between border-b border-border px-6">
      <div class="flex items-center">
        <div class="mr-3 flex h-8 w-8 items-center justify-center">
          <img src="/gostream.png" alt="GoStream" class="h-8 w-8" />
        </div>
        <h1 class="text-xl font-bold tracking-wide">GoStream</h1>
      </div>
      <button
        type="button"
        class="rounded-lg p-2 text-textSecondary transition-colors hover:bg-white/5 hover:text-textPrimary lg:hidden"
        aria-label="Close menu"
        @click="emit('close')"
      >
        <X class="h-5 w-5" />
      </button>
    </div>

    <nav class="flex-1 space-y-2 overflow-y-auto px-4 py-6">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="group flex items-center rounded-lg px-4 py-3 transition-all duration-200"
        :class="[
          isActive(item.path)
            ? 'bg-accent/10 font-medium text-accent'
            : 'text-textSecondary hover:bg-white/5 hover:text-textPrimary'
        ]"
        @click="onNavigate"
      >
        <component
          :is="item.icon"
          class="mr-3 h-5 w-5 transition-colors"
          :class="isActive(item.path) ? 'text-accent' : 'text-textSecondary group-hover:text-textPrimary'"
        />
        {{ item.name }}
      </router-link>
    </nav>
  </aside>
</template>
