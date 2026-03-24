<!-- 部门树 -->
<template>
  <el-card shadow="never">
    <el-input v-model="name" placeholder="部门名称" clearable>
      <template #prefix>
        <el-icon><Search /></el-icon>
      </template>
    </el-input>

    <el-tree
      ref="baseDeptTreeRef"
      class="mt-2"
      :data="baseDeptList"
      :props="{ children: 'children', label: 'label', disabled: '' }"
      :expand-on-click-node="false"
      :filter-node-method="handleFilter"
      default-expand-all
      highlight-current
      @node-click="handleNodeClick"
    />
  </el-card>
</template>

<script setup lang="ts">
import { defBaseDeptService } from "@/api/admin/base_dept";
import { TreeOptionResponse_Option } from "@/rpc/common/common";
const props = defineProps({
  modelValue: {
    type: [Number],
    default: undefined,
  },
});

const baseDeptList = ref<TreeOptionResponse_Option[]>(); // 部门列表
const baseDeptTreeRef = ref(ElTree); // 部门树
const name = ref(); // 部门名称

const emits = defineEmits(["node-click"]);

const baseDeptId = useVModel(props, "modelValue", emits);

watchEffect(
  () => {
    baseDeptTreeRef.value.filter(name.value);
  },
  {
    flush: "post", // watchEffect会在DOM挂载或者更新之前就会触发，此属性控制在DOM元素更新后运行
  }
);

/**
 * 部门筛选
 */
function handleFilter(value: string, data: any) {
  if (!value) {
    return true;
  }
  return data.label.indexOf(value) !== -1;
}

/** 部门树节点 Click */
function handleNodeClick(data: { [key: string]: any }) {
  baseDeptId.value = data.value;
  emits("node-click");
}

onBeforeMount(() => {
  defBaseDeptService.OptionBaseDept({}).then((response) => {
    baseDeptList.value = response.list;
  });
});
</script>
