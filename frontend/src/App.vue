<script setup lang="ts">
import Sidebar from './components/Sidebar.vue'
</script>

<template>
  <div class="flex h-screen overflow-hidden bg-background text-textPrimary">
    <Sidebar />
    <main class="flex-1 overflow-y-auto p-8">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
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
