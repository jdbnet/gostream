import { createRouter, createWebHistory } from 'vue-router'
import NowPlaying from '../views/NowPlaying.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'NowPlaying',
      component: NowPlaying
    },
    {
      path: '/tracks',
      name: 'Tracks',
      component: () => import('../views/Tracks.vue')
    },
    {
      path: '/jingles',
      name: 'Jingles',
      component: () => import('../views/Jingles.vue')
    },
    {
      path: '/playlists',
      name: 'Playlists',
      component: () => import('../views/Playlists.vue')
    },
    {
      path: '/settings',
      name: 'Settings',
      component: () => import('../views/Settings.vue')
    },
    {
      path: '/timetable',
      name: 'Timetable',
      component: () => import('../views/Timetable.vue')
    }
  ]
})

export default router
