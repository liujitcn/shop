<!-- 分类树 -->
<template>
  <el-card shadow="never">
    <el-input v-model="name" placeholder="分类名称" clearable>
      <template #prefix>
        <el-icon><Search /></el-icon>
      </template>
    </el-input>

    <el-tree
      ref="goodsCategoryTreeRef"
      class="mt-2"
      :data="goodsCategoryList"
      :check-strictly="true"
      :props="{ children: 'children', label: 'label', disabled: '' }"
      :expand-on-click-node="false"
      :filter-node-method="handleFilter"
      default-expand-all
      @node-click="handleNodeClick"
    />
  </el-card>
</template>

<script setup lang="ts">
import { defGoodsCategoryService } from "@/api/admin/goods_category";
import { TreeOptionResponse_Option } from "@/rpc/common/common";
const props = defineProps({
  modelValue: {
    type: [Number],
    default: undefined,
  },
});

const goodsCategoryList = ref<TreeOptionResponse_Option[]>(); // 分类列表
const goodsCategoryTreeRef = ref(ElTree); // 分类树
const name = ref(); // 分类名称

const emits = defineEmits(["node-click"]);

const deptId = useVModel(props, "modelValue", emits);

watchEffect(
  () => {
    goodsCategoryTreeRef.value.filter(name.value);
  },
  {
    flush: "post", // watchEffect会在DOM挂载或者更新之前就会触发，此属性控制在DOM元素更新后运行
  }
);

/**
 * 分类筛选
 */
function handleFilter(value: string, data: any) {
  if (!value) {
    return true;
  }
  return data.label.indexOf(value) !== -1;
}

/** 分类树节点 Click */
function handleNodeClick(data: { [key: string]: any }) {
  deptId.value = data.value;
  emits("node-click");
}

onBeforeMount(() => {
  defGoodsCategoryService.OptionGoodsCategory({}).then((response) => {
    goodsCategoryList.value = response.list;
  });
});
</script>
