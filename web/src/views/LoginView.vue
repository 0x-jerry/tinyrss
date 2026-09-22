<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useStore } from '../store'
import { useApiToast } from '../api/useApiToast'
import LoginForm from '../components/shared/LoginForm.vue'

const { auth } = useStore()
const router = useRouter()
const toast = useApiToast()
const checking = ref(true)

onMounted(async () => {
  try {
    const ok = await auth.probe()
    if (ok) router.replace('/')
    else checking.value = false
  } catch (e) {
    toast.fromError(e)
    checking.value = false
  }
})

function onLogin(token: string) {
  auth.login(token)
  router.replace('/')
}
</script>

<template>
  <main class="login">
    <div class="login__card">
      <div class="login__brand"><span aria-hidden="true" class="i-lucide-rss text-[22px]" /> TinyRSS</div>
      <p class="login__hint">
        This instance is protected by an access token. Paste it below to sign in.
      </p>
      <p v-if="checking" class="login__hint login__hint--muted">Checking…</p>
      <LoginForm v-else @submit="onLogin" />
    </div>
  </main>
</template>

<style scoped>
.login {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  background: var(--bg-subtle);
}
.login__card {
  width: 380px;
  padding: 28px;
  border-radius: 12px;
  background: var(--surface);
  box-shadow: 0 8px 30px rgba(20, 30, 50, 0.12);
}
.login__brand {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 18px;
  font-weight: 700;
  color: var(--accent);
  margin-bottom: 14px;
}
.login__hint {
  margin: 0 0 16px;
  color: var(--text-muted);
  font-size: 13.5px;
  line-height: 1.5;
}
.login__hint--muted {
  color: var(--text-faint);
}
@media (max-width: 768px) {
  .login { padding: 16px; }
  .login__card { width: 100%; max-width: 400px; padding: 20px; }
}
</style>
