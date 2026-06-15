import api from './index'

export const getConversations   = () => api.get('/conversations')
export const createPrivateConv  = (targetUserId) => api.post('/conversations', { target_user_id: targetUserId })
export const createGroup        = (data) => api.post('/conversations/group', data)
export const getMessages        = (convId, params) => api.get(`/conversations/${convId}/messages`, { params })
export const markRead           = (convId, msgId) => api.put(`/conversations/${convId}/read`, { msg_id: msgId })
export const recallMessage      = (msgId) => api.post(`/messages/${msgId}/recall`)
export const addGroupMembers    = (convId, userIds) => api.post(`/conversations/${convId}/members`, { user_ids: userIds })
export const removeGroupMember  = (convId, uid) => api.delete(`/conversations/${convId}/members/${uid}`)
export const leaveGroup         = (convId) => api.delete(`/conversations/${convId}/members/me`)
export const disbandGroup       = (convId) => api.post(`/conversations/${convId}/disband`)
export const getGroupMembers    = (groupId) => api.get(`/groups/${groupId}/members`)
export const uploadGroupAvatar  = (groupId, file) => {
  const fd = new FormData(); fd.append('file', file)
  return api.post(`/groups/${groupId}/avatar`, fd, { headers: { 'Content-Type': 'multipart/form-data' } })
}
