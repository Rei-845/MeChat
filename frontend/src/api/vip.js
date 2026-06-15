import api from './index'

export const getPlans   = () => api.get('/vip/plans')
export const createOrder = (plan) => api.post('/vip/orders', { plan })
export const payOrder   = (id) => api.post(`/vip/orders/${id}/pay`)
export const getStatus  = () => api.get('/vip/status')
