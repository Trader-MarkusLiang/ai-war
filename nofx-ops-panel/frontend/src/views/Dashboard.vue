<template>
  <div class="dashboard" :class="{ dark: isDark }">
    <div class="header">
      <h2>NOFX 运维面板</h2>
      <div class="header-actions">
        <el-switch v-model="autoRefresh" active-text="自动刷新" size="small" style="margin-right:12px" />
        <el-button @click="refresh" :loading="refreshing" circle><el-icon><Refresh /></el-icon></el-button>
      </div>
    </div>

    <!-- 服务器信息 -->
    <el-card shadow="hover" class="server-info">
      <el-descriptions :column="4" size="small" border>
        <el-descriptions-item label="服务器IP">{{ server.ip || '-' }}</el-descriptions-item>
        <el-descriptions-item label="主机名">{{ server.hostname || '-' }}</el-descriptions-item>
        <el-descriptions-item label="系统">{{ server.os || '-' }}</el-descriptions-item>
        <el-descriptions-item label="运行时间">{{ server.uptime || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <!-- 容器状态 -->
    <el-row :gutter="16">
      <el-col :span="12" v-for="c in containers" :key="c.name">
        <el-card shadow="hover" class="container-card">
          <div class="container-header">
            <div class="container-title">
              <span class="status-dot" :class="c.status?.includes('Up') ? 'online' : 'offline'"></span>
              <span class="name">{{ c.name }}</span>
            </div>
            <el-tag :type="c.status?.includes('Up') ? 'success' : 'danger'" size="small" effect="dark">
              {{ c.status?.includes('Up') ? '运行中' : '已停止' }}
            </el-tag>
          </div>
          <div class="container-metrics">
            <div class="metric">
              <span class="metric-label">内存</span>
              <el-progress :percentage="parseMemPercent(c.memory)" :stroke-width="6" :show-text="false" />
              <span class="metric-value">{{ c.memory || '-' }}</span>
            </div>
            <div class="metric">
              <span class="metric-label">CPU</span>
              <el-progress :percentage="parseCpuPercent(c.cpu)" :stroke-width="6" :show-text="false" status="warning" />
              <span class="metric-value">{{ c.cpu || '-' }}</span>
            </div>
          </div>
          <div class="container-actions">
            <el-button size="small" plain @click="serviceAction('restart', c.name)"><el-icon><RefreshRight /></el-icon>重启</el-button>
            <el-button size="small" plain @click="showLogs(c.name)"><el-icon><Document /></el-icon>日志</el-button>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- SSH隧道 + 系统资源 -->
    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="12">
        <el-card shadow="hover">
          <template #header>
            <div class="log-header">
              <span>SSH隧道</span>
              <el-tag :type="tunnel.running ? 'success' : 'info'" size="small" effect="dark">{{ tunnel.running ? '已连接' : '未连接' }}</el-tag>
            </div>
          </template>
          <div class="tunnel-info" v-if="tunnel.running">
            <div>前端: localhost:3333 → 3000</div>
            <div>后端: localhost:8888 → 8080</div>
          </div>
          <el-space style="margin-top:8px">
            <el-button type="primary" size="small" @click="tunnelAction('start')" :disabled="tunnel.running">连接</el-button>
            <el-button size="small" @click="tunnelAction('stop')" :disabled="!tunnel.running">断开</el-button>
          </el-space>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="hover" class="resource-card">
          <template #header>
            <div class="log-header">
              <span>系统资源</span>
              <el-button size="small" @click="loadDiskUsage" :loading="diskLoading" circle><el-icon><Refresh /></el-icon></el-button>
            </div>
          </template>
          <div class="resource-item">
            <div class="resource-label">内存 <span>{{ system.memory_used }} / {{ system.memory_total }}</span></div>
            <el-progress :percentage="memoryPercentNum" :stroke-width="8" :status="memoryPercentNum > 80 ? 'exception' : ''" />
          </div>
          <div class="resource-item">
            <div class="resource-label">磁盘 <span>{{ diskUsage.system?.used || '-' }} / {{ diskUsage.system?.total || '-' }}</span></div>
            <el-progress :percentage="diskPercentNum" :stroke-width="8" :status="diskPercentNum > 80 ? 'exception' : ''" />
          </div>
          <div class="disk-info-row">
            <div class="disk-info-item">
              <span class="label">项目</span>
              <span class="value">{{ diskUsage.project?.total || '-' }}</span>
            </div>
            <div class="disk-info-item">
              <span class="label">Docker</span>
              <span class="value">{{ dockerTotalSize }}</span>
            </div>
          </div>
          <el-divider style="margin:8px 0" />
          <el-space wrap size="small">
            <el-tooltip content="清理Docker构建缓存" placement="top">
              <el-button size="small" @click="confirmCleanDockerCache">清理缓存</el-button>
            </el-tooltip>
            <el-tooltip content="清理无用Docker镜像" placement="top">
              <el-button size="small" @click="confirmCleanDockerImages">清理镜像</el-button>
            </el-tooltip>
          </el-space>
        </el-card>
      </el-col>
    </el-row>

    <!-- 服务控制 -->
    <el-row :gutter="16" style="margin-top:16px">
      <el-col :span="24">
        <el-card shadow="hover" class="service-control-card">
          <template #header>
            <div class="service-control-header">
              <span>⚙️ 服务控制</span>
              <el-tag v-if="serviceLoading" type="warning" size="small">
                <el-icon class="is-loading"><Loading /></el-icon> 执行中...
              </el-tag>
            </div>
          </template>
          <div class="service-control-grid">
            <el-tooltip content="启动所有Docker容器" placement="top">
              <el-button type="primary" size="default" @click="serviceAction('start')" :loading="serviceLoading" class="service-btn">
                <el-icon><VideoPlay /></el-icon>
                <span>启动服务</span>
              </el-button>
            </el-tooltip>
            <el-tooltip content="重启所有Docker容器" placement="top">
              <el-button type="warning" size="default" @click="serviceAction('restart')" :loading="serviceLoading" class="service-btn">
                <el-icon><RefreshIcon /></el-icon>
                <span>重启服务</span>
              </el-button>
            </el-tooltip>
            <el-tooltip content="停止所有Docker容器" placement="top">
              <el-button type="danger" size="default" @click="confirmStop" :loading="serviceLoading" class="service-btn">
                <el-icon><SwitchButton /></el-icon>
                <span>停止服务</span>
              </el-button>
            </el-tooltip>
            <el-tooltip content="执行git pull并重启服务" placement="top">
              <el-button type="success" size="default" @click="confirmUpdate" :loading="serviceLoading" class="service-btn">
                <el-icon><Download /></el-icon>
                <span>更新代码</span>
              </el-button>
            </el-tooltip>
            <el-tooltip content="重新构建Docker镜像(--no-cache)" placement="top">
              <el-button type="info" size="default" @click="confirmRebuild" :loading="serviceLoading" class="service-btn">
                <el-icon><SetUp /></el-icon>
                <span>重建镜像</span>
              </el-button>
            </el-tooltip>
            <el-tooltip content="清理30天前的决策日志" placement="top">
              <el-button size="default" @click="confirmCleanLogs" :loading="serviceLoading" class="service-btn">
                <el-icon><Delete /></el-icon>
                <span>清理日志</span>
              </el-button>
            </el-tooltip>
            <el-tooltip content="检查Docker服务、容器状态、磁盘空间" placement="top">
              <el-button size="default" @click="runHealthCheck" :loading="serviceLoading" class="service-btn">
                <el-icon><CircleCheck /></el-icon>
                <span>健康检查</span>
              </el-button>
            </el-tooltip>
          </div>
          <el-progress v-if="serviceLoading" :percentage="serviceProgress" :stroke-width="6" style="margin-top:16px" />
          <el-alert v-if="serviceResult" :type="serviceResult.type" :title="serviceResult.message" :closable="true" @close="serviceResult = null" style="margin-top:12px" show-icon />
        </el-card>
      </el-col>
    </el-row>

    <!-- 日志查看 -->
    <el-card shadow="hover" style="margin-top:16px" class="log-card">
      <template #header>
        <div class="log-header">
          <div class="log-title">
            <span class="status-dot online"></span>
            <span>实时日志</span>
          </div>
          <div>
            <el-select v-model="logService" size="small" style="width:100px;margin-right:8px" @change="loadLogs">
              <el-option label="后端" value="backend" />
              <el-option label="前端" value="frontend" />
            </el-select>
            <el-button size="small" @click="loadLogs" circle><el-icon><Refresh /></el-icon></el-button>
          </div>
        </div>
      </template>
      <div class="log-box">
        <div class="log-line" v-for="(line, i) in logLines" :key="i">
          <span class="line-num">{{ i + 1 }}</span>
          <span class="line-content">{{ line }}</span>
        </div>
      </div>
    </el-card>

    <!-- 系统诊断 -->
    <el-card shadow="hover" style="margin-top:16px">
      <template #header>
        <div class="log-header">
          <span>系统诊断</span>
          <el-button size="small" @click="runDiagnose">运行诊断</el-button>
        </div>
      </template>
      <el-table :data="checks" size="small" v-if="checks.length">
        <el-table-column prop="name" label="检查项" width="160" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status==='ok'?'success':(row.status==='warning'?'warning':'danger')" size="small">
              {{ row.status === 'ok' ? '正常' : (row.status === 'warning' ? '警告' : '异常') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="detail" label="详情" />
      </el-table>
      <el-empty v-else description="点击运行诊断查看结果" :image-size="60" />
    </el-card>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { RefreshRight, Document, VideoPlay, Refresh as RefreshIcon, SwitchButton, Download, SetUp, Delete, CircleCheck, Loading } from '@element-plus/icons-vue'
import api from '../api'
import { useTheme } from '../stores/theme'

const { isDark } = useTheme()
const containers = ref([])
const system = ref({})
const server = ref({})
const refreshing = ref(false)
const logs = ref('')
const logService = ref('backend')
const checks = ref([])
const tunnel = ref({ running: false, pids: [] })
const autoRefresh = ref(true)
const diskUsage = ref({})
const diskLoading = ref(false)
const serviceLoading = ref(false)
const serviceProgress = ref(0)
const serviceResult = ref(null)
let refreshTimer = null
let progressTimer = null

const diskPercentNum = computed(() => {
  const pct = diskUsage.value.system?.percent?.replace('%', '') || 0
  return parseInt(pct) || 0
})

const memoryPercentNum = computed(() => {
  const used = system.value.memory_used
  const total = system.value.memory_total
  if (used && total) {
    const usedNum = parseFloat(used)
    const totalNum = parseFloat(total)
    return Math.min(Math.round((usedNum / totalNum) * 100), 100)
  }
  return 0
})

const dockerTotalSize = computed(() => {
  const docker = diskUsage.value.docker || []
  const images = docker.find(d => d.type === 'Images')
  const cache = docker.find(d => d.type === 'Build Cache')
  if (images || cache) {
    return `${images?.size || '0B'} / ${cache?.size || '0B'}`
  }
  return '-'
})

function parseMemPercent(mem) {
  if (!mem) return 0
  const match = mem.match(/([\d.]+).*\/.*?([\d.]+)/)
  if (match) {
    const used = parseFloat(match[1])
    const total = parseFloat(match[2])
    return Math.min(Math.round((used / total) * 100), 100)
  }
  return 0
}

function parseCpuPercent(cpu) {
  if (!cpu) return 0
  const match = cpu.match(/([\d.]+)%/)
  return match ? Math.min(parseFloat(match[1]), 100) : 0
}

const logLines = computed(() => {
  return logs.value ? logs.value.split('\n').filter(l => l.trim()) : []
})

async function refresh() {
  refreshing.value = true
  try {
    const res = await api.get('/status')
    containers.value = res.containers || []
    system.value = res.system || {}
    server.value = res.server || {}
  } catch (e) {
    ElMessage.error('加载失败')
  } finally {
    refreshing.value = false
  }
}

async function serviceAction(action, containerName = null) {
  // 开始执行，显示进度条
  serviceLoading.value = true
  serviceProgress.value = 0
  serviceResult.value = null

  // 模拟进度条动画
  progressTimer = setInterval(() => {
    if (serviceProgress.value < 90) {
      serviceProgress.value += 10
    }
  }, 300)

  try {
    const data = { confirm: true }
    if (containerName) data.service = containerName
    await api.post(`/service/${action}`, data)

    // 完成进度
    serviceProgress.value = 100

    // 显示成功结果
    serviceResult.value = {
      type: 'success',
      message: `✅ ${action === 'start' ? '启动' : action === 'stop' ? '停止' : action === 'restart' ? '重启' : '操作'}成功`
    }

    setTimeout(refresh, 2000)

    // 3秒后自动隐藏结果
    setTimeout(() => {
      serviceResult.value = null
    }, 3000)
  } catch (e) {
    // 显示错误结果
    serviceResult.value = {
      type: 'error',
      message: `❌ 操作失败: ${e.message || '未知错误'}`
    }
  } finally {
    // 清理进度条
    if (progressTimer) {
      clearInterval(progressTimer)
      progressTimer = null
    }
    serviceLoading.value = false
    serviceProgress.value = 0
  }
}

function confirmStop() {
  ElMessageBox.confirm('确定停止所有服务?', '警告', { type: 'warning' })
    .then(() => serviceAction('stop'))
}

function confirmUpdate() {
  ElMessageBox.confirm('确定更新代码? 将执行git pull并重启服务', '确认', { type: 'info' })
    .then(() => serviceAction('update'))
}

function confirmCleanLogs() {
  ElMessageBox.confirm('确定清理30天前的日志?', '确认', { type: 'warning' })
    .then(() => serviceAction('clean-logs'))
}

function confirmRebuild() {
  ElMessageBox.confirm('确定重建Docker镜像? 这将停止服务并重新构建', '警告', { type: 'warning' })
    .then(() => serviceAction('rebuild'))
}

async function loadTunnelStatus() {
  try {
    const res = await api.get('/tunnel/status')
    tunnel.value = res
  } catch (e) {
    tunnel.value = { running: false, pids: [] }
  }
}

async function tunnelAction(action) {
  try {
    await api.post(`/tunnel/${action}`)
    ElMessage.success(action === 'start' ? '隧道已连接' : '隧道已断开')
    loadTunnelStatus()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function runHealthCheck() {
  try {
    const res = await api.get('/service/health')
    checks.value = res.checks || []
    ElMessage.success('健康检查完成')
  } catch (e) {
    ElMessage.error('健康检查失败')
  }
}

async function loadLogs() {
  try {
    const res = await api.get(`/logs/${logService.value}?lines=50`)
    logs.value = (res.logs || []).join('\n')
    nextTick(() => {
      const box = document.querySelector('.log-box')
      if (box) box.scrollTop = box.scrollHeight
    })
  } catch (e) {
    logs.value = '加载失败'
  }
}

function showLogs(name) {
  logService.value = name.includes('trading') ? 'backend' : 'frontend'
  loadLogs()
}

async function runDiagnose() {
  try {
    const res = await api.get('/diagnose')
    checks.value = res.checks || []
  } catch (e) {
    ElMessage.error('诊断失败')
  }
}

async function loadDiskUsage() {
  diskLoading.value = true
  try {
    const res = await api.get('/disk/usage')
    diskUsage.value = res
  } catch (e) {
    ElMessage.error('加载磁盘信息失败')
  } finally {
    diskLoading.value = false
  }
}

function confirmCleanDockerCache() {
  ElMessageBox.confirm('确定清理Docker构建缓存? 可释放大量空间', '确认', { type: 'warning' })
    .then(async () => {
      try {
        const res = await api.post('/disk/clean/docker-cache', { confirm: true })
        ElMessage.success(res.output || '清理完成')
        loadDiskUsage()
      } catch (e) {
        ElMessage.error('清理失败')
      }
    })
}

function confirmCleanDockerImages() {
  ElMessageBox.confirm('确定清理无用Docker镜像?', '确认', { type: 'warning' })
    .then(async () => {
      try {
        const res = await api.post('/disk/clean/docker-images', { confirm: true })
        ElMessage.success(res.output || '清理完成')
        loadDiskUsage()
      } catch (e) {
        ElMessage.error('清理失败')
      }
    })
}

function confirmCleanOldLogs() {
  ElMessageBox.confirm('确定清理7天前的日志文件?', '确认', { type: 'warning' })
    .then(async () => {
      try {
        const res = await api.post('/disk/clean/logs', { confirm: true })
        ElMessage.success(res.output || '清理完成')
        loadDiskUsage()
      } catch (e) {
        ElMessage.error('清理失败')
      }
    })
}

onMounted(() => {
  refresh()
  loadLogs()
  loadTunnelStatus()
  loadDiskUsage()
  refreshTimer = setInterval(() => {
    if (autoRefresh.value) {
      refresh()
      loadLogs()  // 添加自动刷新日志
    }
  }, 5000)  // 改为5秒刷新一次，更及时
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<style scoped>
/* 浅色主题 */
.dashboard { padding: 20px; background: #f5f7fa; min-height: 100%; transition: all 0.3s; }
.dashboard :deep(.el-card) { transition: transform 0.2s, box-shadow 0.2s; border-radius: 8px; }
.dashboard :deep(.el-card:hover) { transform: translateY(-2px); box-shadow: 0 6px 16px rgba(0,0,0,0.1); }
.header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.header h2 { margin: 0; color: #1f2937; font-size: 24px; font-weight: 700; }
.header-actions { display: flex; align-items: center; gap: 12px; }
.server-info { margin-bottom: 20px; }
.container-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.container-title { display: flex; align-items: center; gap: 10px; }
.container-header .name { font-weight: 600; color: #1f2937; font-size: 16px; }
.status-dot { width: 10px; height: 10px; border-radius: 50%; }
.status-dot.online { background: #10b981; box-shadow: 0 0 10px #10b981; animation: pulse 2s infinite; }
.status-dot.offline { background: #ef4444; }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
.container-metrics { margin: 16px 0; }
.container-metrics .metric { display: flex; align-items: center; gap: 12px; margin-bottom: 10px; }
.container-metrics .metric-label { width: 40px; font-size: 14px; color: #6b7280; font-weight: 500; }
.container-metrics .metric-value { font-size: 13px; color: #374151; min-width: 110px; }
.container-metrics :deep(.el-progress) { flex: 1; }
.container-actions { margin-top: 16px; padding-top: 16px; border-top: 1px solid #e5e7eb; }
.log-header { display: flex; justify-content: space-between; align-items: center; font-size: 16px; font-weight: 600; }
.log-title { display: flex; align-items: center; gap: 10px; }
.log-box { background: #0d1117; color: #22d3ee; padding: 0; border-radius: 8px; height: 240px; overflow: auto; font-family: 'JetBrains Mono', 'Consolas', 'Monaco', monospace; }
.log-line { display: flex; border-bottom: 1px solid #21262d; }
.log-line:hover { background: rgba(34,211,238,0.05); }
.line-num { min-width: 45px; padding: 6px 10px; color: #6e7681; background: #161b22; text-align: right; font-size: 12px; user-select: none; border-right: 1px solid #21262d; }
.line-content { padding: 6px 14px; font-size: 13px; white-space: pre-wrap; word-break: break-all; }
.tunnel-info { font-size: 14px; color: #4b5563; line-height: 2; }

/* 深色主题 */
.dashboard.dark { background: #111827; }
.dashboard.dark .header h2 { color: #f3f4f6; }
.dashboard.dark :deep(.el-card) { background: #1f2937; border-color: #374151; }
.dashboard.dark :deep(.el-card__header) { border-color: #374151; color: #f3f4f6; }
.dashboard.dark :deep(.el-card__body) { color: #e5e7eb; }
.dashboard.dark .container-header .name { color: #f3f4f6; }
.dashboard.dark .container-metrics .metric-label { color: #9ca3af; }
.dashboard.dark .container-metrics .metric-value { color: #d1d5db; }
.dashboard.dark .container-actions { border-color: #374151; }
.dashboard.dark .tunnel-info { color: #9ca3af; }
.dashboard.dark :deep(.el-descriptions) { --el-descriptions-item-bordered-label-background: #374151; }
.dashboard.dark :deep(.el-descriptions__label) { background: #374151 !important; color: #22d3ee !important; }
.dashboard.dark :deep(.el-descriptions__content) { background: #1f2937 !important; color: #e5e7eb !important; }
.dashboard.dark :deep(.el-descriptions__cell) { border-color: #4b5563 !important; }
.dashboard.dark :deep(.el-descriptions__body) { background: #1f2937 !important; }
.dashboard.dark :deep(.el-descriptions__table) { background: #1f2937 !important; }
.dashboard.dark :deep(.el-descriptions-item__label) { color: #22d3ee !important; }
.dashboard.dark :deep(.el-descriptions-item__content) { color: #e5e7eb !important; }
.dashboard.dark :deep(.el-table) { background: #1a1a2e; }
.dashboard.dark :deep(.el-table th) { background: #16213e !important; color: #e0e0e0; }
.dashboard.dark :deep(.el-table tr) { background: #1a1a2e !important; }
.dashboard.dark :deep(.el-table td) { border-color: #2d3748; }
.dashboard.dark :deep(.el-table--border) { border-color: #2d3748; }
.dashboard.dark :deep(.el-empty__description p) { color: #a0aec0; }
.dashboard.dark :deep(.el-button--default) { background: #2d3748; border-color: #4a5568; color: #e0e0e0; }
.dashboard.dark :deep(.el-button--default:hover) { background: #4a5568; border-color: #718096; }
.dashboard.dark :deep(.el-select .el-input__wrapper) { background: #2d3748; box-shadow: 0 0 0 1px #4a5568 inset; }
.dashboard.dark :deep(.el-select .el-input__inner) { color: #e0e0e0; }
.dashboard.dark :deep(.el-tag--info) { background: #2d3748; border-color: #4a5568; color: #a0aec0; }

/* 磁盘管理样式 */
.resource-item { margin-bottom: 12px; }
.resource-label { font-size: 13px; color: #606266; margin-bottom: 6px; display: flex; justify-content: space-between; }
.disk-info-row { display: flex; justify-content: space-between; margin-top: 12px; padding: 8px; background: rgba(64,158,255,0.05); border-radius: 6px; }
.disk-info-item { text-align: center; flex: 1; }
.disk-info-item .label { font-size: 12px; color: #909399; display: block; margin-bottom: 4px; }
.disk-info-item .value { font-size: 15px; font-weight: 600; color: #409eff; }
.dashboard.dark .resource-label { color: #a0aec0; }
.dashboard.dark .disk-info-row { background: rgba(64,158,255,0.1); }
.dashboard.dark .disk-info-item .label { color: #a0aec0; }
.dashboard.dark :deep(.el-divider) { border-color: #2d3748; }
.dashboard.dark :deep(.el-progress__text) { color: #e0e0e0; }

/* 服务控制样式 */
.service-control-card { border: 2px solid #409eff; }
.service-control-header { display: flex; justify-content: space-between; align-items: center; font-size: 16px; font-weight: 600; }
.service-control-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 12px; }
.service-btn { width: 100%; height: 48px; font-size: 14px; font-weight: 600; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px; }
.service-btn :deep(.el-icon) { font-size: 18px; }
.dashboard.dark .service-control-card { border-color: #409eff; }
</style>
