import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  },
  // Enable credentials for cookie-based authentication
  withCredentials: true
})

// Request interceptor
api.interceptors.request.use(
  config => {
    return config
  },
  error => Promise.reject(error)
)

// Response interceptor
api.interceptors.response.use(
  response => response.data,
  error => {
    // Handle 401 Unauthorized - redirect to login
    if (error.response?.status === 401) {
      // Check if not already on login page
      if (window.location.pathname !== '/login') {
        // Store the current path to redirect back after login
        sessionStorage.setItem('redirectAfterLogin', window.location.pathname)
        window.location.href = '/login'
      }
    }
    console.error('API Error:', error.response?.data || error.message)
    return Promise.reject(error)
  }
)

export default api
