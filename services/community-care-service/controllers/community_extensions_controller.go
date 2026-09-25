package controllers

import (
	"net/http"
	"strconv"

	"kather_baksho/community_care_service/database"
	"kather_baksho/community_care_service/models"

	"github.com/gin-gonic/gin"
)

// ---------- Follow ----------

// POST /api/community/follow/:user_id
func FollowUser(c *gin.Context) {
	uid := c.GetUint("user_id")
	target, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || uint(target) == uid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target"})
		return
	}
	row := models.CommunityFollow{FollowerID: uid, FollowedID: uint(target)}
	if err := database.DB.Where("follower_id = ? AND followed_id = ?", uid, target).
		FirstOrCreate(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "following": true})
}

// DELETE /api/community/follow/:user_id
func UnfollowUser(c *gin.Context) {
	uid := c.GetUint("user_id")
	target, _ := strconv.Atoi(c.Param("user_id"))
	database.DB.Where("follower_id = ? AND followed_id = ?", uid, target).
		Delete(&models.CommunityFollow{})
	c.JSON(http.StatusOK, gin.H{"ok": true, "following": false})
}

// GET /api/community/followers/:user_id
func ListFollowers(c *gin.Context) {
	target, _ := strconv.Atoi(c.Param("user_id"))
	var rows []models.CommunityFollow
	database.DB.Where("followed_id = ?", target).Find(&rows)
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.FollowerID)
	}
	c.JSON(http.StatusOK, gin.H{"follower_ids": ids, "count": len(ids)})
}

// GET /api/community/following/:user_id
func ListFollowing(c *gin.Context) {
	target, _ := strconv.Atoi(c.Param("user_id"))
	var rows []models.CommunityFollow
	database.DB.Where("follower_id = ?", target).Find(&rows)
	ids := make([]uint, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.FollowedID)
	}
	c.JSON(http.StatusOK, gin.H{"following_ids": ids, "count": len(ids)})
}

// ---------- Bookmark ----------

// POST /api/community/bookmark/:post_type/:post_id
// post_type: post | product | blog | question
func ToggleBookmark(c *gin.Context) {
	uid := c.GetUint("user_id")
	ptype := c.Param("post_type")
	pid, _ := strconv.Atoi(c.Param("post_id"))
	if pid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var existing models.CommunityBookmark
	err := database.DB.Where("user_id = ? AND post_type = ? AND post_id = ?", uid, ptype, pid).
		First(&existing).Error
	if err == nil {
		database.DB.Delete(&existing)
		c.JSON(http.StatusOK, gin.H{"bookmarked": false})
		return
	}
	database.DB.Create(&models.CommunityBookmark{
		UserID: uid, PostType: ptype, PostID: uint(pid),
	})
	c.JSON(http.StatusOK, gin.H{"bookmarked": true})
}

// GET /api/community/bookmarks
func ListBookmarks(c *gin.Context) {
	uid := c.GetUint("user_id")
	var rows []models.CommunityBookmark
	database.DB.Where("user_id = ?", uid).Order("created_at desc").Find(&rows)
	c.JSON(http.StatusOK, rows)
}

// ---------- Groups ----------

// POST /api/community/groups
func CreateGroup(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Cover       string `json:"cover"`
	}
	if err := c.BindJSON(&body); err != nil || body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	g := models.CommunityGroup{
		Name: body.Name, Description: body.Description, CoverURL: body.Cover, OwnerID: uid,
	}
	if err := database.DB.Create(&g).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// owner auto-joins
	database.DB.Create(&models.CommunityGroupMember{
		GroupID: g.ID, UserID: uid, Role: "owner",
	})
	c.JSON(http.StatusCreated, g)
}

// GET /api/community/groups
func ListGroups(c *gin.Context) {
	var list []models.CommunityGroup
	database.DB.Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// POST /api/community/groups/:id/join
func JoinGroup(c *gin.Context) {
	uid := c.GetUint("user_id")
	gid, _ := strconv.Atoi(c.Param("id"))
	row := models.CommunityGroupMember{
		GroupID: uint(gid), UserID: uid, Role: "member",
	}
	database.DB.Where("group_id = ? AND user_id = ?", gid, uid).FirstOrCreate(&row)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /api/community/groups/:id/join
func LeaveGroup(c *gin.Context) {
	uid := c.GetUint("user_id")
	gid, _ := strconv.Atoi(c.Param("id"))
	database.DB.Where("group_id = ? AND user_id = ? AND role <> ?", gid, uid, "owner").
		Delete(&models.CommunityGroupMember{})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /api/community/groups/:id/members
func ListGroupMembers(c *gin.Context) {
	gid, _ := strconv.Atoi(c.Param("id"))
	var rows []models.CommunityGroupMember
	database.DB.Where("group_id = ?", gid).Find(&rows)
	c.JSON(http.StatusOK, rows)
}

// ---------- Q&A ----------

// POST /api/community/questions
func AskQuestion(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		Title     string `json:"title"`
		Body      string `json:"body"`
		Tags      string `json:"tags"`
		ProductID uint   `json:"product_id"`
	}
	if err := c.BindJSON(&body); err != nil || body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}
	q := models.CommunityQuestion{
		UserID: uid, Title: body.Title, Body: body.Body, Tags: body.Tags,
		ProductID: body.ProductID, Status: "open",
	}
	if err := database.DB.Create(&q).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, q)
}

