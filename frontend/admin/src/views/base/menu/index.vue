<template>
  <div class="app-container">
    <div class="search-bar">
      <el-form ref="queryFormRef" :model="queryParams" :inline="true">
        <el-form-item>
          <el-button class="filter-item" type="primary" icon="search" @click="handleQuery">
            搜索
          </el-button>
          <el-button icon="refresh" @click="handleResetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </div>

    <el-card shadow="never">
      <div class="mb-10px">
        <el-button
          v-hasPerm="['base:menu:create']"
          type="success"
          icon="plus"
          @click="handleOpenDialog(0)"
        >
          新增
        </el-button>
        <el-button
          v-hasPerm="['base:menu:delete']"
          type="danger"
          :disabled="selectIds.length == 0"
          icon="delete"
          @click="handleDelete()"
        >
          删除
        </el-button>
      </div>

      <el-table
        v-loading="loading"
        :data="menuList"
        row-key="id"
        :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" align="center" />
        <el-table-column label="菜单名称" min-width="200">
          <template #default="scope">
            <template v-if="scope.row.meta.icon && scope.row.meta.icon.startsWith('el-icon')">
              <el-icon style="vertical-align: -0.15em">
                <component :is="scope.row.meta.icon.replace('el-icon-', '')" />
              </el-icon>
            </template>
            <template v-else-if="scope.row.meta.icon">
              <svg-icon :icon-class="scope.row.meta.icon" />
            </template>
            {{ scope.row.meta.title }}
          </template>
        </el-table-column>
        <el-table-column prop="type" label="菜单类型">
          <template #default="scope">
            <DictLabel v-model="scope.row.type" code="base_menu_type" />
          </template>
        </el-table-column>
        <el-table-column prop="path" label="路由路径/权限标识" />
        <el-table-column prop="name" label="路由名称" />
        <el-table-column prop="component" label="组件路径" />
        <el-table-column prop="redirect" label="重定向地址" />
        <el-table-column label="是否显示" align="center">
          <template #default="scope">
            <el-tag v-if="scope.row.meta.hidden" type="info">隐藏</el-tag>
            <el-tag v-else type="success">显示</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" align="right" />
        <el-table-column label="状态" align="center" prop="status" width="80">
          <template #default="scope">
            <el-switch
              v-model="scope.row.status"
              inline-prompt
              :active-value="Status.ENABLE"
              :inactive-value="Status.DISABLE"
              active-text="启用"
              inactive-text="禁用"
              :disabled="!useUserStore().hasPerm('base:menu:status')"
              @change="handleSetStatus(scope.row)"
            />
          </template>
        </el-table-column>
        <el-table-column label="创建时间" prop="createdAt" align="center" />
        <el-table-column label="更新时间" prop="updatedAt" align="center" />
        <el-table-column label="操作" fixed="right" align="left" width="200">
          <template #default="scope">
            <el-button
              v-hasPerm="['base:menu:create']"
              type="primary"
              link
              size="small"
              icon="plus"
              @click.stop="handleOpenDialog(scope.row.parentId, scope.row.id)"
            >
              新增
            </el-button>
            <el-button
              v-hasPerm="['base:menu:update']"
              type="primary"
              link
              size="small"
              icon="edit"
              @click.stop="handleOpenDialog(scope.row.parentId, scope.row.id)"
            >
              编辑
            </el-button>
            <el-button
              v-hasPerm="['base:menu:delete']"
              type="danger"
              link
              size="small"
              icon="delete"
              @click.stop="handleDelete(scope.row.id)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog
      v-model="dialog.visible"
      :title="dialog.title"
      width="1400px"
      @closed="handleCloseDialog"
    >
      <el-form ref="dataFormRef" :model="formData" :rules="rules" label-width="120px">
        <el-form-item label="上级菜单" prop="parentId">
          <el-tree-select
            v-model="formData.parentId"
            placeholder="选择上级菜单"
            :data="menuOptions"
            filterable
            check-strictly
            :render-after-expand="false"
          />
        </el-form-item>
        <el-form-item label="菜单类型" prop="type">
          <Dict v-model="formData.type" code="base_menu_type" />
        </el-form-item>
        <el-form-item
          v-if="
            formData.type == BaseMenuType.FOLDER ||
            formData.type == BaseMenuType.MENU ||
            formData.type == BaseMenuType.EXT_LINK
          "
          label="菜单标题"
          prop="meta.title"
        >
          <el-input v-model="meta.title" placeholder="请输入菜单标题" />
        </el-form-item>
        <el-form-item
          v-if="formData.type == BaseMenuType.BUTTON"
          label="按钮名称"
          prop="meta.title"
        >
          <el-input v-model="meta.title" placeholder="请输入按钮名称" />
        </el-form-item>

        <el-form-item v-if="formData.type == BaseMenuType.BUTTON" label="权限标识" prop="path">
          <el-input v-model="formData.path" placeholder="请输入权限标识" />
        </el-form-item>

        <el-form-item v-if="formData.type == BaseMenuType.EXT_LINK" label="外链地址" prop="path">
          <el-input v-model="formData.path" placeholder="请输入外链完整路径" />
        </el-form-item>

        <el-form-item
          v-if="formData.type == BaseMenuType.FOLDER || formData.type == BaseMenuType.MENU"
          prop="path"
        >
          <template #label>
            <div class="flex-y-center">
              路由路径
              <el-tooltip placement="bottom" effect="light">
                <template #content>
                  定义应用中不同页面对应的 URL 路径，目录需以 / 开头，菜单项不用。例如：系统管理目录
                  /base，系统管理下的用户管理菜单 user。
                </template>
                <el-icon class="ml-1 cursor-pointer">
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-input
            v-if="formData.type == BaseMenuType.FOLDER"
            v-model="formData.path"
            placeholder="base"
          />
          <el-input v-else v-model="formData.path" placeholder="user" />
        </el-form-item>

        <el-form-item v-if="formData.type == BaseMenuType.FOLDER" label="跳转路由" prop="redirect">
          <el-input v-model="formData.redirect" placeholder="跳转路由" />
        </el-form-item>
        <el-form-item
          v-if="
            formData.type == BaseMenuType.FOLDER ||
            formData.type == BaseMenuType.MENU ||
            formData.type == BaseMenuType.EXT_LINK
          "
          label="菜单图标"
          prop="meta.icon"
        >
          <!-- 图标选择器 -->
          <icon-select v-model="meta.icon" />
        </el-form-item>

        <el-form-item v-if="formData.type == BaseMenuType.MENU" prop="name">
          <template #label>
            <div class="flex-y-center">
              路由名称
              <el-tooltip placement="bottom" effect="light">
                <template #content>
                  如果需要开启缓存，需保证页面 defineOptions 中的 name 与此处一致，建议使用驼峰。
                </template>
                <el-icon class="ml-1 cursor-pointer">
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-input v-model="formData.name" placeholder="请输入路由名称" />
        </el-form-item>

        <el-form-item v-if="formData.type == BaseMenuType.MENU" prop="component">
          <template #label>
            <div class="flex-y-center">
              组件路径
              <el-tooltip placement="bottom" effect="light">
                <template #content>
                  组件页面完整路径，相对于 src/views/，如 base/user/index，缺省后缀 .vue
                </template>
                <el-icon class="ml-1 cursor-pointer">
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>

          <el-input v-model="formData.component" placeholder="base/user/index" style="width: 95%">
            <template v-if="formData.type == BaseMenuType.MENU" #prepend>src/views/</template>
            <template v-if="formData.type == BaseMenuType.MENU" #append>.vue</template>
          </el-input>
        </el-form-item>

        <el-form-item
          v-if="
            formData.type == BaseMenuType.FOLDER ||
            formData.type == BaseMenuType.MENU ||
            formData.type == BaseMenuType.EXT_LINK
          "
          prop="meta.hidden"
          label="是否隐藏"
        >
          <el-switch
            v-model="meta.hidden"
            inline-prompt
            active-text="是"
            inactive-text="否"
            :active-value="true"
            :inactive-value="false"
          />
        </el-form-item>

        <el-form-item
          v-if="formData.type == BaseMenuType.FOLDER || formData.type == BaseMenuType.MENU"
          prop="meta.keepAlive"
        >
          <template #label>
            <div class="flex-y-center">
              始终显示
              <el-tooltip placement="bottom" effect="light">
                <template #content>
                  选择"是"，即使目录或菜单下只有一个子节点，也会显示父节点。
                  <br />
                  选择"否"，如果目录或菜单下只有一个子节点，则只显示该子节点，隐藏父节点。
                  <br />
                  如果是叶子节点，请选择"否"。
                </template>
                <el-icon class="ml-1 cursor-pointer">
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-switch
            v-model="meta.alwaysShow"
            inline-prompt
            active-text="是"
            inactive-text="否"
            :active-value="true"
            :inactive-value="false"
          />
        </el-form-item>

        <el-form-item
          v-if="formData.type == BaseMenuType.MENU"
          label="缓存页面"
          prop="meta.keepAlive"
        >
          <el-switch
            v-model="meta.keepAlive"
            inline-prompt
            active-text="开启"
            inactive-text="关闭"
            :active-value="true"
            :inactive-value="false"
          />
        </el-form-item>

        <el-form-item v-if="formData.type == BaseMenuType.MENU">
          <template #label>
            <div class="flex-y-center">
              路由参数
              <el-tooltip placement="bottom" effect="light">
                <template #content>
                  组件页面使用 `useRoute().query.参数名` 获取路由参数值。
                </template>
                <el-icon class="ml-1 cursor-pointer">
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>

          <div v-if="!meta.params || meta.params.length == 0">
            <el-button
              type="success"
              plain
              @click="meta.params = [{ key: '', value: '' }]"
            >
              添加路由参数
            </el-button>
          </div>

          <div v-else>
            <div v-for="(item, index) in meta.params" :key="index">
              <el-input v-model="item.key" placeholder="参数名" style="width: 100px" />

              <span class="mx-1">=</span>

              <el-input v-model="item.value" placeholder="参数值" style="width: 100px" />

              <el-icon
                v-if="meta.params.indexOf(item) == meta.params.length - 1"
                class="ml-2 cursor-pointer color-[var(--el-color-success)]"
                style="vertical-align: -0.15em"
                @click="meta.params.push({ key: '', value: '' })"
              >
                <CirclePlusFilled />
              </el-icon>
              <el-icon
                class="ml-2 cursor-pointer color-[var(--el-color-danger)]"
                style="vertical-align: -0.15em"
                @click="meta.params.splice(meta.params.indexOf(item), 1)"
              >
                <DeleteFilled />
              </el-icon>
            </div>
          </div>
        </el-form-item>
        <!-- API 穿梭框 -->
        <el-form-item
          v-if="formData.type == BaseMenuType.MENU || formData.type == BaseMenuType.BUTTON"
          label="API 列表"
        >
          <el-transfer
            v-model="formData.apis"
            :titles="['可选API', '已选API']"
            filterable
            :data="transferData"
          >
            <template #default="{ option }">
              <el-popover effect="light" trigger="hover" placement="top" width="auto">
                <template #default>
                  <div>操作名称: {{ option.operation }}</div>
                  <div>请求方式: {{ option.method }}</div>
                  <div>请求地址: {{ option.path }}</div>
                </template>
                <template #reference>{{ option.label }}</template>
              </el-popover>
            </template>
          </el-transfer>
        </el-form-item>
        <el-form-item
          v-if="
            formData.type == BaseMenuType.FOLDER ||
            formData.type == BaseMenuType.MENU ||
            formData.type == BaseMenuType.EXT_LINK
          "
          label="排序"
          prop="sort"
        >
          <el-input-number
            v-model="formData.sort"
            controls-position="right"
            :min="1"
            :precision="0"
            :step="1"
          />
        </el-form-item>
        <el-form-item label="状态" prop="status">
          <el-switch
            v-model="formData.status"
            inline-prompt
            active-text="启用"
            inactive-text="禁用"
            :active-value="Status.ENABLE"
            :inactive-value="Status.DISABLE"
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <div class="dialog-footer">
          <el-button type="primary" @click="handleSubmit">确 定</el-button>
          <el-button @click="handleCloseDialog">取 消</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { TreeOptionResponse_Option } from "@/rpc/common/common";

