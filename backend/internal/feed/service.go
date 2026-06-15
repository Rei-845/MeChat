package feed

import (
	"context"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"mechat/internal/friend"
	"mechat/internal/level"
	"mechat/internal/user"

	"github.com/redis/go-redis/v9"
)

const (
	hotKey = "feed:hot:daily" // 热榜 ZSET 增量维护互动热度
	hotTTL = 24 * time.Hour

	likeWeight    = 3.0
	commentWeight = 5.0

	// Reddit 风格热度
	decaySeconds = 45000.0 // 重力
	friendBoost  = 1.5     // 好友帖加成

	feedFriendDays  = 7   // 好友帖时间窗口
	feedFriendLimit = 100 // 好友帖数量上限
	feedPoolLimit   = 60  // 热榜与最新各取多少
)

type Service struct {
	repo      *Repository
	userRepo  *user.Repository
	friendSvc *friend.Service
	levelSvc  *level.Service
	rdb       *redis.Client
}

// 创建 Feed 服务
func NewService(repo *Repository, userRepo *user.Repository, friendSvc *friend.Service, rdb *redis.Client) *Service {
	return &Service{repo: repo, userRepo: userRepo, friendSvc: friendSvc, rdb: rdb}
}

// SetLevelSvc 注入等级服务
func (s *Service) SetLevelSvc(svc *level.Service) { s.levelSvc = svc }

// touchHot 调整热榜分值并续期
func (s *Service) touchHot(ctx context.Context, postID uint64, delta float64) {
	member := strconv.FormatUint(postID, 10)
	s.rdb.ZIncrBy(ctx, hotKey, delta, member)
	s.rdb.Expire(ctx, hotKey, hotTTL)
}

