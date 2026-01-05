<template>
  <el-config-provider :locale="zhCn">
    <router-view v-if="$route.path === '/login'" />
    <el-container v-else class="layout" :class="{ dark: isDark }">
      <el-aside width="200px">
        <div class="logo">NOFX 运维</div>
        <el-menu :default-active="$route.path" router>
          <el-menu-item index="/">
            <el-icon><Monitor /></el-icon>
            <span>仪表盘</span>
          </el-menu-item>
          <el-menu-item index="/backup">
            <el-icon><FolderOpened /></el-icon>
            <span>备份管理</span>
          </el-menu-item>
          <el-menu-item index="/config">
            <el-icon><Edit /></el-icon>
            <span>配置管理</span>
          </el-menu-item>
          <el-menu-item index="/settings">
            <el-icon><Setting /></el-icon>
            <span>系统设置</span>
          </el-menu-item>
        </el-menu>
        <div class="theme-toggle">
          <el-button @click="toggleTheme" circle size="small">
            <el-icon><Sunny v-if="isDark" /><Moon v-else /></el-icon>
          </el-button>
        </div>
      </el-aside>
      <el-main><router-view /></el-main>
    </el-container>
  </el-config-provider>
</template>

<script setup>
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import { Sunny, Moon, Setting } from '@element-plus/icons-vue'
import { useTheme } from './stores/theme'
const { isDark, toggleTheme } = useTheme()
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, #app { height: 100%; }
.layout { height: 100%; }

/* 侧边栏 - 始终深色 */
.el-aside { background: linear-gradient(180deg, #1a1a2e 0%, #16213e 100%); position: relative; }
.logo { height: 60px; line-height: 60px; text-align: center; color: #16c79a; font-size: 18px; font-weight: bold; letter-spacing: 2px; }
.el-menu { border: none; background: transparent; }
.el-menu-item { color: #a0aec0; }
.el-menu-item:hover, .el-menu-item.is-active { background: rgba(22, 199, 154, 0.15) !important; color: #16c79a; }
.theme-toggle { position: absolute; bottom: 20px; left: 50%; transform: translateX(-50%); }

/* 浅色主题 */
.layout .el-main { background: #f0f2f5; padding: 0; overflow: auto; }

/* 深色主题 */
.layout.dark .el-main { background: #0f0f1a; }
</style>
