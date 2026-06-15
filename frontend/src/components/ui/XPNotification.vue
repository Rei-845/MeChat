<template>
  <Teleport to="body">
    <TransitionGroup name="xp-pop" tag="div" class="xp-container">
      <div v-for="item in items" :key="item.id" class="xp-pill">
        <span class="xp-label">经验</span><span class="xp-plus">+</span>{{ item.amount }}
      </div>
    </TransitionGroup>
  </Teleport>
</template>

<script setup>
import { useXPNotify } from '@/composables/useXPNotify'
const { items } = useXPNotify()
</script>

<style scoped>
.xp-container {
  position: fixed;
  top: 38%;
  left: 50%;
  transform: translateX(-50%);
  z-index: 99999;
  pointer-events: none;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.xp-pill {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 8px 20px;
  border-radius: 999px;
  font-size: 20px;
  font-weight: 800;
  color: #EAB308;
  background: rgba(234, 179, 8, 0.15);
  border: 1.5px solid rgba(234, 179, 8, 0.4);
  backdrop-filter: blur(8px);
  box-shadow: 0 4px 24px rgba(234, 179, 8, 0.25);
  white-space: nowrap;
}

.xp-label { font-size: 15px; font-weight: 700; margin-right: 2px; color: #FCD34D; }
.xp-plus  { font-size: 20px; color: #FBBF24; margin-right: 1px; }

/* Enter: scale up + fade in */
.xp-pop-enter-active {
  animation: xp-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
}
/* Leave: float up + fade out */
.xp-pop-leave-active {
  animation: xp-out 1.1s ease-in forwards;
}

@keyframes xp-in {
  from { opacity: 0; transform: scale(0.5) translateY(10px); }
  to   { opacity: 1; transform: scale(1)   translateY(0);    }
}

@keyframes xp-out {
  0%   { opacity: 1; transform: translateY(0);    }
  60%  { opacity: 0.8; }
  100% { opacity: 0; transform: translateY(-48px); }
}
</style>
