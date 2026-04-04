<template>
  <!-- 分页组件 -->
  <el-pagination
    background
    :current-page="currentPage"
    :page-size="pageSize"
    :page-sizes="[10, 25, 50, 100]"
    :total="pageable.total"
    :size="globalStore?.assemblySize ?? 'default'"
    layout="total, sizes, prev, pager, next, jumper"
    @update:current-page="handleCurrentPageUpdate"
    @update:page-size="handlePageSizeUpdate"
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

/** 同步当前页码更新，兼容 Element Plus 当前推荐写法。 */
function handleCurrentPageUpdate(value: number) {
  currentPage.value = value;
}

/** 同步分页大小更新，兼容 Element Plus 当前推荐写法。 */
function handlePageSizeUpdate(value: number) {
  pageSize.value = value;
}
</script>
