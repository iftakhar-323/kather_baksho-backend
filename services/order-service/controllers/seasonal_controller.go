package controllers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Bangladesh-friendly planting calendar. Static for the MVP — no DB lookup needed.
// Each month lists plant names that are good to plant during that period.
// "Why" explains the reason in one short line (used as tooltip in the UI).
var CALENDAR = map[string][]CalendarEntry{
	"January": {
		{Name: "Cabbage", Why: "Cool weather crop"},
		{Name: "Cauliflower", Why: "Loves the cold"},
		{Name: "Spinach", Why: "Fast cool-season yield"},
		{Name: "Tomato seedlings", Why: "Start indoors for early spring"},
	},
	"February": {
		{Name: "Tomato seedlings", Why: "Transplant after mid-Feb"},
		{Name: "Coriander", Why: "Cool-season herb"},
		{Name: "Mint", Why: "Establishes quickly"},
		{Name: "Marigold", Why: "Cold-tolerant flower"},
	},
	"March": {
		{Name: "Bottle gourd (Lau)", Why: "Warm-season climber"},
		{Name: "Cucumber", Why: "Spring sowing window"},
		{Name: "Sweet potato", Why: "Vigorous in warming soil"},
		{Name: "Sunflower", Why: "Spring blooms"},
	},
	"April": {
		{Name: "Okra (Bhindi)", Why: "Heat-loving"},
		{Name: "Brinjal (Eggplant)", Why: "Loves heat"},
		{Name: "Chili peppers", Why: "Thrives in summer"},
		{Name: "Indoor ferns", Why: "Move to shade"},
	},
	"May": {
		{Name: "Bottle gourd (Lau)", Why: "Direct sow"},
		{Name: "Coconut palms", Why: "Monsoon planting"},
		{Name: "Snake plant", Why: "Heat-tolerant houseplant"},
		{Name: "Aglaonema", Why: "Loves humidity"},
	},
	"June": {
		{Name: "Aloe vera", Why: "Rainy season, fast growth"},
		{Name: "Tulsi (Holy Basil)", Why: "Monsoon herb"},
		{Name: "Jasmine", Why: "Loves monsoon humidity"},
		{Name: "Hibiscus", Why: "Established by monsoon"},
	},
	"July": {
		{Name: "Ginger", Why: "Plant rhizomes in monsoon"},
		{Name: "Turmeric", Why: "Rainy season crop"},
		{Name: "Papaya seedling", Why: "Establishes before winter"},
		{Name: "Money plant", Why: "Easy monsoon climber"},
	},
	"August": {
		{Name: "Mango grafting", Why: "Pre-monsoon humidity"},
		{Name: "Lemon seedling", Why: "Quick to establish"},
		{Name: "Bougainvillea", Why: "Colorful monsoon climber"},
		{Name: "Areca palm", Why: "Indoor air-purifier"},
	},
	"September": {
		{Name: "Radish", Why: "Cool-season root"},
		{Name: "Lettuce", Why: "Shade-grown leafy"},
		{Name: "Peace lily", Why: "Indoor autumn refresh"},
		{Name: "Snake plant cuttings", Why: "Easy propagation"},
	},
	"October": {
		{Name: "Marigold", Why: "Winter bedding"},
		{Name: "Calendula", Why: "Cool-season bloom"},
		{Name: "Cabbage seedlings", Why: "Transplant now"},
		{Name: "Rose bushes", Why: "Pre-winter root growth"},
	},
	"November": {
		{Name: "Cauliflower", Why: "Cool-weather heading"},
		{Name: "Spinach", Why: "Quick winter yield"},
		{Name: "Petunia", Why: "Winter colour"},
		{Name: "Peace lily", Why: "Indoor gift plant"},
	},
	"December": {
		{Name: "Indoor succulents", Why: "Easy care, low water"},
		{Name: "ZZ plant", Why: "Low light tolerant"},
		{Name: "Cactus", Why: "Drought-resilient"},
		{Name: "Coriander pots", Why: "Windowsill herb"},
	},
}

type CalendarEntry struct {
	Name string `json:"name"`
	Why  string `json:"why"`
}

// GET /api/seasonal-guide
// Optional query: ?month=March returns just that month, otherwise full year.
func SeasonalGuide(c *gin.Context) {
	month := strings.TrimSpace(c.Query("month"))
	if month == "" {
		c.JSON(http.StatusOK, gin.H{"calendar": CALENDAR})
		return
	}
	entry, ok := CALENDAR[month]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Unknown month: " + month})
		return
	}
	c.JSON(http.StatusOK, gin.H{"month": month, "entries": entry})
}