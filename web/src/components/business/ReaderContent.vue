<script setup lang="ts">
import { ref, watch } from 'vue'

export interface ReaderContentProps {
  html: string
  title?: string
  feedTitle?: string
  author?: string
  publishedLabel?: string
  loading?: boolean
}

export interface ReaderContentEmits {
  feedTitleClick: []
}

const props = defineProps<ReaderContentProps>()
const emit = defineEmits<ReaderContentEmits>()

const root = ref<HTMLElement | null>(null)

// The same component instance persists across article switches, so reset the
// scroll position to the top whenever the article content changes.
watch(
  () => props.html,
  () => {
    if (root.value) root.value.scrollTop = 0
  },
)
</script>

<template>
  <div ref="root" class="reader-content">
    <header v-if="title || feedTitle || author || publishedLabel" class="reader-content__head">
      <h1 v-if="title" class="reader-content__title">{{ title }}</h1>
      <div v-if="feedTitle || author || publishedLabel" class="reader-content__meta">
        <button
          v-if="feedTitle"
          type="button"
          class="reader-content__feed"
          title="Reveal feed in sidebar"
          @click="emit('feedTitleClick')"
        >
          {{ feedTitle }}
        </button>
        <span v-if="author"> · {{ author }}</span>
        <span v-if="publishedLabel"> · {{ publishedLabel }}</span>
      </div>
    </header>
    <div class="reader-content__body" v-html="html"></div>
    <div v-if="loading" class="reader-content__loading" aria-live="polite">
      <span aria-hidden="true" class="reader-content__spinner i-lucide-loader-circle" />
    </div>
  </div>
</template>

<style scoped>
.reader-content {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 20px 28px 48px;
  font-size: 15px;
  line-height: 1.65;
  color: var(--text);
  overflow-wrap: break-word;
}
.reader-content__head {
  margin-bottom: 18px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border-subtle);
}
.reader-content__title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  line-height: 1.3;
  color: var(--text);
}
.reader-content__meta {
  margin-top: 8px;
  font-size: 13px;
  color: var(--text-faint);
}
.reader-content__feed {
  padding: 0;
  border: 0;
  background: none;
  font: inherit;
  font-size: 13px;
  color: var(--text-faint);
  cursor: pointer;
  vertical-align: baseline;
}
.reader-content__feed:hover {
  color: var(--accent);
  text-decoration: underline;
}
.reader-content__feed:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
  border-radius: 4px;
}
.reader-content__body > :first-child {
  margin-top: 0;
}
.reader-content__body :deep(img),
.reader-content__body :deep(video) {
  max-width: 100%;
  height: auto;
  border-radius: 6px;
}
.reader-content__body :deep(a) {
  color: var(--accent);
  text-decoration: none;
}
.reader-content__body :deep(a):hover {
  text-decoration: underline;
}
.reader-content__body :deep(h1),
.reader-content__body :deep(h2),
.reader-content__body :deep(h3),
.reader-content__body :deep(h4),
.reader-content__body :deep(h5),
.reader-content__body :deep(h6) {
  line-height: 1.3;
  margin: 1.4em 0 0.6em;
}
.reader-content__body :deep(p) {
  margin: 0 0 1em;
}
.reader-content__body :deep(pre) {
  overflow-x: auto;
  padding: 12px 14px;
  border-radius: 8px;
  background: var(--bg-subtle);
  font-size: 13px;
  line-height: 1.5;
}
.reader-content__body :deep(code) {
  font-family: ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace;
  font-size: 0.9em;
  background: var(--bg-subtle);
  padding: 0.15em 0.4em;
  border-radius: 5px;
}
.reader-content__body :deep(pre code) {
  background: none;
  padding: 0;
}
.reader-content__body :deep(blockquote) {
  margin: 1em 0;
  padding: 0 1em;
  border-left: 3px solid var(--border-strong);
  color: var(--text-muted);
}
.reader-content__body :deep(hr) {
  border: 0;
  border-top: 1px solid var(--border);
  margin: 1.6em 0;
}
.reader-content__body :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 1em 0;
  font-size: 14px;
}
.reader-content__body :deep(th),
.reader-content__body :deep(td) {
  border: 1px solid var(--border);
  padding: 8px 10px;
  text-align: left;
}
.reader-content__body :deep(th) {
  background: var(--bg-subtle);
}
.reader-content__body :deep(ul),
.reader-content__body :deep(ol) {
  padding-left: 1.4em;
}
.reader-content__body :deep(figure) {
  margin: 1em 0;
  text-align: center;
}
.reader-content__body :deep(figcaption) {
  margin-top: 6px;
  font-size: 13px;
  color: var(--text-muted);
}
.reader-content__loading {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--bg) 70%, transparent);
  z-index: 5;
}
.reader-content__spinner {
  font-size: 22px;
  color: var(--text-muted);
  animation: reader-spin 0.8s linear infinite;
}
@keyframes reader-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
