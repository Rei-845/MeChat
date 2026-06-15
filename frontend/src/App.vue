<template>
  <RouterView />
  <!-- Global user profile popup -->
  <UserProfileModal />
  <!-- Global add-friend (greeting) popup -->
  <AddFriendModal />
  <!-- Global XP gain notification -->
  <XPNotification />
  <!-- Global Toast -->
  <div class="toast-container">
    <TransitionGroup name="toast">
      <div
        v-for="t in toasts"
        :key="t.id"
        class="flex items-center gap-3 px-4 py-3 rounded-md text-sm font-medium
               animate-slide-up shadow-glass"
        :class="{
          'bg-accent/20 border border-accent/30 text-accent': t.type === 'success',
          'bg-danger/20 border border-danger/30 text-danger': t.type === 'error',
          'bg-primary/20 border border-primary/30 text-primary-light': t.type === 'info',
        }"
      >
        <CheckCircle v-if="t.type === 'success'" :size="16" />
        <XCircle     v-else-if="t.type === 'error'"   :size="16" />
        <Info        v-else                            :size="16" />
        {{ t.message }}
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { RouterView } from 'vue-router'
import { CheckCircle, XCircle, Info } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'
import UserProfileModal from '@/components/user/UserProfileModal.vue'
import AddFriendModal   from '@/components/friend/AddFriendModal.vue'
import XPNotification   from '@/components/ui/XPNotification.vue'

const { toasts } = useToast()
</script>

<style>
.toast-enter-active { transition: all 0.25s cubic-bezier(0.16,1,0.3,1); }
.toast-leave-active { transition: all 0.15s ease-in; }
.toast-enter-from  { opacity: 0; transform: translateX(20px); }
.toast-leave-to    { opacity: 0; transform: translateX(20px); }
</style>
