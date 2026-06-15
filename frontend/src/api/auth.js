import api from './index'

export const sendCode  = (email, purpose) => api.post('/auth/send-code', { email, purpose })
export const register  = (data) => api.post('/auth/register', data)
export const login     = (payload) => api.post('/auth/login', payload)
export const logout    = () => api.post('/auth/logout')
export const getMe     = () => api.get('/users/me')
export const updateMe  = (data) => api.put('/users/me', data)
export const searchUsers    = (q) => api.get('/users/search', { params: { q } })
export const recommendUsers = () => api.get('/users/recommend')
export const getUser        = (id) => api.get(`/users/${id}`)
