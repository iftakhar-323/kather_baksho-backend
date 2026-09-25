package controllers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// ---------- Achievements ----------

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GET /api/loyalty/achievements
func ListAchievements(c *gin.Context) {
	var list []models.Achievement
	database.DB.Order("points_bonus asc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// GET /api/loyalty/achievements/me — which ones the current user has unlocked
func MyAchievements(c *gin.Context) {
	uid := c.GetUint("user_id")
	var rows []models.UserAchievement
	database.DB.Where("user_id = ?", uid).Find(&rows)
	got := map[uint]models.UserAchievement{}
	for _, r := range rows {
		got[r.AchievementID] = r
	}
	var all []models.Achievement
	database.DB.Order("points_bonus asc").Find(&all)
	type item struct {
		models.Achievement
		Unlocked   bool       `json:"unlocked"`
		UnlockedAt *time.Time `json:"unlocked_at,omitempty"`
	}
	out := make([]item, 0, len(all))
	for _, a := range all {
		if ua, ok := got[a.ID]; ok {
			out = append(out, item{a, true, &ua.UnlockedAt})
		} else {
			out = append(out, item{a, false, nil})
		}
	}
	c.JSON(http.StatusOK, out)
}

// POST /api/loyalty/achievements/:id/claim — claim points for an unlocked achievement
func ClaimAchievement(c *gin.Context) {
	uid := c.GetUint("user_id")
	aid, _ := strconv.Atoi(c.Param("id"))
	var a models.Achievement
	if err := database.DB.First(&a, aid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such achievement"})
		return
	}
	var existing models.UserAchievement
	if err := database.DB.Where("user_id = ? AND achievement_id = ?", uid, aid).
		First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already claimed"})
		return
	}
	ua := models.UserAchievement{
		UserID: uid, AchievementID: uint(aid), UnlockedAt: time.Now(),
	}
	database.DB.Create(&ua)
	// award points
	database.DB.Model(&models.User{}).Where("id = ?", uid).
		UpdateColumn("points", database.DB.Raw("points + ?", a.PointsBonus))
	c.JSON(http.StatusOK, gin.H{"ok": true, "awarded": a.PointsBonus})
}

// Achievement code prefix conventions (encoded in Achievement.Code):
//   first_order | orders_5 | orders_20
//   points_100 | points_500 | points_1000
//   answers_10 | reviews_5 | referrals_3
//
// POST /api/loyalty/check-achievements  (called after any milestone-triggering event)
// Inspects simple rules and unlocks matching achievements.
func CheckAchievements(c *gin.Context) {
	uid := c.GetUint("user_id")
	var all []models.Achievement
	database.DB.Find(&all)
	var me models.User
	database.DB.First(&me, uid)

	// Gather metrics
	var orderCount int64
	database.DB.Model(&models.Order{}).Where("user_id = ?", uid).Count(&orderCount)
	var answerCount int64
	database.DB.Model(&models.CommunityAnswer{}).Where("user_id = ?", uid).Count(&answerCount)

	metrics := map[string]int{
		"first_order":  1, // boolean check below
		"orders_5":     int(orderCount),
		"orders_20":    int(orderCount),
		"points_100":   int(me.Points),
		"points_500":   int(me.Points),
		"points_1000":  int(me.Points),
		"answers_10":   int(answerCount),
		"referrals_3":  0, // populated below
	}

	// Referral count for "referrals_3"
	var refCount int64
	database.DB.Model(&models.Referral{}).Where("referrer_id = ?", uid).Count(&refCount)
	metrics["referrals_3"] = int(refCount)

	// First-order special-case (boolean)
	if orderCount == 0 {
		metrics["first_order"] = 0
	}

	unlocked := 0
	for _, a := range all {
		key := a.Code // by convention Code encodes the rule key
		if v, ok := metrics[key]; ok && v >= 1 {
			var existing models.UserAchievement
			if err := database.DB.Where("user_id = ? AND achievement_id = ?", uid, a.ID).
				First(&existing).Error; err == nil {
				continue // already unlocked
			}
			database.DB.Create(&models.UserAchievement{
				UserID: uid, AchievementID: a.ID, UnlockedAt: time.Now(),
			})
			unlocked++
		}
	}
	c.JSON(http.StatusOK, gin.H{"unlocked": unlocked})
}

// ---------- Membership Tier (per-user) ----------

// Compute tier from total spend: Bronze < 15k || Silver < 50k || Gold >= 50k
func currentTier(totalSpend float64) string {
	switch {
	case totalSpend >= 50000:
		return "Gold"
	case totalSpend >= 15000:
		return "Silver"
	default:
		return "Bronze"
	}
}

// GET /api/loyalty/tier
func MyTier(c *gin.Context) {
	uid := c.GetUint("user_id")
	var u models.User
	database.DB.First(&u, uid)
	var total float64
	row := database.DB.Model(&models.Order{}).
		Select("COALESCE(SUM(total_price),0)").Where("user_id = ?", uid).
		Row()
	row.Scan(&total)
	tier := currentTier(total)

	// Upsert per-user tier row
	var um models.UserMembership
	database.DB.Where("user_id = ?", uid).First(&um)
	um.UserID = uid
	um.Tier = tier
	um.TotalSpend = total
	um.Points = u.Points
	um.LastUpdated = time.Now()
	if um.ID == 0 {
		database.DB.Create(&um)
	} else {
		database.DB.Save(&um)
	}

	discount := 0.0
	switch tier {
	case "Silver":
		discount = 5
	case "Gold":
		discount = 10
	}

	c.JSON(http.StatusOK, gin.H{
		"tier":         tier,
		"total_spend":  total,
		"points":       u.Points,
		"discount_pct": discount,
		"next_tier":    nextTierInfo(tier),
	})
}

func nextTierInfo(t string) gin.H {
	switch t {
	case "Bronze":
		return gin.H{"name": "Silver", "need_spend": 15000}
	case "Silver":
		return gin.H{"name": "Gold", "need_spend": 50000}
	}
	return gin.H{"name": "Gold", "need_spend": 0}
}

// ---------- Referrals ----------

// GET /api/loyalty/referral — get or generate this user's referral code
func MyReferralCode(c *gin.Context) {
	uid := c.GetUint("user_id")
	var rc models.ReferralCode
	if err := database.DB.Where("user_id = ?", uid).First(&rc).Error; err != nil {
		rc = models.ReferralCode{
			UserID: uid,
			Code:   fmt.Sprintf("KB%s%04d", randHex(2), uid%10000),
		}
		database.DB.Create(&rc)
	}
	c.JSON(http.StatusOK, rc)
}

// POST /api/loyalty/referral/redeem  body: {code}
func RedeemReferral(c *gin.Context) {
	uid := c.GetUint("user_id")
	var body struct {
		Code string `json:"code"`
	}
	if err := c.BindJSON(&body); err != nil || body.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code required"})
		return
	}
	var rc models.ReferralCode
	if err := database.DB.Where("code = ?", body.Code).First(&rc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "invalid code"})
		return
	}
	if rc.UserID == uid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot redeem own code"})
		return
	}
	// already redeemed by this user?
	var existing models.Referral
	if err := database.DB.Where("referred_id = ?", uid).First(&existing).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "already redeemed"})
		return
	}
	r := models.Referral{
		ReferrerID:  rc.UserID,
		ReferredID:  uid,
		ReferrerReward: 100,
		ReferredReward: 100,
	}
	database.DB.Create(&r)
	// award bonus points to BOTH users
	database.DB.Model(&models.User{}).Where("id = ?", rc.UserID).
		UpdateColumn("points", database.DB.Raw("points + 100"))
	database.DB.Model(&models.User{}).Where("id = ?", uid).
		UpdateColumn("points", database.DB.Raw("points + 100"))
	c.JSON(http.StatusOK, gin.H{"ok": true, "bonus": 100})
}

