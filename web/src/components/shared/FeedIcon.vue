<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { feedIconUrl, type FeedIconSource } from '../../helpers/feedIcon'

const props = withDefaults(defineProps<{ feed?: FeedIconSource | null; size?: number }>(), {
  feed: null,
  size: 16,
})

const src = computed(() => feedIconUrl(props.feed, 32))
// Start with the RSS glyph; only swap to the favicon once the image has
// actually decoded (a 200 still has to download), so a slow or unreachable
// icon never leaves an empty box.
const loaded = ref(false)
const failed = ref(false)

// A new feed (or a feed whose site changed) starts from the glyph again so the
// stale icon isn't shown while the fresh one loads.
watch(src, () => {
  loaded.value = false
  failed.value = false
})

// Object-fit + a fixed box keeps rows from shifting whether we show an image or
// the glyph fallback.
const boxStyle = computed(() => ({ width: props.size + 'px', height: props.size + 'px' }))
</script>

<template>
  <span class="feed-icon" :style="boxStyle">
    <!-- Shown until the favicon loads (and retained as the fallback on error). -->
    <span
      v-if="!loaded || failed"
      aria-hidden="true"
      class="i-lucide-rss feed-icon__glyph"
      :style="{ fontSize: size + 'px' }"
    />
    <!-- Kept mounted so it can request/decode; revealed on load. -->
    <img
      v-if="src && !failed"
      :src="src"
      alt=""
      aria-hidden="true"
      loading="lazy"
      decoding="async"
      class="feed-icon__img"
      :class="{ 'feed-icon__img--ready': loaded }"
      @load="loaded = true"
      @error="failed = true"
    />
  </span>
</template>

<style scoped>
.feed-icon {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  overflow: hidden;
}
/* The icon box is a fixed square; the glyph and the loaded image occupy the
   same cell. The image is overlaid on the glyph and only painted once loaded. */
.feed-icon__img {
  position: absolute;
  width: 100%;
  height: 100%;
  object-fit: contain;
  border-radius: 3px;
  opacity: 0;
  transition: opacity 0.15s ease;
}
.feed-icon__img--ready {
  opacity: 1;
}
/* UnoCSS renders .i-lucide-rss as an inline-block sized by font-size; the
   colour follows currentColor so it adapts to light/dark themes. */
.feed-icon__glyph {
  display: block;
}
</style>
