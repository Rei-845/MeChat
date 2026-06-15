<template>
  <div class="h-full overflow-y-auto">
    <div class="max-w-2xl mx-auto px-6 py-8">
      <!-- Header -->
      <div class="text-center mb-10">
        <div class="relative inline-block mb-4">
          <div class="blob absolute w-32 h-32 opacity-30 -z-0"
               style="background:#F59E0B;top:-20px;left:-20px;animation-delay:0s" />
          <div class="w-16 h-16 rounded-2xl flex items-center justify-center relative z-10"
               style="background:linear-gradient(135deg,#F59E0B,#FBBF24);box-shadow:0 0 30px rgba(245,158,11,0.4)">
            <Crown :size="28" class="text-yellow-900" />
          </div>
        </div>

        <h2 class="text-2xl font-bold text-ink">MeChat VIP</h2>
        <p class="text-sm text-ink/40 mt-2">解锁 AI Agent、更长对话记忆及更多权益</p>

        <!-- Current status -->
        <div v-if="vipStatus" class="inline-flex items-center gap-3 mt-4 px-5 py-3 rounded-xl"
             :style="vipStatus.is_active
               ? 'background:rgba(245,158,11,0.12);border:1px solid rgba(245,158,11,0.3)'
               : 'background:rgb(var(--ink) / 0.04);border:1px solid rgb(var(--ink) / 0.08)'">
          <Crown :size="16" :class="vipStatus.is_active ? 'text-yellow-400' : 'text-ink/30'" />
          <span class="text-sm" :class="vipStatus.is_active ? 'text-yellow-400 font-semibold' : 'text-ink/40'">
            <template v-if="vipStatus.is_active && vipStatus.is_lifetime">永久 VIP</template>
            <template v-else-if="vipStatus.is_active">VIP 有效至 {{ formatDate(vipStatus.expired_at) }}</template>
            <template v-else>当前为免费版</template>
          </span>
        </div>
      </div>

      <!-- Feature comparison -->
      <div class="glass rounded-2xl p-6 mb-8">
        <h3 class="text-sm font-semibold text-ink/70 mb-4">功能对比</h3>
        <div class="space-y-3">
          <div v-for="f in features" :key="f.name"
               class="flex items-center justify-between py-2"
               style="border-bottom:1px solid rgb(var(--ink) / 0.05)">
            <span class="text-sm text-ink/70">{{ f.name }}</span>
            <div class="flex items-center gap-8">
              <div class="text-center w-24">
                <span class="text-xs text-ink/40">免费版</span>
                <p class="text-sm mt-0.5 text-ink/50">{{ f.free }}</p>
              </div>
              <div class="text-center w-24">
                <span class="text-xs text-yellow-400">VIP</span>
                <p class="text-sm font-semibold mt-0.5 text-yellow-400">{{ f.vip }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Plans: free + 3 paid -->
      <div class="grid grid-cols-2 gap-3 mb-6">
        <!-- Free tier (display only) -->
        <div class="rounded-2xl p-5 col-span-2 sm:col-span-1"
             style="background:rgb(var(--ink) / 0.03);border:1.5px solid rgb(var(--ink) / 0.08)">
          <p v-if="!isActive" class="text-xs text-accent mb-1">当前</p>
          <p v-else class="text-xs text-ink/30 mb-1">基础版</p>
          <div class="mb-2">
            <span class="text-2xl font-bold text-ink/40">免费</span>
          </div>
          <ul class="text-xs text-ink/30 space-y-1">
            <li>· AI 聊天 5次/天</li>
            <li>· 保留 10 条历史</li>
            <li>· 不支持 AI Agent</li>
          </ul>
        </div>

        <!-- Paid plans -->
        <div v-for="plan in plans" :key="plan.plan"
             @click="selectedPlan = plan.plan"
             class="rounded-2xl p-5 cursor-pointer transition-all col-span-2 sm:col-span-1"
             :style="selectedPlan === plan.plan
               ? 'background:rgba(245,158,11,0.12);border:2px solid rgba(245,158,11,0.5);box-shadow:0 0 20px rgba(245,158,11,0.1)'
               : 'background:rgb(var(--ink) / 0.04);border:2px solid rgb(var(--ink) / 0.08)'">
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-semibold text-ink">{{ plan.name }}</span>
            <span v-if="isLifetime && plan.plan === 'lifetime'"
                  class="text-[11px] px-2 py-0.5 rounded-full font-medium"
                  style="background:rgba(16,185,129,0.15);color:#34D399">当前</span>
            <span v-else-if="plan.badge"
                  class="text-[11px] px-2 py-0.5 rounded-full font-medium"
                  style="background:rgba(239,68,68,0.15);color:#FCA5A5">{{ plan.badge }}</span>
          </div>
          <div class="mb-3">
            <span class="text-3xl font-bold" :class="selectedPlan === plan.plan ? 'text-yellow-400' : 'text-ink'">
              ¥{{ plan.price }}
            </span>
            <span class="text-xs text-ink/40 ml-1">/ {{ plan.duration }}</span>
          </div>
          <p class="text-xs text-ink/50">{{ plan.description }}</p>
          <div v-if="selectedPlan === plan.plan" class="mt-3 flex items-center gap-1.5 text-xs text-yellow-400">
            <CheckCircle :size="12" /> 已选择
          </div>
        </div>
      </div>

      <!-- Pay button -->
      <button v-if="!isLifetime" @click="pay" :disabled="!selectedPlan || paying"
              class="w-full py-4 rounded-xl text-base font-bold transition-all"
              :style="selectedPlan
                ? 'background:linear-gradient(135deg,#F59E0B,#FBBF24);color:#78350F;box-shadow:0 4px 20px rgba(245,158,11,0.35)'
                : 'background:rgb(var(--ink) / 0.05);color:rgb(var(--ink) / 0.3);cursor:not-allowed'">
        <Loader2 v-if="paying" :size="18" class="inline animate-spin mr-2" />
        <Crown v-else :size="18" class="inline mr-2" />
        {{ paying ? '支付中…' : (isActive ? '立即续费' : '立即开通') }}
      </button>
      <div v-else class="w-full py-4 rounded-xl text-base font-bold text-center"
           style="background:rgba(245,158,11,0.12);color:#FBBF24;border:1px solid rgba(245,158,11,0.3)">
        <Crown :size="18" class="inline mr-2" /> 您已是永久 VIP
      </div>

      <p class="text-center text-xs text-ink/25 mt-4">
          此为项目Demo，未接入支付SDK
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Crown, CheckCircle, Loader2 } from 'lucide-vue-next'
import * as vipApi from '@/api/vip'
import { useAuthStore } from '@/stores/auth'
import { useToast }     from '@/composables/useToast'
import { formatFullDate as formatDate } from '@/utils/time'