defineOptions({
  name: "BaseMenu",
  inheritAttrs: false,
});

import { defBaseMenuService } from "@/api/admin/base_menu";
import type { BaseMenu, BaseMenuForm, BaseMenuMeta, BaseMenuParams } from "@/rpc/admin/base_menu";
import type { Empty } from "@/rpc/google/protobuf/empty";
import { useUserStore } from "@/store";
import type { BaseApi } from "@/rpc/admin/base_api";
import { defBaseApiService } from "@/api/admin/base_api";
import { BaseMenuType, Status } from "@/rpc/common/enum";

const queryFormRef = ref(ElForm);
const dataFormRef = ref(ElForm);

const loading = ref(false);
const selectIds = ref<number[]>([]);
const queryParams = reactive<Empty>({});
const apiList = ref<BaseApi[]>([]);

const dialog = reactive({
  title: "",
  visible: false,
});

const menuList = ref<BaseMenu[]>();
const menuOptions = ref<TreeOptionResponse_Option[]>();

const formData = reactive<BaseMenuForm>({
  /** 菜单ID */
  id: 0,
  /** 父节点ID */
  parentId: 0,
  /** 菜单类型 */
  type: undefined,
  /** 路由地址 */
  path: "",
  /** 路由命名，然后我们可以使用 name 而不是 path 来传递 to 属性给 <router-link>。 */
  name: "",
  /** 组件路径 */
  component: "",
  /** 重定向地址 */
  redirect: "",
  /** 路由元信息 */
  meta: {
    /** 菜单标题 */
    title: "",
    /** 菜单图标 */
    icon: "",
    /** 【目录】只有一个子路由是否始终显示 */
    alwaysShow: false,
    /** 是否隐藏(true-是 false-否) */
    hidden: false,
    /** 【菜单】是否开启页面缓存 */
    keepAlive: false,
    /** 参数 */
    params: [],
  },
  apis: [],
  /** 排序 */
  sort: 1,
  /** 菜单状态 */
  status: Status.ENABLE,
});

