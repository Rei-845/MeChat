<template>
  <span class="inline-flex items-center gap-0.5 rounded-md font-bold shrink-0"
        :style="`padding:${dense ? '0 4px' : '1px 6px'};
                 background:linear-gradient(135deg,#F59E0B,#FBBF24);
                 color:#78350F;
                 font-size:${dense ? '9px' : '10px'};
                 line-height:1.4`"
        title="VIP 会员">
    <Crown :size="dense ? 9 : 11" />
    <span v-if="!iconOnly">VIP</span>
  </span>
</template>

<script setup>
import { Crown } from 'lucide-vue-next'
defineProps({
  dense:    { type: Boolean, default: false },
  iconOnly: { type: Boolean, default: false },
})
</script>
