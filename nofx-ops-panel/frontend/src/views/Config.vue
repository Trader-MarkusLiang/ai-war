<template>
  <div>
    <h2>配置管理</h2>
    <el-card>
      <el-button @click="loadConfig">刷新</el-button>
      <el-button type="primary" @click="saveConfig" :loading="saving">保存配置</el-button>
      <el-input
        v-model="content"
        type="textarea"
        :rows="20"
        style="margin-top:15px;font-family:monospace"
      />
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api'

const content = ref('')
const saving = ref(false)

async function loadConfig() {
  const res = await api.get('/config')
  content.value = res.content
}

async function saveConfig() {
  try {
    await ElMessageBox.confirm('确定保存配置？', '确认')
    saving.value = true
    await api.put('/config', { content: content.value, confirm: true })
    ElMessage.success('保存成功，需要重启服务生效')
  } catch (e) {
    if (e !== 'cancel') ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(loadConfig)
</script>
