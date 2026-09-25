package controllers

import (
	"net/http"
	"strconv"

	"kather_baksho/database"
	"kather_baksho/models"

	"github.com/gin-gonic/gin"
)

// GET /api/gifts/recommend?budget=2000&occasion=birthday&indoor=true
// Rule-based scoring — no AI.
//   - Plants within ±30% of budget score highest; far outside drop off.
//   - Occasion keywords (birthday / housewarming / thank-you / get-well / anniversary)
//     match against a tag list baked into each product's description (lightweight).
//   - indoor / outdoor / either filter narrows the candidate set.
//
// Returns up to 8 ranked products with a `score` field for transparency.
func RecommendGifts(c *gin.Context) {
	budget, _ := strconv.ParseFloat(c.DefaultQuery("budget", "1500"), 64)
	occasion := c.Query("occasion")
	indoor := c.Query("indoor")

	var products []models.Product
	q := database.DB.Model(&models.Product{})
	if indoor == "indoor" {
		q = q.Where("indoor_outdoor IN ?", []string{"indoor", "both"})
	} else if indoor == "outdoor" {
		q = q.Where("indoor_outdoor IN ?", []string{"outdoor", "both"})
	}
	q.Find(&products)

	type scored struct {
		Product models.Product `json:"product"`
		Score   float64        `json:"score"`
		Reason  string         `json:"reason"`
	}

	keywords := map[string][]string{
		"birthday":    {"color", "flower", "bright", "pink", "yellow", "red"},
		"housewarming": {"large", "indoor", "palm", "monstera", "fiddle"},
		"thank-you":   {"succulent", "small", "cactus", "desk"},
		"get-well":    {"calm", "green", "easy", "snake", "peace"},
		"anniversary": {"romantic", "rose", "flower", "orchid"},
	}

	var out []scored
	for _, p := range products {
		// base price-fit score (peaks at exact budget, falls off linearly)
		if budget <= 0 {
			budget = 1500
		}
		diff := p.Price - budget
		if diff < 0 {
			diff = -diff
		}
		priceScore := 100.0 - (diff/budget)*100.0
		if priceScore < 0 {
			priceScore = 0
		}

		// occasion keyword score
		occScore := 0.0
		match := ""
		if occasion != "" {
			for _, kw := range keywords[occasion] {
				if containsCI(p.Name+" "+p.Description, kw) {
					occScore += 25
					match = kw
				}
			}
		}

		// prefer in-stock items
		stockBonus := 0.0
		if p.Stock > 0 {
			stockBonus = 10
		}

		score := priceScore*0.6 + occScore + stockBonus
		reason := "Within budget"
		if match != "" {
			reason = "Matches '" + match + "' for " + occasion
		} else if p.Stock == 0 {
			reason = "Out of stock"
		}

		if score >= 25 { // drop very poor matches
			out = append(out, scored{Product: p, Score: score, Reason: reason})
		}
	}

	// simple in-place selection sort by score desc (slice is small)
	for i := 0; i < len(out); i++ {
		best := i
		for j := i + 1; j < len(out); j++ {
			if out[j].Score > out[best].Score {
				best = j
			}
		}
		out[i], out[best] = out[best], out[i]
	}

	if len(out) > 8 {
		out = out[:8]
	}

	c.JSON(http.StatusOK, out)
}

func containsCI(s, sub string) bool {
	S := []rune(s)
	Sub := []rune(sub)
	if len(Sub) == 0 || len(Sub) > len(S) {
		return false
	}
	for i := 0; i+len(Sub) <= len(S); i++ {
		ok := true
		for j := 0; j < len(Sub); j++ {
			a, b := S[i+j], Sub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}