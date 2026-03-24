<script setup lang="ts">
import Order from "../index.vue";
defineOptions({
  name: "Shipped",
  inheritAttrs: false,
});
</script>

<template>
  <Order :status="2" />
</template>

<style scoped lang="scss"></style>