// 添加一个 computed 属性来确保 meta 始终存在
const meta = computed({
  get: () => {
    if (!formData.meta) {
      formData.meta = {
        title: "",
        icon: "",
        alwaysShow: false,
        hidden: false,
        keepAlive: false,
        params: [],
      };
    }
    return formData.meta;
  },
  set: (value) => {
    formData.meta = value;
  }
});

const rules = reactive({
  parentId: [{ required: true, message: "请选择上级菜单", trigger: "change" }],
  "meta.title": [
    {
      required: true,
      trigger: "blur",
      validator: (rule: any, val: string, callback: any) => {
        if (!val) {
          if (formData.type == BaseMenuType.BUTTON) {
            callback(new Error("请输入按钮名称"));
          } else {
            callback(new Error("请输入菜单标题"));
          }
        } else {
          callback();
        }
      },
    },
  ],
  type: [{ required: true, message: "请选择菜单类型", trigger: "change" }],
  "meta.icon": [{ required: true, message: "请选择菜单图标", trigger: "blur" }],
  path: [
    {
      required: true,
      trigger: "blur",
      validator: (rule: any, val: string, callback: any) => {
        if (!val) {
          if (formData.type == BaseMenuType.BUTTON) {
            callback(new Error("请输入权限标识"));
          } else if (formData.type == BaseMenuType.EXT_LINK) {
            callback(new Error("请输入完整外部路径"));
          } else {
            callback(new Error("请输入路由路径"));
          }
        } else {
          callback();
        }
      },
    },
  ],
  name: [
    {
      required: true,
      trigger: "blur",
      validator: (rule: any, val: string, callback: any) => {
        if (formData.type == BaseMenuType.MENU && !val) {
          callback(new Error("请输入路由名称"));
        } else {
          callback();
        }
      },
    },
  ],
  component: [
    {
      required: true,
      trigger: "blur",
      validator: (rule: any, val: string, callback: any) => {
        if (formData.type == BaseMenuType.MENU && !val) {
          callback(new Error("请输入组件路径"));
        } else {
          callback();
        }
      },
    },
  ],
  sort: [{ required: true, message: "排序不能为空", trigger: "blur" }],
  status: [{ required: true, message: "状态不能为空", trigger: "blur" }],
});

