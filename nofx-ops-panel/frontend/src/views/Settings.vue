<template>
  <div class="settings-page">
    <h2>系统设置</h2>

    <!-- 密码修改 -->
    <el-card class="settings-card">
      <template #header>
        <div class="card-header">
          <el-icon><Lock /></el-icon>
          <span>修改密码</span>
        </div>
      </template>
      <el-form :model="passwordForm" label-width="100px">
        <el-form-item label="原密码">
          <el-input v-model="passwordForm.old_password" type="password" show-password placeholder="请输入原密码" />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="passwordForm.new_password" type="password" show-password placeholder="请输入新密码（至少6位）" />
        </el-form-item>
        <el-form-item label="确认密码">
          <el-input v-model="passwordForm.confirm_password" type="password" show-password placeholder="请再次输入新密码" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="changePassword" :loading="passwordLoading">修改密码</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 服务器配置 -->
    <el-card class="settings-card">
      <template #header>
        <div class="card-header">
          <el-icon><Connection /></el-icon>
          <span>服务器配置</span>
        </div>
      </template>
      <el-form :model="serverForm" label-width="120px">
        <el-form-item label="远程服务器IP">
          <el-input v-model="serverForm.remote_host" placeholder="例如: 47.236.159.60" />
        </el-form-item>
        <el-form-item label="SSH用户名">
          <el-input v-model="serverForm.remote_user" placeholder="例如: root" />
        </el-form-item>
        <el-form-item label="远程目录">
          <el-input v-model="serverForm.remote_dir" placeholder="例如: /opt/nofx" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveServerSettings" :loading="serverLoading">保存配置</el-button>
          <el-tag type="warning" style="margin-left: 10px">修改后需重启服务生效</el-tag>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- API配置 -->
    <el-card class="settings-card">
      <template #header>
        <div class="card-header">
          <el-icon><Setting /></el-icon>
          <span>API配置</span>
        </div>
      </template>
      <el-form :model="apiForm" label-width="120px">
        <el-form-item label="API端口">
          <el-input-number v-model="apiForm.api_port" :min="1024" :max="65535" />
        </el-form-item>
        <el-form-item label="Token有效期">
          <el-input-number v-model="apiForm.jwt_expire_hours" :min="1" :max="720" />
          <span style="margin-left: 10px; color: #909399">小时</span>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="saveApiSettings" :loading="apiLoading">保存配置</el-button>
          <el-tag type="warning" style="margin-left: 10px">修改后需重启服务生效</el-tag>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Lock, Connection, Setting } from '@element-plus/icons-vue'
import api from '../api'

const passwordForm = ref({ old_password: '', new_password: '', confirm_password: '' })
const passwordLoading = ref(false)
const serverForm = ref({ remote_host: '', remote_user: '', remote_dir: '' })
const serverLoading = ref(false)
const apiForm = ref({ api_port: 8800, jwt_expire_hours: 24 })
const apiLoading = ref(false)

async function loadSettings() {
  try {
    const res = await api.get('/settings')
    serverForm.value.remote_host = res.remote_host
    serverForm.value.remote_user = res.remote_user
    serverForm.value.remote_dir = res.remote_dir
    apiForm.value.api_port = res.api_port
    apiForm.value.jwt_expire_hours = res.jwt_expire_hours
  } catch (e) {
    ElMessage.error('加载设置失败')
  }
}

async function changePassword() {
  if (passwordForm.value.new_password !== passwordForm.value.confirm_password) {
    ElMessage.error('两次输入的密码不一致')
    return
  }
  passwordLoading.value = true
  try {
    await api.post('/settings/password', {
      old_password: passwordForm.value.old_password,
      new_password: passwordForm.value.new_password
    })
    ElMessage.success('密码修改成功，重启服务后生效')
    passwordForm.value = { old_password: '', new_password: '', confirm_password: '' }
  } catch (e) {
    ElMessage.error(e.response?.data?.detail || '修改失败')
  } finally {
    passwordLoading.value = false
  }
}

async function saveServerSettings() {
  serverLoading.value = true
  try {
    await api.put('/settings', serverForm.value)
    ElMessage.success('保存成功，重启服务后生效')
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    serverLoading.value = false
  }
}

async function saveApiSettings() {
  apiLoading.value = true
  try {
    await api.put('/settings', apiForm.value)
    ElMessage.success('保存成功，重启服务后生效')
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    apiLoading.value = false
  }
}

onMounted(loadSettings)
</script>

<style scoped>
.settings-page { padding: 20px; }
h2 { margin-bottom: 20px; color: #303133; }
.settings-card { margin-bottom: 20px; }
.card-header { display: flex; align-items: center; gap: 8px; font-weight: 500; }
.el-form { max-width: 500px; }
</style>
