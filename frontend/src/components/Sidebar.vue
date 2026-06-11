<script setup lang="ts">
import { useRoute } from 'vue-router'
import { Radio, Music, Mic2, ListMusic, Settings, Calendar } from 'lucide-vue-next'

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
</script>

<template>
  <aside class="w-64 bg-surface border-r border-border flex flex-col h-full shrink-0">
    <div class="h-16 flex items-center px-6 border-b border-border">
      <div class="w-8 h-8 rounded bg-accent flex items-center justify-center mr-3 shadow-[0_0_15px_rgba(0,188,212,0.4)]">
        <Radio class="w-5 h-5 text-white" />
      </div>
      <h1 class="text-xl font-bold tracking-wide">GoStream</h1>
    </div>
    
    <nav class="flex-1 py-6 px-4 space-y-2">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="flex items-center px-4 py-3 rounded-lg transition-all duration-200 group"
        :class="[
          isActive(item.path) 
            ? 'bg-accent/10 text-accent font-medium' 
            : 'text-textSecondary hover:bg-white/5 hover:text-textPrimary'
        ]"
      >
        <component 
          :is="item.icon" 
          class="w-5 h-5 mr-3 transition-colors"
          :class="isActive(item.path) ? 'text-accent' : 'text-textSecondary group-hover:text-textPrimary'"
        />
        {{ item.name }}
      </router-link>
    </nav>
    
  </aside>
</template>
