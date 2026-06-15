import api from './index'

export const getFriends        = () => api.get('/friends')
export const getFriendRequests = () => api.get('/friends/requests')
export const sendFriendRequest = (toUserId, message) => api.post('/friends/requests', { to_user_id: toUserId, message })
export const handleRequest     = (id, action) => api.put(`/friends/requests/${id}`, { action })
export const deleteFriend      = (friendId) => api.delete(`/friends/${friendId}`)
