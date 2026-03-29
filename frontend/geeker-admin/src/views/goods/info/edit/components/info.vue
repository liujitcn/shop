<template>
  <div class="app-container">
    <el-card shadow="never">
      <el-form ref="dataFormRef" :rules="rules" :model="formData" label-width="120px">
        <el-form-item label="商品分类" prop="categoryId">
          <el-tree-select v-model="formData.categoryId" placeholder="请选择商品分类" :data="goodsCategoryOptions" filterable />
        </el-form-item>
        <el-form-item label="商品名称" prop="name">
          <el-input v-model="formData.name" placeholder="请输入商品名称" />
        </el-form-item>
        <el-form-item label="商品描述" prop="desc">
          <el-input v-model="formData.desc" placeholder="请输入商品描述" type="textarea" />
        </el-form-item>

        <el-form-item label="商品主图" prop="picture">
          <UploadImg v-model:image-url="formData.picture" />
        </el-form-item>

        <el-form-item label="商品轮播图" prop="banner">
          <UploadImgs v-model:file-list="bannerFileList" />
        </el-form-item>

        <el-form-item label="商品详情" prop="detail">
          <UploadImgs v-model:file-list="detailFileList" />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-switch
            v-model="formData.status"
            inline-prompt
            active-text="上架"
            inactive-text="下架"
            :active-value="GoodsStatus.PUT_ON"
            :inactive-value="GoodsStatus.PULL_OFF"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" @click="handleNext">下一步，设置商品属性</el-button>
      </template>
    </el-card>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, reactive, ref, toRefs } from "vue";
import UploadImg from "@/components/Upload/Img.vue";
import UploadImgs from "@/components/Upload/Imgs.vue";
import { defGoodsCategoryService } from "@/api/admin/goods_category";
import type { UploadUserFile } from "element-plus";
defineOptions({
  name: "GoodsEditInfo",
  inheritAttrs: false
});
const emit = defineEmits(["next", "update:modelValue"]);
import type { TreeOptionResponse_Option } from "@/rpc/common/common";
import { GoodsStatus } from "@/rpc/common/enum";
const dataFormRef = ref();

const props = defineProps({
  modelValue: {
    type: Object,
    default: () => ({})
  }
});

const formData: any = computed({
  get: () => props.modelValue,
  set: value => {
    emit("update:modelValue", value);
  }
});

/** 将商品轮播图值适配为上传组件文件列表。 */
const bannerFileList = computed<UploadUserFile[]>({
  get: () => (formData.value.banner ?? []).map((url: string) => ({ name: url.split("/").pop() ?? "image", url })),
  set: value => {
    formData.value.banner = value.map(item => item.url ?? "").filter(Boolean);
  }
});

/** 将商品详情图值适配为上传组件文件列表。 */
const detailFileList = computed<UploadUserFile[]>({
  get: () => (formData.value.detail ?? []).map((url: string) => ({ name: url.split("/").pop() ?? "image", url })),
  set: value => {
    formData.value.detail = value.map(item => item.url ?? "").filter(Boolean);
  }
});

const state = reactive({
  goodsCategoryOptions: [] as Array<TreeOptionResponse_Option>,
  rules: {
    categoryId: [{ required: true, message: "请选择商品分类", trigger: "change" }],
    name: [{ required: true, message: "请输入商品名称", trigger: "blur" }],
    desc: [{ required: true, message: "请输入商品描述", trigger: "blur" }],
    picture: [{ required: true, message: "请上传商品图片", trigger: "change" }],
    banner: [{ required: true, message: "请上传商品轮播图", trigger: "change" }],
    detail: [{ required: true, message: "请上传商品详情", trigger: "change" }]
  }
});

const { goodsCategoryOptions, rules } = toRefs(state);

function handleNext() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      emit("next");
    }
  });
}

// 查询
function handleQuery() {
  // 加载分类下拉数据源
  defGoodsCategoryService.OptionGoodsCategory({}).then(res => {
    state.goodsCategoryOptions = res.list;
  });
}

onMounted(() => {
  handleQuery();
});
</script>
