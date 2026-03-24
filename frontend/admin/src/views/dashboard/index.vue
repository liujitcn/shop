<template>
  <div class="dashboard-container">
    <el-card shadow="never">
      <el-row justify="space-between">
        <el-col :span="18" :xs="24">
          <div class="flex h-full items-center">
            <img
              class="w-20 h-20 mr-5 rounded-full"
              :src="userStore.userInfo.avatar + '?imageView2/1/w/80/h/80'"
            />
            <div>
              <p>{{ greetings }}</p>
              <p class="text-sm text-gray">今日天气晴朗，气温在15℃至25℃之间，东南风。</p>
            </div>
          </div>
        </el-col>
      </el-row>
    </el-card>

    <!-- 数据卡片 -->
    <el-row :gutter="10" class="mt-3">
      <!-- 用户 -->
      <el-col v-if="useUserStore().hasPerm('dashboard:count:user')" :xs="24" :sm="12" :lg="6">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-[var(--el-text-color-secondary)]">新增用户</span>
              <el-tag type="success">日</el-tag>
            </div>
          </template>

          <div class="flex items-center justify-between mt-5">
            <div class="text-lg text-right">
              {{ dashboardCountUser.newNum }}
            </div>
            <svg-icon icon-class="visit" size="2em" />
          </div>

          <div
            class="flex items-center justify-between mt-5 text-sm text-[var(--el-text-color-secondary)]"
          >
            <span>总用户数</span>
            <span>{{ dashboardCountUser.totalNum }}</span>
          </div>
        </el-card>
      </el-col>

      <!--商品-->
      <el-col v-if="useUserStore().hasPerm('dashboard:count:goods')" :xs="24" :sm="12" :lg="6">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-[var(--el-text-color-secondary)]">新增商品</span>
              <el-tag type="success">日</el-tag>
            </div>
          </template>

          <div class="flex items-center justify-between mt-5">
            <div class="text-lg text-right">
              {{ dashboardCountGoods.newNum }}
            </div>
            <svg-icon icon-class="ip" size="2em" />
          </div>

          <div
            class="flex items-center justify-between mt-5 text-sm text-[var(--el-text-color-secondary)]"
          >
            <span>总商品数</span>
            <span>{{ dashboardCountGoods.totalNum }}</span>
          </div>
        </el-card>
      </el-col>

      <!--订单量-->
      <el-col v-if="useUserStore().hasPerm('dashboard:count:order')" :xs="24" :sm="12" :lg="6">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-[var(--el-text-color-secondary)]">订单量</span>
              <el-tag type="success">日</el-tag>
            </div>
          </template>

          <div class="flex items-center justify-between mt-5">
            <div class="text-lg text-right">
              {{ dashboardCountOrder.newNum }}
            </div>
            <svg-icon icon-class="order" size="2em" />
          </div>

          <div
            class="flex items-center justify-between mt-5 text-sm text-[var(--el-text-color-secondary)]"
          >
            <span>总订单量</span>
            <span>{{ dashboardCountOrder.totalNum }}</span>
          </div>
        </el-card>
      </el-col>

      <!--销售额-->
      <el-col v-if="useUserStore().hasPerm('dashboard:count:sale')" :xs="24" :sm="12" :lg="6">
        <el-card shadow="never">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="text-[var(--el-text-color-secondary)]">销售额</span>
              <el-tag type="success">日</el-tag>
            </div>
          </template>

          <div class="flex items-center justify-between mt-5">
            <div class="text-lg text-right">
              {{ formatPrice(dashboardCountOrder.newNum) }}
            </div>
            <svg-icon icon-class="money" size="2em" />
          </div>

          <div
            class="flex items-center justify-between mt-5 text-sm text-[var(--el-text-color-secondary)]"
          >
            <span>总销售额</span>
            <span>{{ formatPrice(dashboardCountOrder.totalNum) }}</span>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- Echarts 图表 -->
    <el-row :gutter="10" class="mt-3">
      <el-col v-if="useUserStore().hasPerm('dashboard:bar:order')" :sm="24" :lg="12" class="mb-2">
        <OrderBarChart class="bg-[var(--el-bg-color-overlay)]" />
      </el-col>

      <el-col v-if="useUserStore().hasPerm('dashboard:bar:goods')" :sm="24" :lg="12" class="mb-2">
        <GoodsBarChart class="bg-[var(--el-bg-color-overlay)]" />
      </el-col>

      <el-col v-if="useUserStore().hasPerm('dashboard:pie:goods')" :sm="24" :lg="12" class="mb-2">
        <GoodsPieChart class="bg-[var(--el-bg-color-overlay)]" />
      </el-col>

      <el-col v-if="useUserStore().hasPerm('dashboard:radar:order')" :sm="24" :lg="12" class="mb-2">
        <OrderRadarChart class="bg-[var(--el-bg-color-overlay)]" />
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import OrderBarChart from "./components/OrderBarChart.vue";
import GoodsBarChart from "./components/GoodsBarChart.vue";
import GoodsPieChart from "./components/GoodsPieChart.vue";
import OrderRadarChart from "./components/OrderRadarChart.vue";
import { useUserStore } from "@/store/modules/user";
import { defDashboardService } from "@/api/admin/dashboard";
import type { DashboardCountResponse } from "@/rpc/admin/dashboard";
import { DashboardTimeType } from "@/rpc/admin/dashboard";
import { formatPrice } from "@/utils/utils";

