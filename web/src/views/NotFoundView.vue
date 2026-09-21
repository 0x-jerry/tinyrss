<script setup lang="ts">
import { useRouter } from 'vue-router'
import Button from '../components/shared/Button.vue'

const router = useRouter()
</script>

<template>
  <main class="notfound">
    <div class="notfound__card">
      <h1 class="notfound__code">404</h1>
      <p class="notfound__title">Page not found</p>
      <p class="notfound__desc">The page you're looking for doesn't exist.</p>
      <Button variant="primary" @click="router.push('/')">Back to main page</Button>
    </div>
  </main>
</template>

<style scoped>
.notfound {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 24px;
  background: var(--bg-subtle);
}
.notfound__card {
  width: 100%;
  max-width: 420px;
  padding: 32px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--surface);
  text-align: center;
  box-shadow: 0 8px 30px rgba(20, 30, 50, 0.1);
}
.notfound__code {
  margin: 0;
  line-height: 1;
  font-size: 48px;
  font-weight: 700;
  color: var(--accent);
}
.notfound__title {
  margin: 10px 0 8px;
  font-size: 18px;
  font-weight: 650;
  color: var(--text);
}
.notfound__desc {
  margin: 0 0 22px;
  font-size: 13.5px;
  line-height: 1.5;
  color: var(--text-muted);
}
@media (max-width: 768px) {
  .notfound {
    padding: 16px;
  }
  .notfound__card {
    padding: 24px 20px;
  }
}
</style>
