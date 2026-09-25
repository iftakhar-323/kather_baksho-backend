package controllers

import (
	"net/http"
	"strconv"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"

	"github.com/gin-gonic/gin"
)

type CreatePostInput struct {
	Title    string `json:"title" binding:"required"`
	Body     string `json:"body" binding:"required"`
	ImageURL string `json:"image_url"`
	Category string `json:"category"`
}

// GET /api/community/posts
func ListPosts(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	var posts []models.CommunityPost
	database.DB.Order("created_at desc").Limit(100).Find(&posts)

	// enrich with like-count + comment-count
	type PostOut struct {
		models.CommunityPost
		LikeCount    int64    `json:"like_count"`
		CommentCount int64    `json:"comment_count"`
		LikedByMe    bool     `json:"liked_by_me"`
		LikedByNames []string `json:"liked_by_names"`
	}
	out := make([]PostOut, 0, len(posts))
	uid := c.GetUint("user_id")
	for _, p := range posts {
		if p.Author == "" || p.Author == "Anonymous" {
			var u models.User
			if err := database.DB.First(&u, p.UserID).Error; err == nil && u.Name != "" {
				p.Author = u.Name
			}
		}

		var lc int64
		var cc int64
		var mine int64
		database.DB.Model(&models.CommunityLike{}).Where("post_id = ?", p.ID).Count(&lc)
		database.DB.Model(&models.CommunityComment{}).Where("post_id = ?", p.ID).Count(&cc)
		database.DB.Model(&models.CommunityLike{}).Where("post_id = ? AND user_id = ?", p.ID, uid).Count(&mine)

		var likeUsers []models.User
		database.DB.Joins("JOIN community_likes ON community_likes.user_id = users.id").
			Where("community_likes.post_id = ?", p.ID).Find(&likeUsers)
		var likedByNames []string
		for _, lu := range likeUsers {
			if lu.Name != "" {
				likedByNames = append(likedByNames, lu.Name)
			} else {
				likedByNames = append(likedByNames, lu.Email)
			}
		}

		out = append(out, PostOut{
			CommunityPost: p,
			LikeCount:     lc,
			CommentCount:  cc,
			LikedByMe:     mine > 0,
			LikedByNames:  likedByNames,
		})
	}
	c.JSON(http.StatusOK, out)
}

// POST /api/community/posts
func CreatePost(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	userID := c.GetUint("user_id")
	var input CreatePostInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	author := ""
	var user models.User
	if err := database.DB.Select("name").First(&user, userID).Error; err == nil {
		author = user.Name
	}
	if input.Category == "" {
		input.Category = "story"
	}
	post := models.CommunityPost{
		UserID:   userID,
		Author:   author,
		Title:    input.Title,
		Body:     input.Body,
		ImageURL: input.ImageURL,
		Category: input.Category,
	}
	database.DB.Create(&post)
	c.JSON(http.StatusCreated, post)
}

type CommentInput struct {
	Body string `json:"body" binding:"required"`
}

// GET /api/community/posts/:id/comments
func ListComments(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	id := c.Param("id")
	var list []models.CommunityComment
	database.DB.Where("post_id = ?", id).Order("created_at asc").Find(&list)
	for i := range list {
		if list[i].Author == "" || list[i].Author == "Anonymous" {
			var u models.User
			if err := database.DB.First(&u, list[i].UserID).Error; err == nil && u.Name != "" {
				list[i].Author = u.Name
			}
		}
	}
	c.JSON(http.StatusOK, list)
}

// POST /api/community/posts/:id/comments
func AddComment(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	id := c.Param("id")
	userID := c.GetUint("user_id")
	var input CommentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	author := ""
	var user models.User
	if err := database.DB.Select("name").First(&user, userID).Error; err == nil {
		author = user.Name
	}
	postID, _ := strconv.Atoi(id)
	comm := models.CommunityComment{
		PostID: uint(postID),
		UserID: userID,
		Author: author,
		Body:   input.Body,
	}
	database.DB.Create(&comm)
	c.JSON(http.StatusCreated, comm)
}

// POST /api/community/posts/:id/like
func ToggleLike(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	id := c.Param("id")
	userID := c.GetUint("user_id")
	postID, _ := strconv.Atoi(id)

	// check existing
	var existing models.CommunityLike
	err := database.DB.Where("post_id = ? AND user_id = ?", postID, userID).First(&existing).Error
	if err == nil {
		// already exists → unlike
		database.DB.Delete(&existing)
		c.JSON(http.StatusOK, gin.H{"liked": false})
		return
	}
	like := models.CommunityLike{PostID: uint(postID), UserID: userID}
	database.DB.Create(&like)
	c.JSON(http.StatusOK, gin.H{"liked": true})
}

// DELETE /api/community/posts/:id  (author or admin)
func DeletePost(c *gin.Context) {
	if utils.ProxyToService(c, "COMMUNITY_CARE_SERVICE_URL") {
		return
	}
	id := c.Param("id")

	userID := c.GetUint("user_id")
	role := c.GetString("role")

	var p models.CommunityPost
	if err := database.DB.First(&p, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}
	if role != "admin" && p.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not allowed"})
		return
	}
	database.DB.Where("post_id = ?", id).Delete(&models.CommunityComment{})
	database.DB.Where("post_id = ?", id).Delete(&models.CommunityLike{})
	database.DB.Delete(&p)
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}
