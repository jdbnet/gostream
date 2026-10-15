<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Calendar, Library, ListMusic, Mic2, Radio, Settings, X } from '@lucide/vue'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const route = useRoute()

const navItems = [
  { name: 'Now Playing', path: '/', icon: Radio },
  { name: 'Library', path: '/library', icon: Library },
  { name: 'Playlists', path: '/playlists', icon: ListMusic },
  { name: 'Jingles', path: '/jingles', icon: Mic2 },
  { name: 'Timetable', path: '/timetable', icon: Calendar },
  { name: 'Settings', path: '/settings', icon: Settings },
]

const isActive = (path: string) => {
  if (path === '/') return route.path === '/'
  return route.path === path || route.path.startsWith(`${path}/`)
}

const libraryTabs = computed(() => route.path.startsWith('/library'))

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
    <div class="flex h-16 items-center justify-between border-b border-border px-5">
      <div class="flex items-center gap-3">
        <img src="/gostream.png" alt="" class="h-8 w-8" />
        <h1 class="text-xl font-bold tracking-wide">GoStream</h1>
      </div>
      <button
        type="button"
        class="rounded-lg p-2 text-textSecondary hover:bg-white/5 hover:text-textPrimary lg:hidden"
        aria-label="Close menu"
        @click="emit('close')"
      >
        <X class="h-5 w-5" />
      </button>
    </div>

    <nav class="flex-1 space-y-1 overflow-y-auto px-3 py-4">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="flex items-center rounded-lg px-3 py-2.5 text-sm transition-colors"
        :class="isActive(item.path) ? 'bg-accent/15 font-medium text-accent' : 'text-textSecondary hover:bg-white/5 hover:text-textPrimary'"
        @click="onNavigate"
      >
        <component :is="item.icon" class="mr-3 h-5 w-5" />
        {{ item.name }}
      </router-link>

      <div v-if="libraryTabs" class="ml-4 mt-1 space-y-1 border-l border-border pl-3">
        <router-link to="/library" class="block rounded-md px-3 py-1.5 text-sm text-textSecondary hover:text-textPrimary" exact-active-class="text-accent">Tracks</router-link>
        <router-link to="/library/artists" class="block rounded-md px-3 py-1.5 text-sm text-textSecondary hover:text-textPrimary" active-class="text-accent">Artists</router-link>
        <router-link to="/library/albums" class="block rounded-md px-3 py-1.5 text-sm text-textSecondary hover:text-textPrimary" active-class="text-accent">Albums</router-link>
        <router-link to="/library/genres" class="block rounded-md px-3 py-1.5 text-sm text-textSecondary hover:text-textPrimary" active-class="text-accent">Genres</router-link>
      </div>
    </nav>
  </aside>
</template>