// GET /api/community/questions
func ListQuestions(c *gin.Context) {
	q := database.DB.Order("created_at desc")
	if s := c.Query("status"); s != "" {
		q = q.Where("status = ?", s)
	}
	if t := c.Query("tag"); t != "" {
		q = q.Where("tags LIKE ?", "%"+t+"%")
	}
	if pid := c.Query("product_id"); pid != "" {
		q = q.Where("product_id = ?", pid)
	}
	var list []models.CommunityQuestion
	q.Find(&list)
	c.JSON(http.StatusOK, list)
}

// GET /api/community/questions/:id
func GetQuestion(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var q models.CommunityQuestion
	if err := database.DB.First(&q, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var answers []models.CommunityAnswer
	database.DB.Where("question_id = ?", id).Order("created_at asc").Find(&answers)
	c.JSON(http.StatusOK, gin.H{"question": q, "answers": answers})
}

// POST /api/community/questions/:id/answer
func AnswerQuestion(c *gin.Context) {
	uid := c.GetUint("user_id")
	qid, _ := strconv.Atoi(c.Param("id"))
	var body struct {
		Body string `json:"body"`
	}
	if err := c.BindJSON(&body); err != nil || body.Body == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body required"})
		return
	}
	a := models.CommunityAnswer{
		QuestionID: uint(qid), UserID: uid, Body: body.Body,
	}
	if err := database.DB.Create(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// close question if this is the first answer
	var cnt int64
	database.DB.Model(&models.CommunityAnswer{}).Where("question_id = ?", qid).Count(&cnt)
	if cnt == 1 {
		database.DB.Model(&models.CommunityQuestion{}).Where("id = ?", qid).
			Update("status", "answered")
	}
	c.JSON(http.StatusCreated, a)
}

// PATCH /api/community/answers/:id/accept  (question owner only)
func AcceptAnswer(c *gin.Context) {
	uid := c.GetUint("user_id")
	aid, _ := strconv.Atoi(c.Param("id"))
	var a models.CommunityAnswer
	if err := database.DB.First(&a, aid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var q models.CommunityQuestion
	database.DB.First(&q, a.QuestionID)
	if q.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "only the asker can accept"})
		return
	}
	database.DB.Model(&models.CommunityAnswer{}).Where("question_id = ?", a.QuestionID).
		Update("is_accepted", false)
	database.DB.Model(&models.CommunityAnswer{}).Where("id = ?", aid).
		Update("is_accepted", true)
	database.DB.Model(&models.CommunityQuestion{}).Where("id = ?", q.ID).
		Updates(map[string]interface{}{"status": "resolved"})
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ---------- Leaderboard ----------

type leaderRow struct {
	UserID uint   `json:"user_id"`
	Name   string `json:"name"`
	Score  int    `json:"score"`
}

// GET /api/community/leaderboard?limit=10
func Leaderboard(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	scoreMap := map[uint]int{}

	var qs []models.CommunityQuestion
	database.DB.Find(&qs)
	for _, q := range qs {
		scoreMap[q.UserID] += 5
	}

	var as []models.CommunityAnswer
	database.DB.Find(&as)
	for _, a := range as {
		scoreMap[a.UserID] += 3
		if a.IsAccepted {
			scoreMap[a.UserID] += 10
		}
	}

	var fs []models.CommunityFollow
	database.DB.Find(&fs)
	for _, f := range fs {
		scoreMap[f.FollowedID] += 2
	}

	type kv struct {
		UID uint
		S   int
	}
	pairs := make([]kv, 0, len(scoreMap))
	for uid, s := range scoreMap {
		pairs = append(pairs, kv{uid, s})
	}
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].S > pairs[i].S {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	if len(pairs) > limit {
		pairs = pairs[:limit]
	}
	uids := make([]uint, 0, len(pairs))
	for _, p := range pairs {
		uids = append(uids, p.UID)
	}
	users := map[uint]string{}
	if len(uids) > 0 {
		var us []models.User
		database.DB.Where("id IN ?", uids).Find(&us)
		for _, u := range us {
			n := u.Name
			if n == "" {
				n = u.Email
			}
			users[u.ID] = n
		}
	}
	out := make([]leaderRow, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, leaderRow{UserID: p.UID, Name: users[p.UID], Score: p.S})
	}
	c.JSON(http.StatusOK, out)
}