// 查询菜单
function handleQuery() {
  loading.value = true;
  defBaseMenuService.TreeBaseMenu(queryParams).then((data) => {
    menuList.value = data.list;
    loading.value = false;
  });
}

// 重置查询
function handleResetQuery() {
  queryFormRef.value.resetFields();
  handleQuery();
}

// 处理选中项变化
function handleSelectionChange(selection: any) {
  selectIds.value = selection.map((item: any) => item.id);
}

// api 穿梭框数据
const transferData = computed(() => {
  return apiList.value.map((item) => ({
    ...item,
    key: item.operation,
    label: `${item.serviceDesc}/${item.desc}`,
  }));
});

/**
 * 打开菜单弹窗
 *
 * @param parentId 父菜单ID
 * @param menuId 菜单ID
 */
async function handleOpenDialog(parentId: number, menuId?: number) {
  // 加载菜单下拉数据
  const optionBaseMenuResponse = await defBaseMenuService.OptionBaseMenu({});
  menuOptions.value = [
    {
      value: 0,
      label: "顶级菜单",
      disabled: false,
      children: optionBaseMenuResponse.list || [],
    },
  ];
  // 加载API列表
  const listBaseApiResponse = await defBaseApiService.ListBaseApi({});
  apiList.value = listBaseApiResponse.list || [];
  dialog.visible = true;
  if (menuId) {
    dialog.title = "修改菜单";
    defBaseMenuService
      .GetBaseMenu({
        value: menuId,
      })
      .then((data) => {
        Object.assign(formData, data);
      });
  } else {
    dialog.title = "新增菜单";
    formData.parentId = parentId;
  }
}

