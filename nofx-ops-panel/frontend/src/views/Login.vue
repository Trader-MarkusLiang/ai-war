<template>
  <div class="login-container">
    <el-card class="login-card">
      <h2>NOFX 运维管理</h2>
      <el-form @submit.prevent="handleLogin">
        <el-form-item>
          <el-input v-model="password" type="password" placeholder="请输入管理密码" show-password />
        </el-form-item>
        <el-button type="primary" native-type="submit" :loading="loading" style="width:100%">
          登录
        </el-button>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import api from '../api'

const router = useRouter()
const password = ref('')
const loading = ref(false)

async function handleLogin() {
  if (!password.value) return ElMessage.warning('请输入密码')
  loading.value = true
  try {
    const res = await api.post('/auth/login', { password: password.value })
    localStorage.setItem('token', res.token)
    router.push('/')
  } catch (e) {
    ElMessage.error('密码错误')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container { height: 100vh; display: flex; align-items: center; justify-content: center; background: #2d3a4b; }
.login-card { width: 350px; padding: 20px; }
h2 { text-align: center; margin-bottom: 30px; color: #303133; }
</style>
