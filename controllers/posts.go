package controllers

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mfjkri/OneNUS-Backend/database"
	"github.com/mfjkri/OneNUS-Backend/models"
	"github.com/mfjkri/OneNUS-Backend/utils"
	"gorm.io/gorm"
)

/* -------------------------------------------------------------------------- */
/*                              Helper functions                              */
/* -------------------------------------------------------------------------- */
func verifyTag(tag string) (valid bool) {
	valid = false
	for _, x := range models.ValidTags {
		if x == tag {
			valid = true
		}
	}
	return
}

type PostResponse struct {
	ID            uint   `json:"id" binding:"required"`
	Title         string `json:"title" binding:"required"`
	Tag           string `json:"tag" binding:"required"`
	Text          string `json:"text" binding:"required"`
	Author        string `json:"author" binding:"required"`
	UserID        uint   `json:"userId" binding:"required"`
	CommentsCount uint   `json:"commentsCount" binding:"required"`
	CommentedAt   int64  `json:"commentedAt" binding:"required"`
	StarsCount    uint   `json:"starsCount" binding:"required"`
	CreatedAt     int64  `json:"createdAt" binding:"required"`
	UpdatedAt     int64  `json:"updatedAt" binding:"required"`
}

// Convert a Post Model into a JSON format
func CreatePostResponse(post *models.Post) PostResponse {
	return PostResponse{
		ID:            post.ID,
		Title:         post.Title,
		Tag:           post.Tag,
		Text:          post.Text,
		Author:        post.Author,
		UserID:        post.UserID,
		CommentsCount: post.CommentsCount,
		CommentedAt:   post.CommentedAt.Unix(),
		StarsCount:    post.StarsCount,
		CreatedAt:     post.CreatedAt.Unix(),
		UpdatedAt:     post.UpdatedAt.Unix(),
	}
}

type GetPostsResponse struct {
	Posts      []PostResponse `json:"posts" binding:"required"`
	PostsCount int64          `json:"postsCount" binding:"required"`
}

// Bundles and convert multiple Post models into a JSON format
func CreatePostsResponse(posts *[]models.Post, totalPostsCount int64) GetPostsResponse {
	var postsResponse []PostResponse
	for _, post := range *posts {
		postReponse := CreatePostResponse(&post)
		postsResponse = append(postsResponse, postReponse)
	}

	return GetPostsResponse{
		Posts:      postsResponse,
		PostsCount: totalPostsCount,
	}
}

// Fetches posts based on provided configuration
func GetPostsFromContext(dbContext *gorm.DB, perPage uint, pageNumber uint, sortOption string, sortOrder string) ([]models.Post, int64) {
	var posts []models.Post

	// Limit PerPage to MAX_PER_PAGE
	clampedPerPage := int64(math.Min(MAX_PER_PAGE, float64(perPage)))
	offsetPostsCount := int64(pageNumber-1) * clampedPerPage

	// Get total count for Posts
	var totalPostsCount int64
	dbContext.Count(&totalPostsCount)

	// If we are request beyond the bounds of total count, error
	if (offsetPostsCount < 0) || (offsetPostsCount > totalPostsCount) {
		return posts, 0
	}

	// Sort Posts by sort option provided (defaults to byNew)
	defaultSortOption := ByNew
	if sortOption == "recent" {
		defaultSortOption = ByRecent
	} else if sortOption == "hot" {
		defaultSortOption = ByHot
	}

	// Fetch Posts from [offsetCount, offsetCount + perPage]
	// results order depends on SortOption and SortOrder
	if sortOrder == "ascending" {
		// Reverse page number based on totalPostsCount
		leftOverRecords := math.Min(float64(clampedPerPage), float64(totalPostsCount-offsetPostsCount))
		offsetPostsCount = totalPostsCount - offsetPostsCount - clampedPerPage
		dbContext.Limit(int(leftOverRecords)).Order(defaultSortOption).Offset(int(offsetPostsCount)).Find(&posts)

		// Reverse the page results for descending order
		for i, j := 0, len(posts)-1; i < j; i, j = i+1, j-1 {
			posts[i], posts[j] = posts[j], posts[i]
		}
	} else {
		dbContext.Limit(int(clampedPerPage)).Order(defaultSortOption).Offset(int(offsetPostsCount)).Find(&posts)
	}

	return posts, totalPostsCount
}