defineOptions({
  name: "Dashboard",
  inheritAttrs: false,
});

const userStore = useUserStore();
const date: Date = new Date();

const greetings = computed(() => {
  const hours = date.getHours();
  if (hours >= 6 && hours < 8) {
    return "晨起披衣出草堂，轩窗已自喜微凉🌅！";
  } else if (hours >= 8 && hours < 12) {
    return "上午好，" + useUserStore().userInfo.nickName + "！";
  } else if (hours >= 12 && hours < 18) {
    return "下午好，" + useUserStore().userInfo.nickName + "！";
  } else if (hours >= 18 && hours < 24) {
    return "晚上好，" + useUserStore().userInfo.nickName + "！";
  } else if (hours >= 0 && hours < 6) {
    return "偷偷向银河要了一把碎星，只等你闭上眼睛撒入你的梦中，晚安🌛！";
  }
});

const dashboardCountUser = reactive<DashboardCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0,
});
const dashboardCountGoods = reactive<DashboardCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0,
});
const dashboardCountOrder = reactive<DashboardCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0,
});
const dashboardCountSale = reactive<DashboardCountResponse>({
  /** 新增数量 */
  newNum: 0,
  /** 总数量 */
  totalNum: 0,
});

async function handleQuery() {
  if (useUserStore().hasPerm("dashboard:count:user")) {
    const user = await defDashboardService.DashboardCountUser({
      timeType: DashboardTimeType.DAY,
    });
    Object.assign(dashboardCountUser, user);
  }

  if (useUserStore().hasPerm("dashboard:count:goods")) {
    const goods = await defDashboardService.DashboardCountGoods({
      timeType: DashboardTimeType.DAY,
    });
    Object.assign(dashboardCountGoods, goods);
  }
  if (useUserStore().hasPerm("dashboard:count:order")) {
    const order = await defDashboardService.DashboardCountOrder({
      timeType: DashboardTimeType.DAY,
    });
    Object.assign(dashboardCountOrder, order);
  }

  if (useUserStore().hasPerm("dashboard:count:sale")) {
    const sale = await defDashboardService.DashboardCountSale({
      timeType: DashboardTimeType.DAY,
    });
    Object.assign(dashboardCountSale, sale);
  }
}

onMounted(() => {
  handleQuery();
});
</script>

<style lang="scss" scoped>
.dashboard-container {
  position: relative;
  padding: 24px;
}
</style>
