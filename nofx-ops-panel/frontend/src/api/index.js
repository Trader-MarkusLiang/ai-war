import axios from 'axios'
import { setLoading, setError } from '../stores/global'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
api.interceptors.request.use(
  config => {
    setLoading(true)
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  error => {
    setLoading(false)
    setError('请求失败')
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  res => {
    setLoading(false)
    return res.data
  },
  err => {
    setLoading(false)
    const status = err.response?.status
    const message = err.response?.data?.error?.message || err.message

    if (status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    } else if (status === 429) {
      setError('请求过于频繁，请稍后再试')
    } else if (status >= 500) {
      setError('服务器错误: ' + message)
    } else if (err.code === 'ECONNABORTED') {
      setError('请求超时')
    } else {
      setError(message || '请求失败')
    }

    return Promise.reject(err)
  }
)

export default api
