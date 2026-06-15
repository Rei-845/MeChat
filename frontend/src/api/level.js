import api from './index'

export const getMyLevel = () => api.get('/level/me')
export const checkin    = () => api.post('/level/checkin')
