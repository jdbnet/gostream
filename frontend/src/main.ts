import { createApp } from 'vue'
import './style.css'
import '@fontsource/space-grotesk'
import '@fontsource/dm-mono'
import App from './App.vue'
import router from './router'

createApp(App).use(router).mount('#app')
