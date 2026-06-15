import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { vDoubleTap } from './directives/doubleTap'
import './style.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.directive('double-tap', vDoubleTap)
app.mount('#app')
