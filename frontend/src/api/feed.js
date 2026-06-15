import api from './index'

export const getFeed      = (page = 1, pageSize = 20, sort = 'hot') => api.get('/posts/feed', { params: { page, page_size: pageSize, sort } })
export const searchPosts  = (q, sort = 'hot', page = 1, pageSize = 20) => api.get('/posts/search', { params: { q, sort, page, page_size: pageSize } })
export const getUserPosts = (uid, page = 1, pageSize = 20) => api.get(`/posts/user/${uid}`, { params: { page, page_size: pageSize } })
export const getPost      = (id) => api.get(`/posts/${id}`)
export const createPost   = (data) => api.post('/posts', data)
export const deletePost   = (id) => api.delete(`/posts/${id}`)
export const likePost     = (id) => api.post(`/posts/${id}/like`)
export const unlikePost   = (id) => api.delete(`/posts/${id}/like`)
export const getComments   = (id, page = 1, sort = 'hot') => api.get(`/posts/${id}/comments`, { params: { page, sort } })
export const getReplies    = (postId, cid, page = 1, pageSize = 10) => api.get(`/posts/${postId}/comments/${cid}/replies`, { params: { page, page_size: pageSize } })
export const createComment = (id, content, parentId) => api.post(`/posts/${id}/comments`, { content, parent_id: parentId })
export const likeComment   = (postId, cid) => api.post(`/posts/${postId}/comments/${cid}/like`)
export const unlikeComment = (postId, cid) => api.delete(`/posts/${postId}/comments/${cid}/like`)
export const uploadImage  = (file) => {
  const fd = new FormData(); fd.append('file', file)
  return api.post('/upload/image', fd, { headers: { 'Content-Type': 'multipart/form-data' } })
}
