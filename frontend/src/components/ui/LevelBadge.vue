<template>
  <span class="inline-flex items-center justify-center rounded shrink-0 font-bold select-none"
        :style="style"
        :title="`Lv.${level} · ${label}`">
    Lv.{{ level }}
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  level: { type: Number, default: 1 },
  tier:  { type: String, default: 'gray' }, // gray / blue / yellow / orange
  dense: { type: Boolean, default: false },
})

const TIER_STYLES = {
  gray:   { bg: 'rgba(156,163,175,0.18)', color: '#9CA3AF', border: 'rgba(156,163,175,0.3)' },
  blue:   { bg: 'rgba(96,165,250,0.18)',  color: '#60A5FA', border: 'rgba(96,165,250,0.3)' },
  yellow: { bg: 'rgba(234,179,8,0.18)',   color: '#EAB308', border: 'rgba(234,179,8,0.3)' },
  orange: { bg: 'rgba(249,115,22,0.18)',  color: '#F97316', border: 'rgba(249,115,22,0.3)' },
}

const TIER_LABELS = { gray: '灰牌', blue: '蓝牌', yellow: '黄牌', orange: '橙牌' }

const label = computed(() => TIER_LABELS[props.tier] || '')

const style = computed(() => {
  const t = TIER_STYLES[props.tier] || TIER_STYLES.gray
  const size = props.dense ? '9px' : '10px'
  const pad  = props.dense ? '0 3px' : '1px 5px'
  return `background:${t.bg};color:${t.color};border:1px solid ${t.border};` +
         `font-size:${size};line-height:1.4;padding:${pad};`
})
</script>
