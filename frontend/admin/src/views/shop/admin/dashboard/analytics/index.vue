<template>
  <div class="analytics-entry" />
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import { useRouter } from "vue-router";

defineOptions({
  name: "Analytics"
});

const router = useRouter();

onMounted(() => {
  router.replace("/dashboard/analytics/user");
});
</script>

<style scoped lang="scss">
.analytics-entry {
  min-height: 1px;
}
</style>
