<template>
  <span class="ai-assistant-icon" aria-hidden="true">
    <span class="ai-assistant-icon__shell">
      <span class="ai-assistant-icon__bubble"></span>
      <span class="ai-assistant-icon__tail"></span>
      <span class="ai-assistant-icon__agent-ring"></span>
      <span class="ai-assistant-icon__agent-core"></span>
      <span class="ai-assistant-icon__signal ai-assistant-icon__signal--left"></span>
      <span class="ai-assistant-icon__signal ai-assistant-icon__signal--center"></span>
      <span class="ai-assistant-icon__signal ai-assistant-icon__signal--right"></span>
      <span class="ai-assistant-icon__spark ai-assistant-icon__spark--top"></span>
      <span class="ai-assistant-icon__spark ai-assistant-icon__spark--bottom"></span>
      <span class="ai-assistant-icon__orbit"></span>
      <span class="ai-assistant-icon__orbit ai-assistant-icon__orbit--inner"></span>
      <span class="ai-assistant-icon__node ai-assistant-icon__node--left"></span>
      <span class="ai-assistant-icon__node ai-assistant-icon__node--right"></span>
    </span>
  </span>
</template>

<script setup lang="ts" name="Icon"></script>

<style scoped lang="scss">
.ai-assistant-icon {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
}

.ai-assistant-icon__shell {
  position: absolute;
  inset: 0;
  display: block;
  background: linear-gradient(135deg, #44c3c8, #1d9aa1);
  border-radius: 50%;
  box-shadow:
    inset 0 1px 10px rgb(255 255 255 / 28%),
    0 10px 24px rgb(29 154 161 / 20%);
}

.ai-assistant-icon__bubble {
  position: absolute;
  top: 10px;
  left: 9px;
  width: 30px;
  height: 24px;
  background: rgb(255 255 255 / 18%);
  border: 1.5px solid rgb(255 255 255 / 68%);
  border-radius: 14px;
  box-shadow: inset 0 1px 6px rgb(255 255 255 / 14%);
}

.ai-assistant-icon__tail {
  position: absolute;
  bottom: 11px;
  left: 14px;
  width: 11px;
  height: 11px;
  background: rgb(255 255 255 / 18%);
  border-bottom: 1.5px solid rgb(255 255 255 / 68%);
  border-left: 1.5px solid rgb(255 255 255 / 68%);
  border-radius: 0 0 0 10px;
  transform: skewX(-18deg) rotate(-8deg);
}

.ai-assistant-icon__agent-ring {
  position: absolute;
  top: 14px;
  left: 14px;
  width: 20px;
  height: 20px;
  border: 1.5px solid rgb(255 255 255 / 82%);
  border-radius: 50%;
  box-shadow: 0 0 12px rgb(255 255 255 / 10%);
}

.ai-assistant-icon__agent-core {
  position: absolute;
  top: 20px;
  left: 20px;
  width: 8px;
  height: 8px;
  background: #ffffff;
  border-radius: 50%;
  box-shadow: 0 0 12px rgb(255 255 255 / 35%);
}

.ai-assistant-icon__signal {
  position: absolute;
  bottom: 16px;
  width: 3px;
  background: rgb(255 255 255 / 88%);
  border-radius: 999px;
}

.ai-assistant-icon__signal--left {
  right: 14px;
  height: 8px;
}

.ai-assistant-icon__signal--center {
  right: 19px;
  height: 12px;
}

.ai-assistant-icon__signal--right {
  right: 24px;
  height: 6px;
}

.ai-assistant-icon__spark {
  position: absolute;
  width: 4px;
  height: 4px;
  background: rgb(255 255 255 / 92%);
  border-radius: 50%;
}

.ai-assistant-icon__spark--top {
  top: 10px;
  right: 11px;
}

.ai-assistant-icon__spark--bottom {
  bottom: 11px;
  right: 10px;
}

.ai-assistant-icon__orbit {
  position: absolute;
  top: 8px;
  left: 8px;
  width: 32px;
  height: 28px;
  border: 1px solid rgb(255 255 255 / 22%);
  border-radius: 50%;
  transform: rotate(-18deg);
}

.ai-assistant-icon__orbit--inner {
  top: 11px;
  left: 11px;
  width: 26px;
  height: 22px;
  opacity: 0.6;
  transform: rotate(18deg);
}

.ai-assistant-icon__node {
  position: absolute;
  top: 21px;
  width: 5px;
  height: 5px;
  background: rgb(255 255 255 / 78%);
  border-radius: 50%;
}

.ai-assistant-icon__node--left {
  left: 7px;
}

.ai-assistant-icon__node--right {
  right: 7px;
}

html.dark {
  .ai-assistant-icon__shell {
    box-shadow:
      inset 0 1px 10px rgb(255 255 255 / 18%),
      0 10px 24px rgb(29 154 161 / 16%);
  }
}
</style>
