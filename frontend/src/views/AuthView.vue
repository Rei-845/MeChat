<template>
  <div class="relative min-h-screen w-full overflow-hidden flex items-center justify-center app-bg">
    <!-- Ambient blobs -->
    <div class="blob absolute w-96 h-96 opacity-20"
         style="background:#3390EC;top:-80px;left:-80px;animation-delay:0s" />
    <div class="blob absolute w-80 h-80 opacity-15"
         style="background:#2980DE;bottom:10%;right:-60px;animation-delay:-4s" />
    <div class="blob absolute w-64 h-64 opacity-10"
         style="background:#10B981;bottom:-40px;left:30%;animation-delay:-8s" />

    <!-- Card -->
    <div class="relative z-10 w-full max-w-md mx-4 animate-scale-in">
      <!-- Logo -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-14 h-14 rounded-2xl mb-4"
             style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff;box-shadow:0 0 30px rgba(51,144,236,0.4)">
          <MessageSquare :size="28" class="text-white" />
        </div>
        <h1 class="text-2xl font-bold text-ink">MeChat</h1>
        <p class="text-sm text-ink/40 mt-1">高雅人士都在用的社交平台</p>
      </div>

      <div class="glass-strong rounded-2xl p-8">
        <!-- Tabs -->
        <div class="flex gap-1 p-1 rounded-md mb-8" style="background:rgb(var(--ink) / 0.04)">
          <button
            v-for="tab in ['login','register']"
            :key="tab"
            class="flex-1 py-2 text-sm font-semibold rounded transition-all"
            :style="activeTab === tab
              ? 'background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff;color:#fff;box-shadow:0 4px 12px rgba(51,144,236,0.3)'
              : 'color:rgb(var(--ink) / 0.4)'"
            @click="activeTab = tab"
          >
            {{ tab === 'login' ? '登录' : '注册' }}
          </button>
        </div>

        <!-- Login Form -->
        <form v-if="activeTab === 'login'" @submit.prevent="handleLogin" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-ink/50 mb-2">邮箱</label>
            <input v-model="form.email" type="email" placeholder="your@email.com"
                   class="mc-input" required />
          </div>

          <!-- 密码登录 -->
          <div v-if="loginMode === 'password'">
            <label class="block text-xs font-medium text-ink/50 mb-2">密码</label>
            <div class="relative">
              <input v-model="form.password" :type="showPwd ? 'text' : 'password'"
                     placeholder="输入密码" class="mc-input pr-12" required />
              <button type="button" class="absolute right-3 top-1/2 -translate-y-1/2 text-ink/30 hover:text-ink/60"
                      @click="showPwd = !showPwd">
                <Eye v-if="!showPwd" :size="16" />
                <EyeOff v-else :size="16" />
              </button>
            </div>
          </div>

          <!-- 验证码登录 -->
          <div v-else>
            <label class="block text-xs font-medium text-ink/50 mb-2">验证码</label>
            <div class="flex gap-2">
              <input v-model="form.code" type="text" placeholder="6位验证码" maxlength="6"
                     class="mc-input flex-1" required />
              <button type="button" :disabled="codeCooldown > 0"
                      class="px-4 py-3 rounded-md text-sm font-medium whitespace-nowrap transition-all"
                      :style="codeCooldown > 0
                        ? 'background:rgb(var(--ink) / 0.04);color:rgb(var(--ink) / 0.3);cursor:not-allowed'
                        : 'background:rgba(51,144,236,0.15);color:#3390EC;border:1px solid rgba(51,144,236,0.3)'"
                      @click="sendVerifyCode('login')">
                {{ codeCooldown > 0 ? `${codeCooldown}s` : '发送' }}
              </button>
            </div>
          </div>

          <button type="submit" class="btn-primary w-full mt-2" :disabled="loading">
            <span v-if="loading" class="flex items-center justify-center gap-2">
              <Loader2 :size="16" class="animate-spin" /> 登录中...
            </span>
            <span v-else>登录</span>
          </button>

          <p class="text-center text-xs text-ink/30">
            <button type="button" class="text-primary-light hover:underline"
                    @click="loginMode = loginMode === 'password' ? 'code' : 'password'; form.password = ''; form.code = ''">
              {{ loginMode === 'password' ? '改用验证码登录' : '改用密码登录' }}
            </button>
          </p>
        </form>

        <!-- Register Form -->
        <form v-else @submit.prevent="handleRegister" class="space-y-4">
          <div>
            <label class="block text-xs font-medium text-ink/50 mb-2">邮箱</label>
            <input v-model="form.email" type="email" placeholder="your@email.com"
                   class="mc-input" required />
          </div>
          <div>
            <label class="block text-xs font-medium text-ink/50 mb-2">昵称</label>
            <input v-model="form.nickname" type="text" placeholder="你的昵称（2-20字符）"
                   class="mc-input" required />
          </div>
          <div>
            <label class="block text-xs font-medium text-ink/50 mb-2">密码</label>
            <div class="relative">
              <input v-model="form.password" :type="showPwd ? 'text' : 'password'"
                     placeholder="至少6位" class="mc-input pr-12" required />
              <button type="button" class="absolute right-3 top-1/2 -translate-y-1/2 text-ink/30 hover:text-ink/60"
                      @click="showPwd = !showPwd">
                <Eye v-if="!showPwd" :size="16" />
                <EyeOff v-else :size="16" />
              </button>
            </div>
          </div>
          <div>
            <label class="block text-xs font-medium text-ink/50 mb-2">验证码</label>
            <div class="flex gap-2">
              <input v-model="form.code" type="text" placeholder="6位验证码" maxlength="6"
                     class="mc-input flex-1" required />
              <button type="button" :disabled="codeCooldown > 0"
                      class="px-4 py-3 rounded-md text-sm font-medium whitespace-nowrap transition-all"
                      :style="codeCooldown > 0
                        ? 'background:rgb(var(--ink) / 0.04);color:rgb(var(--ink) / 0.3);cursor:not-allowed'
                        : 'background:rgba(51,144,236,0.15);color:#3390EC;border:1px solid rgba(51,144,236,0.3)'"
                      @click="sendVerifyCode('register')">
                {{ codeCooldown > 0 ? `${codeCooldown}s` : '发送' }}
              </button>
            </div>
          </div>
          <button type="submit" class="btn-primary w-full mt-2" :disabled="loading">
            <span v-if="loading" class="flex items-center justify-center gap-2">
              <Loader2 :size="16" class="animate-spin" /> 注册中...
            </span>
            <span v-else>创建账号</span>
          </button>
        </form>

        <!-- Error -->
        <p v-if="error" class="mt-4 text-sm text-danger text-center animate-fade-in">{{ error }}</p>
      </div>

      <p class="text-center text-xs text-ink/20 mt-6">
        使用即同意服务条款和隐私政策
      </p>
    </div>
  </div>