// hotIDs 取热榜前 n 个帖 ID
func (s *Service) hotIDs(ctx context.Context, n int) []uint64 {
	strs, _ := s.rdb.ZRevRange(ctx, hotKey, 0, int64(n-1)).Result()
	ids := make([]uint64, 0, len(strs))
	for _, str := range strs {
		if id, err := strconv.ParseUint(str, 10, 64); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// 发帖
func (s *Service) CreatePost(ctx context.Context, userID uint64, req *CreatePostReq, ip string) (*PostInfo, error) {
	post := &Post{
		UserID:  userID,
		Title:   req.Title,
		Content: req.Content,
		Images:  req.Images,
		IP:      ip,
	}
	if err := s.repo.CreatePost(ctx, post); err != nil {
		return nil, err
	}

	// 发帖加经验
	if s.levelSvc != nil {
		u, _ := s.userRepo.GetByID(ctx, userID)
		s.levelSvc.AddPostXP(ctx, userID, u != nil && u.IsVIP())
	}

	s.touchHot(ctx, post.ID, 0) // 加入热榜
	return s.toPostInfo(ctx, post, userID)
}

// 删帖
func (s *Service) DeletePost(ctx context.Context, userID, postID uint64) error {
	if err := s.repo.DeletePost(ctx, postID, userID); err != nil {
		return err
	}
	s.rdb.ZRem(ctx, hotKey, strconv.FormatUint(postID, 10))
	return nil
}

// scorePost Reddit 风格热度分 互动 log + 发帖时间 + 好友加成
func scorePost(p *Post, isFriend bool) float64 {
	engagement := float64(p.LikeCount)*likeWeight + float64(p.CommentCount)*commentWeight
	score := math.Log10(math.Max(engagement, 1)) + float64(p.CreatedAt.Unix())/decaySeconds
	if isFriend {
		score += friendBoost
	}
	return score
}

// 主页 Feed
func (s *Service) GetFeed(ctx context.Context, viewerID uint64, page, pageSize int, sortMode string) ([]*PostInfo, bool, error) {
	// 好友集合 含自己
	friendIDs, _ := s.friendSvc.GetFriendIDs(ctx, viewerID)
	friendSet := make(map[uint64]bool, len(friendIDs)+1)
	for _, id := range friendIDs {
		friendSet[id] = true
	}
	friendSet[viewerID] = true

	// 候选池 好友帖 + 热榜 + 全局最新 无条件纳入
	since := time.Now().AddDate(0, 0, -feedFriendDays)
	friendPosts, _ := s.repo.GetPostsByUserIDs(ctx, append(friendIDs, viewerID), since, feedFriendLimit)
	hotPosts, _ := s.repo.GetPostsByIDs(ctx, s.hotIDs(ctx, feedPoolLimit))
	recentPosts, _ := s.repo.GetRecentPosts(ctx, 1, feedPoolLimit)
	merged := mergePosts(friendPosts, hotPosts, recentPosts)

	// 先把分算好再排 比较函数里不重复算 log
	type scored struct {
		post  *Post
		score float64
	}
	list := make([]scored, len(merged))
	for i, p := range merged {
		list[i] = scored{p, scorePost(p, friendSet[p.UserID])}
	}
	if sortMode == "time" {
		sort.Slice(list, func(i, j int) bool { return list[i].post.CreatedAt.After(list[j].post.CreatedAt) })
	} else {
		sort.Slice(list, func(i, j int) bool { return list[i].score > list[j].score })
	}

	// 分页
	start := (page - 1) * pageSize
	if start >= len(list) {
		return []*PostInfo{}, false, nil
	}
	end := min(start+pageSize, len(list))
	pagePosts := list[start:end]

	// 批量组装消除 N+1
	pageList := make([]*Post, len(pagePosts))
	for i, sp := range pagePosts {
		pageList[i] = sp.post
	}
	return s.buildPostInfos(ctx, pageList, viewerID, friendSet), end < len(list), nil
}

// SearchPosts 按标题搜索
func (s *Service) SearchPosts(ctx context.Context, viewerID uint64, keyword, sortMode string, page, pageSize int) ([]*PostInfo, bool, error) {
	posts, err := s.repo.SearchPostsByTitle(ctx, keyword, sortMode, page, pageSize+1)
	if err != nil {
		return nil, false, err
	}
	posts, hasMore := peek(posts, pageSize)
	friendIDs, _ := s.friendSvc.GetFriendIDs(ctx, viewerID)
	friendSet := make(map[uint64]bool, len(friendIDs))
	for _, id := range friendIDs {
		friendSet[id] = true
	}
	return s.buildPostInfos(ctx, posts, viewerID, friendSet), hasMore, nil
}

// GetHotPosts 热榜 空时回退最新
func (s *Service) GetHotPosts(ctx context.Context, viewerID uint64) ([]*PostInfo, error) {
	posts, _ := s.repo.GetPostsByIDs(ctx, s.hotIDs(ctx, 20))
	if len(posts) == 0 {
		posts, _ = s.repo.GetRecentPosts(ctx, 1, 20)
	}
	return s.buildPostInfos(ctx, posts, viewerID, nil), nil
}

// 单帖
func (s *Service) GetPost(ctx context.Context, postID, viewerID uint64) (*PostInfo, error) {
	post, err := s.repo.GetPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	return s.toPostInfo(ctx, post, viewerID)
}

// 点赞帖子
func (s *Service) LikePost(ctx context.Context, viewerID, postID uint64) error {
	if err := s.repo.LikePost(ctx, postID, viewerID); err != nil {
		return err
	}
	s.touchHot(ctx, postID, likeWeight)
	return nil
}

// 取消点赞
func (s *Service) UnlikePost(ctx context.Context, viewerID, postID uint64) error {
	if err := s.repo.UnlikePost(ctx, postID, viewerID); err != nil {
		return err
	}
	s.touchHot(ctx, postID, -likeWeight)
	return nil
}

// CreateComment 创建评论 返回实得经验
func (s *Service) CreateComment(ctx context.Context, userID, postID uint64, req *CreateCommentReq) (*CommentInfo, int, error) {
	comment := &PostComment{
		PostID:   postID,
		UserID:   userID,
		ParentID: req.ParentID,
		Content:  req.Content,
	}
	if err := s.repo.CreateComment(ctx, comment); err != nil {
		return nil, 0, err
	}
	s.touchHot(ctx, postID, commentWeight)

	u, _ := s.userRepo.GetByID(ctx, userID)

	// 评论加经验 有每日上限
	xpGained := 0
	if s.levelSvc != nil && u != nil {
		xpGained = s.levelSvc.AddCommentXP(ctx, userID, u.IsVIP())
	}

	info := &CommentInfo{
		ID:        comment.ID,
		PostID:    postID,
		ParentID:  req.ParentID,
		Content:   req.Content,
		CreatedAt: comment.CreatedAt,
	}
	if u != nil {
		info.User = AuthorInfo{ID: u.ID, Nickname: u.Nickname, AvatarURL: u.AvatarURL, VIP: u.IsVIP(), Level: u.Level(), Tier: u.Tier()}
	}
	return info, xpGained, nil
}

// 点赞评论
func (s *Service) LikeComment(ctx context.Context, viewerID, commentID uint64) error {
	return s.repo.LikeComment(ctx, commentID, viewerID)
}

// 取消评论点赞
func (s *Service) UnlikeComment(ctx context.Context, viewerID, commentID uint64) error {
	return s.repo.UnlikeComment(ctx, commentID, viewerID)
}

// GetUserPosts 目标用户的帖子
func (s *Service) GetUserPosts(ctx context.Context, viewerID, targetID uint64, page, pageSize int) ([]*PostInfo, bool, error) {
	posts, err := s.repo.GetPostsByUser(ctx, targetID, page, pageSize, pageSize+1)
	if err != nil {
		return nil, false, err
	}
	posts, hasMore := peek(posts, pageSize)
	// 同一人 好友只判一次
	var friendSet map[uint64]bool
	if viewerID != targetID && s.isFriend(ctx, viewerID, targetID) {
		friendSet = map[uint64]bool{targetID: true}
	}
	return s.buildPostInfos(ctx, posts, viewerID, friendSet), hasMore, nil
}

// GetMyPosts 我的帖子
func (s *Service) GetMyPosts(ctx context.Context, userID uint64, page, pageSize int) ([]*PostInfo, bool, error) {
	posts, err := s.repo.GetPostsByUser(ctx, userID, page, pageSize, pageSize+1)
	if err != nil {
		return nil, false, err
	}
	posts, hasMore := peek(posts, pageSize)
	return s.buildPostInfos(ctx, posts, userID, nil), hasMore, nil
}

const replyPreviewLimit = 3 // 根评论默认展示回复数

// 评论列表 含回复预览
func (s *Service) GetComments(ctx context.Context, viewerID, postID uint64, page, pageSize int, sortMode string) ([]*CommentInfo, bool, error) {
	// 多取一条判断 has_more
	comments, err := s.repo.GetComments(ctx, postID, page, pageSize+1, sortMode)
	if err != nil {
		return nil, false, err
	}
	comments, hasMore := peek(comments, pageSize)

	// 预取回复
	allComments := make([]*PostComment, 0, len(comments)*4)
	replyMap := make(map[uint64][]*PostComment, len(comments))
	hasMoreReplyMap := make(map[uint64]bool, len(comments))
	for _, c := range comments {
		allComments = append(allComments, c)
		replies, _ := s.repo.GetReplies(ctx, c.ID, replyPreviewLimit+1)
		if len(replies) > replyPreviewLimit {
			hasMoreReplyMap[c.ID] = true
			replies = replies[:replyPreviewLimit]
		}
		replyMap[c.ID] = replies
		allComments = append(allComments, replies...)
	}
	allCIDs := make([]uint64, len(allComments))
	authorIDs := make([]uint64, len(allComments))
	for i, c := range allComments {
		allCIDs[i] = c.ID
		authorIDs[i] = c.UserID
	}
	likedMap, _ := s.repo.BatchCommentLiked(ctx, allCIDs, viewerID)
	authors := s.usersByID(ctx, authorIDs) // 批量预取作者

	result := make([]*CommentInfo, 0, len(comments))
	for _, c := range comments {
		info := buildCommentInfo(c, authors, likedMap)
		info.HasMoreReplies = hasMoreReplyMap[c.ID]
		for _, r := range replyMap[c.ID] {
			info.Replies = append(info.Replies, buildCommentInfo(r, authors, likedMap))
		}
		result = append(result, &info)
	}
	return result, hasMore, nil
}

// GetCommentReplies 分页取回复
func (s *Service) GetCommentReplies(ctx context.Context, viewerID, postID, commentID uint64, page, pageSize int) ([]CommentInfo, bool, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	// 多取一条判断下一页
	replies, err := s.repo.GetRepliesPaged(ctx, commentID, offset, pageSize+1)
	if err != nil {
		return nil, false, err
	}
	replies, hasMore := peek(replies, pageSize)

	ids := make([]uint64, len(replies))
	authorIDs := make([]uint64, len(replies))
	for i, r := range replies {
		ids[i] = r.ID
		authorIDs[i] = r.UserID
	}
	likedMap, _ := s.repo.BatchCommentLiked(ctx, ids, viewerID)
	authors := s.usersByID(ctx, authorIDs)

	result := make([]CommentInfo, 0, len(replies))
	for _, r := range replies {
		result = append(result, buildCommentInfo(r, authors, likedMap))
	}
	return result, hasMore, nil
}

// 删评论
func (s *Service) DeleteComment(ctx context.Context, userID, postID, commentID uint64) error {
	return s.repo.DeleteComment(ctx, commentID, userID)
}

// buildPostInfo 纯组装 不查库
func buildPostInfo(p *Post, author *user.User, liked bool) *PostInfo {
	info := &PostInfo{
		PostID:       p.ID,
		Title:        p.Title,
		Content:      p.Content,
		Images:       p.Images,
		IP:           maskIP(p.IP), // 脱敏 IP
		LikeCount:    p.LikeCount,
		CommentCount: p.CommentCount,
		CreatedAt:    p.CreatedAt,
		IsLiked:      liked,
	}
	// 旧帖无标题用 ID 兜底
	if info.Title == "" {
		info.Title = "帖子 #" + strconv.FormatUint(p.ID, 10)
	}
	if author != nil {
		info.User = authorOf(author)
	}
	return info
}

// authorOf 提取作者信息
func authorOf(u *user.User) AuthorInfo {
	return AuthorInfo{ID: u.ID, Nickname: u.Nickname, AvatarURL: u.AvatarURL, VIP: u.IsVIP(), Level: u.Level(), Tier: u.Tier()}
}

// toPostInfo 单篇组装 自带查询
func (s *Service) toPostInfo(ctx context.Context, p *Post, viewerID uint64) (*PostInfo, error) {
	u, _ := s.userRepo.GetByID(ctx, p.UserID)
	liked, _ := s.repo.IsLiked(ctx, p.ID, viewerID)
	return buildPostInfo(p, u, liked), nil
}

// buildPostInfos 批量组装 消除 N+1
func (s *Service) buildPostInfos(ctx context.Context, posts []*Post, viewerID uint64, friendSet map[uint64]bool) []*PostInfo {
	ids := make([]uint64, len(posts))
	authorIDs := make([]uint64, len(posts))
	for i, p := range posts {
		ids[i] = p.ID
		authorIDs[i] = p.UserID
	}
	likedMap, _ := s.repo.BatchIsLiked(ctx, ids, viewerID)
	authors := s.usersByID(ctx, authorIDs)

	result := make([]*PostInfo, 0, len(posts))
	for _, p := range posts {
		info := buildPostInfo(p, authors[p.UserID], likedMap[p.ID])
		info.IsFriend = p.UserID != viewerID && friendSet[p.UserID]
		result = append(result, info)
	}
	return result
}

// isFriend 是否好友
func (s *Service) isFriend(ctx context.Context, viewerID, targetID uint64) bool {
	ids, _ := s.friendSvc.GetFriendIDs(ctx, viewerID)
	return slices.Contains(ids, targetID)
}

// usersByID 批量查用户 返回 id->User
func (s *Service) usersByID(ctx context.Context, ids []uint64) map[uint64]*user.User {
	m := make(map[uint64]*user.User, len(ids))
	if len(ids) == 0 {
		return m
	}
	users, _ := s.userRepo.GetByIDs(ctx, ids)
	for _, u := range users {
		m[u.ID] = u
	}
	return m
}

// mergePosts 合并去重
func mergePosts(groups ...[]*Post) []*Post {
	seen := make(map[uint64]bool)
	result := make([]*Post, 0)
	for _, g := range groups {
		for _, p := range g {
			if !seen[p.ID] {
				seen[p.ID] = true
				result = append(result, p)
			}
		}
	}
	return result
}

// buildCommentInfo 组装评论信息
func buildCommentInfo(c *PostComment, authors map[uint64]*user.User, likedMap map[uint64]bool) CommentInfo {
	info := CommentInfo{
		ID: c.ID, PostID: c.PostID, ParentID: c.ParentID,
		Content: c.Content, CreatedAt: c.CreatedAt,
		LikeCount: c.LikeCount, IsLiked: likedMap[c.ID],
	}
	if u := authors[c.UserID]; u != nil {
		info.User = authorOf(u)
	}
	return info
}

// peek 多取一条式分页 裁剪并返回 hasMore
func peek[T any](items []T, pageSize int) ([]T, bool) {
	if len(items) > pageSize {
		return items[:pageSize], true
	}
	return items, false
}

// maskIP 脱敏只留前两段
func maskIP(ip string) string {
	if ip == "" {
		return ""
	}
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + ".*.*"
	}
	// IPv6 只留第一段
	if idx := strings.Index(ip, ":"); idx > 0 {
		return ip[:idx] + ":*"
	}
	return "*.*.*.*"
}