// 提交菜单表单
function handleSubmit() {
  dataFormRef.value.validate((valid: any) => {
    if (valid) {
      loading.value = true;
      const menuId = formData.id;
      if (menuId) {
        defBaseMenuService
          .UpdateBaseMenu(formData)
          .then(() => {
            ElMessage.success("修改成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      } else {
        if (formData.type == BaseMenuType.FOLDER) {
          formData.component = "Layout";
        }
        defBaseMenuService
          .CreateBaseMenu(formData)
          .then(() => {
            ElMessage.success("新增成功");
            handleCloseDialog();
            handleQuery();
          })
          .finally(() => (loading.value = false));
      }
    }
  });
}

// 设置字典状态
function handleSetStatus(row: BaseMenu) {
  let text = row.status == 1 ? "启用" : "禁用";
  ElMessageBox.confirm(`是否确定${text}菜单为：${row.name}?`, "提示", {
    confirmButtonText: "确认",
    cancelButtonText: "取消",
    type: "warning",
  })
    .then(() => {
      defBaseMenuService.SetBaseMenuStatus({ id: row.id, status: row.status }).then(() => {
        handleQuery();
        ElMessage.success(`${text}成功`);
      });
    })
    .catch(() => {
      if (row.status == 0) {
        row.status = 1;
      } else {
        row.status = 0;
      }
    });
}

// 删除菜单
function handleDelete(menuId?: number) {
  const menuIds = [menuId || selectIds.value].join(",");

  if (!menuIds) {
    ElMessage.warning("请勾选删除项");
    return;
  }

  ElMessageBox.confirm("确认删除已选中的数据项?", "警告", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning",
  }).then(
    () => {
      loading.value = true;
      defBaseMenuService
        .DeleteBaseMenu({
          value: menuIds,
        })
        .then(() => {
          ElMessage.success("删除成功");
          handleResetQuery();
        })
        .finally(() => (loading.value = false));
    },
    () => {
      ElMessage.info("已取消删除");
    }
  );
}

// 重置表单
function resetForm() {
  dataFormRef.value.resetFields();
  dataFormRef.value.clearValidate();

  formData.id = 0;
  formData.parentId = 0;
  formData.component = "";
  formData.meta = {
    title: "",
    icon: "",
    alwaysShow: false,
    hidden: false,
    keepAlive: false,
    params: [],
  };
  formData.apis = [];
}

// 关闭弹窗
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

onMounted(() => {
  handleQuery();
});
</script>
<style scoped>
/* 使用深度选择器穿透 scoped 样式 */
:deep(.el-transfer-panel) {
  width: 450px; /* 面板整体宽度 */
}

/* 调整列表区域 */
:deep(.el-transfer-panel__list) {
  height: 600px; /* 列表高度 */
  width: 100%; /* 宽度继承父容器 */
}
</style>
