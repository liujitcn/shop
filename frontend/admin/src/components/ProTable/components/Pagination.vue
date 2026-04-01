<template>
  <!-- 分页组件 -->
  <el-pagination
    :background="true"
    v-model:current-page="currentPage"
    v-model:page-size="pageSize"
    :page-sizes="[10, 25, 50, 100]"
    :total="pageable.total"
    :size="globalStore?.assemblySize ?? 'default'"
    layout="total, sizes, prev, pager, next, jumper"
  />
</template>

<script setup lang="ts" name="Pagination">
import { computed } from "vue";
import { useGlobalStore } from "@/stores/modules/global";
const globalStore = useGlobalStore();

interface Pageable {
  pageNum: number;
  pageSize: number;
  total: number;
}

interface PaginationProps {
  pageable: Pageable;
  handleSizeChange: (size: number) => void;
  handleCurrentChange: (currentPage: number) => void;
}

const props = defineProps<PaginationProps>();

const currentPage = computed({
  get: () => props.pageable.pageNum,
  set: value => props.handleCurrentChange(value)
});

const pageSize = computed({
  get: () => props.pageable.pageSize,
  set: value => props.handleSizeChange(value)
});
</script>