const auth  = useAuthStore()
const toast = useToast()

const plans        = ref([])
const vipStatus    = ref(null)
const selectedPlan = ref('')
const paying       = ref(false)

const isActive   = computed(() => !!vipStatus.value?.is_active)
const isLifetime = computed(() => !!vipStatus.value?.is_lifetime)

const features = [
  { name: 'AI 聊天', free: '✓',      vip: '✓' },
  { name: 'AI 对话记忆',        free: '10 轮',  vip: '30 轮' },
  { name: 'AI 消息润色 / 帖子创作', free: '—',  vip: '✓' },
  { name: 'AI Agent',  free: '—',      vip: '✓ 开放' },
  { name: '经验加成',           free: '—',     vip: '×2 双倍' },
  { name: 'VIP 专属标识',       free: '—',      vip: '✓' },
]

async function loadStatus() {
  try {
    const res = await vipApi.getStatus()
    vipStatus.value = res.data
  } catch {}
}

async function pay() {
  if (!selectedPlan.value || paying.value) return
  paying.value = true
  try {
    const orderRes = await vipApi.createOrder(selectedPlan.value)
    const payRes   = await vipApi.payOrder(orderRes.data.id)
    auth.updateUser({ vip_level: 1, vip_expired_at: payRes.data.expired_at })
    // 以服务端为准重新拉取状态，确保横幅与按钮即时刷新
    await loadStatus()
    toast.success('🎉 VIP 开通成功！')
    selectedPlan.value = ''
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '支付失败，请重试')
  } finally {
    paying.value = false
  }
}

onMounted(async () => {
  const [plansRes] = await Promise.allSettled([vipApi.getPlans()])
  if (plansRes.status === 'fulfilled') plans.value = plansRes.value.data.list || []
  await loadStatus()
})
</script>
