<script setup lang="ts">
import { ref } from 'vue'
import Button from './Button.vue'

export interface LoginFormEmits {
  submit: [token: string]
}

const emit = defineEmits<LoginFormEmits>()
const token = ref('')

function onSubmit() {
  const value = token.value.trim()
  if (!value) return
  emit('submit', value)
}
</script>

<template>
  <form class="login-form" @submit.prevent="onSubmit">
    <input
      v-model="token"
      class="login-form__input"
      type="password"
      autocomplete="current-password"
      placeholder="Paste your access token"
      aria-label="Access token"
      autofocus
    />
    <Button type="submit" :disabled="!token.trim()">Sign in</Button>
  </form>
</template>

<style scoped>
.login-form {
  display: flex;
  gap: 8px;
  width: 100%;
}
.login-form__input {
  flex: 1;
  padding: 9px 12px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  font: inherit;
}
.login-form__input:focus {
  outline: 2px solid var(--accent);
  border-color: var(--accent);
}
@media (max-width: 768px) {
  .login-form { flex-direction: column; align-items: stretch; }
}
</style>