/* -------------------------------------------------------------------------- */
/*                            GetPosts | route: ...                           */
/* -------------------------------------------------------------------------- */
// route: /posts/get/:perPage/:pageNumber/:sortBy/:filterUserId/:filterTag
type GetPostsRequest struct {
	PerPage      uint   `uri:"perPage" binding:"required"`
	PageNumber   uint   `uri:"pageNumber" binding:"required"`
	SortOption   string `uri:"sortOption"`
	SortOrder    string `uri:"sortOrder"`
	FilterUserID uint   `uri:"filterUserId"`
	FilterTag    string `uri:"filterTag"`
}

func GetPosts(c *gin.Context) {
	// Check that RequestUser is authenticated
	_, found := VerifyAuth(c)
	if found == false {
		return
	}

	// Parse RequestBody
	var json GetPostsRequest
	if err := c.ShouldBindUri(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	dbContext := database.DB.Table("posts")

	// Filter database by UserID (if any)
	if json.FilterUserID != 0 {
		targetUser, found := FindUserFromID(c, json.FilterUserID)
		if found == false {
			return
		} else {
			dbContext = dbContext.Where("user_id = ?", targetUser.ID)
		}
	}

	// Filter database by FilterTag (if any)
	if verifyTag(json.FilterTag) {
		dbContext = dbContext.Where("tag = ?", json.FilterTag)
	}

	// Fetch posts
	posts, totalPostsCount := GetPostsFromContext(dbContext, json.PerPage, json.PageNumber, json.SortOption, json.SortOrder)

	// Return fetched posts
	c.JSON(http.StatusAccepted, CreatePostsResponse(&posts, totalPostsCount))
}

/* -------------------------------------------------------------------------- */
/*                GetPostByID | route : /posts/getbyid/:postId                */
/* -------------------------------------------------------------------------- */
type GetPostByIDRequest struct {
	PostID uint `uri:"postId" binding:"required"`
}

func GetPostByID(c *gin.Context) {
	// Check that RequestUser is authenticated
	_, found := VerifyAuth(c)
	if found == false {
		return
	}

	// Parse RequestBody
	var json GetPostByIDRequest
	if err := c.ShouldBindUri(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Find Post from PostID
	var post models.Post
	database.DB.First(&post, json.PostID)
	if post.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		return
	}

	// Return fetched Post
	c.JSON(http.StatusAccepted, CreatePostResponse(&post))
}

/* -------------------------------------------------------------------------- */
/*                        CreatePost | route: /post/get                       */
/* -------------------------------------------------------------------------- */
type CreatePostRequest struct {
	Title string `json:"title" binding:"required"`
	Tag   string `json:"tag" binding:"required"`
	Text  string `json:"text" binding:"required"`
}

func CreatePost(c *gin.Context) {
	// Check that RequestUser is authenticated
	user, found := VerifyAuth(c)
	if found == false {
		return
	}

	// Parse RequestBody
	var json CreatePostRequest
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Prevent frequent CreatePosts by User
	timeNow, canCreatePost := utils.CheckTimeIsAfter(user.LastPostAt, USER_POST_COOLDOWN)
	if canCreatePost == false {
		cdLeft := utils.GetCooldownLeft(user.LastPostAt, USER_POST_COOLDOWN, timeNow)
		c.JSON(http.StatusForbidden, gin.H{"message": fmt.Sprintf("Creating posts too frequently. Please try again in %ds", int(cdLeft.Seconds()))})
		return
	}

	// Check that Title and Text does not contain illegal characters
	if !(utils.ContainsValidCharactersOnly(json.Title) && utils.ContainsValidCharactersOnly(json.Text)) {
		c.JSON(http.StatusForbidden, gin.H{"message": "Title or Body contains illegal characters."})
		return
	}

	// Check that the Tag provided is valid
	validTag := verifyTag(json.Tag)
	if validTag == false {
		c.JSON(http.StatusForbidden, gin.H{"message": "Unknown tag for post."})
		return
	}

	// Try to create new Post
	post := models.Post{
		Title:         utils.TrimString(strings.TrimSpace(json.Title), MAX_POST_TITLE_CHAR),
		Tag:           json.Tag,
		Text:          utils.TrimString(strings.TrimSpace(json.Text), MAX_POST_TEXT_CHAR),
		Author:        user.Username,
		User:          user,
		CommentsCount: 0,
		CommentedAt:   time.Unix(0, 0),
		StarsCount:    0,
	}
	new_entry := database.DB.Create(&post)

	// Failed to create entry
	if new_entry.Error != nil {
		c.JSON(http.StatusBadGateway, gin.H{"message": "Unable to create post. Try again later."})
		return
	}

	// Successfully created a new Post

	// Update PostsCount and LastPostAt for User
	user.PostsCount += 1
	user.LastPostAt = timeNow
	database.DB.Save(&user)

	fmt.Printf("%s has created a post.\n\tPost title: %s\n\tPost text: %s\n", user.Username, post.Title, post.Text)

	c.JSON(http.StatusAccepted, CreatePostResponse(&post))
}

/* -------------------------------------------------------------------------- */
/*                  UpdatePostText | route: /posts/updatetext                 */
/* -------------------------------------------------------------------------- */
type UpdatePostTextRequest struct {
	GetPostByIDRequest
	Text string `json:"text" binding:"required"`
}

func UpdatePostText(c *gin.Context) {
	// Check that RequestUser is authenticated
	user, found := VerifyAuth(c)
	if found == false {
		return
	}

	// Parse RequestBody
	var json UpdatePostTextRequest
	if err := c.ShouldBindJSON(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Prevent frequent UpdatePostText by User
	timeNow, canCreatePost := utils.CheckTimeIsAfter(user.LastPostAt, USER_POST_COOLDOWN)
	if canCreatePost == false {
		cdLeft := utils.GetCooldownLeft(user.LastPostAt, USER_POST_COOLDOWN, timeNow)
		c.JSON(http.StatusForbidden, gin.H{"message": fmt.Sprintf("Updating posts too frequently. Please try again in %ds", int(cdLeft.Seconds()))})
		return
	}

	// Find Post from PostID
	var post models.Post
	database.DB.First(&post, json.PostID)
	if post.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		return
	}

	// Check User is the author
	if post.UserID != user.ID {
		c.JSON(http.StatusForbidden, gin.H{"message": "You do not have valid permissions."})
		return
	}

	// Replace Post text and update User LastPostAt
	post.Text = utils.TrimString(strings.TrimSpace(json.Text), MAX_POST_TEXT_CHAR)
	user.LastPostAt = timeNow
	database.DB.Save(&post)
	database.DB.Save(&user)

	fmt.Printf("%s has updated a post.\n\tPost title: %s\n\tNew text: %s\n", user.Username, post.Title, post.Text)

	// Return new Post data
	c.JSON(http.StatusAccepted, CreatePostResponse(&post))
}

/* -------------------------------------------------------------------------- */
/*                     DeletePost | route: /delete/:postId                    */
/* -------------------------------------------------------------------------- */
type DeletePostRequest struct {
	PostID uint `uri:"postId" binding:"required"`
}

func DeletePost(c *gin.Context) {
	// Check that RequestUser is authenticated
	user, found := VerifyAuth(c)
	if found == false {
		return
	}

	// Parse RequestBody
	var json DeletePostRequest
	if err := c.ShouldBindUri(&json); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	// Find Post from PostID
	var post models.Post
	database.DB.First(&post, json.PostID)
	if post.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "Post not found."})
		return
	}

	// Check User is the author or is admin
	if (post.UserID != user.ID) && (user.Role != ADMIN) {
		c.JSON(http.StatusForbidden, gin.H{"message": "You do not have valid permissions."})
		return
	}

	database.DB.Delete(&post)

	// Update PostsCount for User
	// user.PostsCount -= 1
	// database.DB.Save(&user)

	fmt.Printf("%s has deleted a post.\n\tPost title: %s\n", user.Username, post.Title)

	// Return new Post data
	c.JSON(http.StatusAccepted, CreatePostResponse(&post))
}
