<script setup lang="ts">
import { ref, watch } from 'vue'
import { Menu } from 'lucide-vue-next'
import Sidebar from './components/Sidebar.vue'

const sidebarOpen = ref(false)

watch(sidebarOpen, (open) => {
  document.body.style.overflow = open ? 'hidden' : ''
})
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-background text-textPrimary">
    <Transition
      enter-active-class="transition-opacity duration-300"
      leave-active-class="transition-opacity duration-300"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <button
        v-if="sidebarOpen"
        type="button"
        class="fixed inset-0 z-40 bg-black/60 lg:hidden"
        aria-label="Close menu"
        @click="sidebarOpen = false"
      />
    </Transition>

    <Sidebar :open="sidebarOpen" @close="sidebarOpen = false" />

    <div class="flex min-w-0 flex-1 flex-col overflow-hidden">
      <header class="flex h-14 shrink-0 items-center gap-3 border-b border-border bg-surface px-4 lg:hidden">
        <button
          type="button"
          class="rounded-lg p-2 text-textSecondary transition-colors hover:bg-white/5 hover:text-textPrimary"
          aria-label="Open menu"
          @click="sidebarOpen = true"
        >
          <Menu class="h-5 w-5" />
        </button>
        <img src="/gostream.png" alt="" class="h-7 w-7" />
        <span class="text-lg font-bold tracking-wide">GoStream</span>
      </header>

      <main class="flex-1 overflow-y-auto overflow-x-hidden p-4 sm:p-6 lg:p-8">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