// GET /api/loyalty/referrals/me — list who used my code
func MyReferrals(c *gin.Context) {
	uid := c.GetUint("user_id")
	var rs []models.Referral
	database.DB.Where("referrer_id = ?", uid).Order("created_at desc").Find(&rs)
	c.JSON(http.StatusOK, rs)
}

// ---------- Coupon Rewards (rewards the user can swap points for) ----------

// GET /api/loyalty/rewards
func ListRewards(c *gin.Context) {
	var list []models.CouponReward
	database.DB.Order("points_cost asc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// POST /api/loyalty/rewards/:id/redeem
func RedeemReward(c *gin.Context) {
	uid := c.GetUint("user_id")
	rid, _ := strconv.Atoi(c.Param("id"))
	var r models.CouponReward
	if err := database.DB.First(&r, rid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no such reward"})
		return
	}
	var u models.User
	database.DB.First(&u, uid)
	if int(u.Points) < r.PointsCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "not enough points"})
		return
	}
	database.DB.Model(&u).UpdateColumn("points", database.DB.Raw("points - ?", r.PointsCost))
	code := fmt.Sprintf("RW-%s-%05d", randHex(2), rid)
	c.JSON(http.StatusOK, gin.H{
		"coupon_code": code,
		"discount":    r.Percent,
		"spend":       u.Points - uint(r.PointsCost),
	})
}

// helper retained so strconv import stays referenced
var _ = strconv.Atoi