</template>

<script setup>
import { ref, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { MessageSquare, Loader2, Eye, EyeOff } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useWsStore }   from '@/stores/ws'
import { sendCode }     from '@/api/auth'
import { useToast }     from '@/composables/useToast'

const router  = useRouter()
const auth    = useAuthStore()
const ws      = useWsStore()
const toast   = useToast()

const activeTab  = ref('login')
const loginMode  = ref('password')   // 'password' | 'code'
const loading    = ref(false)
const error      = ref('')
const showPwd    = ref(false)
const codeCooldown = ref(0)
let cooldownTimer  = null

const form = ref({ email: '', code: '', nickname: '', password: '' })

async function sendVerifyCode(purpose) {
  if (!form.value.email) { error.value = '请先输入邮箱'; return }
  try {
    await sendCode(form.value.email, purpose)
    toast.success('验证码已发送，请查收邮件')
    codeCooldown.value = 60
    cooldownTimer = setInterval(() => {
      codeCooldown.value--
      if (codeCooldown.value <= 0) clearInterval(cooldownTimer)
    }, 1000)
  } catch (e) {
    error.value = typeof e === 'string' ? e : '发送失败，请稍后重试'
  }
}

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    const payload = loginMode.value === 'password'
      ? { email: form.value.email, password: form.value.password }
      : { email: form.value.email, code: form.value.code }
    await auth.loginAction(payload)
    ws.connect(auth.token)
    router.push('/')
  } catch (e) {
    error.value = typeof e === 'string' ? e : '登录失败，请检查邮箱和密码'
  } finally {
    loading.value = false
  }
}

async function handleRegister() {
  error.value = ''
  loading.value = true
  try {
    await auth.registerAction(form.value)
    ws.connect(auth.token)
    router.push('/')
    toast.success('欢迎加入 MeChat！')
  } catch (e) {
    error.value = typeof e === 'string' ? e : '注册失败，请重试'
  } finally {
    loading.value = false
  }
}

onUnmounted(() => clearInterval(cooldownTimer))
</script>
