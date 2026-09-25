// Standalone CLI: seeds N dummy products into katherbox.db (idempotent).
// Run with:
//
//	go run ./cmd/seedproducts                # default 100
//	go run ./cmd/seedproducts 1000           # add 1000 more
//	# or after building:
//	seedproducts 500
package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"kather_baksho/database"
	"kather_baksho/models"
)

// Small attribute pools so every product carries believable metadata
// (brand, care level, light/water needs) instead of blank fields.
var (
	brands       = []string{"KatherBox", "GreenLeaf", "Urban Jungle", "Verdant", "PlantPeople", "BloomWorks", "EcoScape", "Habitat"}
	difficulties = []string{"easy", "easy", "medium", "medium", "hard"}
	sunlights    = []string{"low", "medium", "bright", "direct"}
	waterNeeds   = []string{"low", "medium", "high"}
	offerLabels  = []string{"", "", "", "New", "Hot", "Limited", "Sale", "Staff pick"}
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify: "Snake Plant Indoor #0001" -> "snake-plant-indoor-0001".
func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func pick(xs []string, i int) string { return xs[i%len(xs)] }

// Pool of descriptive fragments composed into product names so we get
// believable variation without repeating the same string 100 times.
var (
	indoorPlants = []string{
		"Snake Plant", "ZZ Plant", "Pothos", "Peace Lily", "Monstera Deliciosa",
		"Rubber Plant", "Spider Plant", "Anthurium", "Philodendron Brasil",
		"Fiddle Leaf Fig", "Calathea Orbifolia", "Areca Palm", "Money Plant",
		"Aglaonema Red", "Maranta Leuconeura", "Boston Fern", "English Ivy",
		"Lucky Bamboo", "Bromeliad", "Dieffenbachia",
	}
	outdoorPlants = []string{
		"Rose Hybrid", "Hibiscus", "Jasmine", "Marigold", "Bougainvillea",
		"Lantana", "Ixora", "Plumeria", "Champa", "Tecoma",
		"Duranta", "Poinsettia", "Petunia", "Zinnia", "Cosmos",
		"Sunflower", "Dahlia", "Tulip Bulb", "Daffodil Bulb", "Lily Bulb",
	}
	plantBoxes = []string{
		"Cozy Desk Box", "Air-Purifier Box", "Low-Light Box", "Pet-Friendly Box",
		"Zen Garden Box", "Office Greenery Box", "Beginner Box", "Tropical Box",
		"Succulent Starter Box", "Herb Garden Box", "Balcony Bloom Box",
		"Indoor Forest Box", "Floral Medley Box", "Native Greens Box",
		"Wedding Favours Box", "Corporate Eco Box", "Festival Gift Box",
		"Get-Well Soon Box", "Thank-You Box", "Birthday Surprise Box",
	}
	decor = []string{
		"Handmade Terracotta Pot", "Stoneware Planter", "Cement Pot Set",
		"Hanging Macramé Holder", "Wooden Stand", "Brass Watering Can",
		"Mist Sprayer", "Self-Watering Globe", "Plant Mister Bottle",
		"Round Pebble Tray", "Decorative Moss Mat", "Coco Coir Pole",
		"LED Grow Light", "Bamboo Tray", "Glass Cloche Bell Jar",
		"Steel Garden Trowel", "Pruning Shears", "Soil Scoop Set",
		"Plant Mister 500ml", "Mini Greenhouse Dome",
	}
	care = []string{
		"All-Purpose Potting Mix", "Cactus & Succulent Soil", "Perlite 1kg",
		"Vermiculite 500g", "Neem Cake Fertilizer", "Organic Compost 2kg",
		"Liquid Plant Food 250ml", "Vermicompost 1kg", "Bone Meal 500g",
		"Epsom Salt 250g", "Diatomaceous Earth 500g", "Rooting Hormone Powder",
		"Pruning Seal 100ml", "Neem Oil Spray 200ml", "Fungicide Spray 250ml",
		"Insecticidal Soap 500ml", "Plant Vitamin Tonic", "Anti-Transpirant Spray",
		"Soil Moisture Meter", "pH Test Strips",
	}
)

type template struct {
	category      string
	subcategory   string
	indoorOutdoor string // may be ""
	pool          []string
	basePrice     float64
	priceJitter   float64
	stockMin      uint
	stockMax      uint
	descStyle     string // "plant" | "box" | "decor" | "care"
	weight        int    // proportional share of the total (relative)
}

func templates() []template {
	return []template{
		{
			category: "plant", subcategory: "indoor_plant", indoorOutdoor: "indoor",
			pool: indoorPlants, basePrice: 350, priceJitter: 1800,
			stockMin: 5, stockMax: 40, descStyle: "plant", weight: 20,
		},
		{
			category: "plant", subcategory: "outdoor_plant", indoorOutdoor: "outdoor",
			pool: outdoorPlants, basePrice: 250, priceJitter: 1100,
			stockMin: 5, stockMax: 35, descStyle: "plant", weight: 18,
		},
		{
			category: "decor", subcategory: "decor",
			pool: decor, basePrice: 150, priceJitter: 2200,
			stockMin: 5, stockMax: 50, descStyle: "decor", weight: 18,
		},
		{
			category: "care", subcategory: "soil",
			pool: care, basePrice: 80, priceJitter: 1200,
			stockMin: 10, stockMax: 80, descStyle: "care", weight: 14,
		},
		{
			category: "care", subcategory: "fertilizer",
			pool: care, basePrice: 120, priceJitter: 1500,
			stockMin: 10, stockMax: 60, descStyle: "care", weight: 14,
		},
		{
			category: "decor", subcategory: "plant_box",
			pool: plantBoxes, basePrice: 1200, priceJitter: 3200,
			stockMin: 3, stockMax: 25, descStyle: "box", weight: 8,
		},
		{
			category: "care", subcategory: "care_kit",
			pool: care, basePrice: 950, priceJitter: 1800,
			stockMin: 3, stockMax: 30, descStyle: "box", weight: 8,
		},
	}
}

// Distribute `total` across templates in proportion to weight. Returns a
// per-template count that may add up to slightly less than `total` due to
// integer rounding; the last template absorbs the remainder so the sum is exact.
func split(total int, ts []template) []int {
	sumW := 0
	for _, t := range ts {
		sumW += t.weight
	}
	out := make([]int, len(ts))
	allocated := 0
	for i, t := range ts {
		out[i] = total * t.weight / sumW
		allocated += out[i]
	}
	out[len(out)-1] += total - allocated
	return out
}

func descriptionFor(style, name string) string {
	switch style {
	case "plant":
		return fmt.Sprintf(
			"%s in a healthy nursery pot. Comes with a printed care card "+
				"covering light, watering frequency, and the one mistake most new owners make.",
			name,
		)
	case "box":
		return fmt.Sprintf(
			"Curated gift box featuring %s. Includes care guide, "+
				"eco packaging and a handwritten note option at checkout.",
			name,
		)
	case "decor":
		return fmt.Sprintf(
			"%s crafted for both indoor and outdoor use. Durable finish, "+
				"easy to clean, pairs well with most planters.",
			name,
		)
	case "care":
	default:
		return fmt.Sprintf(
			"%s — premium gardening input, lab-tested for %s-grade soil "+
				"and tropical plants. Use as directed for best results.",
			name, "potting",
		)
	}
	return fmt.Sprintf("%s — quality product from the KatherBox range.", name)
}

// Round to a clean .0 / .50 figure so the shop looks tidy.
func roundPrice(p float64) float64 {
	// nearest 50 BDT
	if p < 50 {
		return 50
	}
	return float64(int((p+25)/50)) * 50
}

func main() {
	database.ConnectDatabase()
	database.DB.AutoMigrate(&models.Product{})

	total := 120
	if len(os.Args) >= 2 {
		if n, err := strconv.Atoi(os.Args[1]); err == nil && n > 0 {
			total = n
		} else {
			log.Printf("invalid count %q, falling back to 120", os.Args[1])
		}
	}

	ts := templates()
	perTemplate := split(total, ts)

	created, skipped := 0, 0
	for ti, t := range ts {
		quota := perTemplate[ti]
		for n := 0; n < quota; n++ {
			base := t.pool[(n*7+ti*3)%len(t.pool)]
			// Globally unique name: include category tag + monotonic serial.
			tag := t.category
			switch t.subcategory {
			case "indoor_plant":
				tag = "Indoor"
			case "outdoor_plant":
				tag = "Outdoor"
			case "decor":
				tag = "Decor"
			case "plant_box":
				tag = "Box"
			case "care_kit":
				tag = "Kit"
			case "soil":
				tag = "Soil"
			case "fertilizer":
				tag = "Fert"
			}
			name := fmt.Sprintf("%s %s #%04d", base, tag, n+1)

			// Idempotent: skip if name already exists.
			var existing models.Product
			if err := database.DB.Where("name = ?", name).First(&existing).Error; err == nil {
				skipped++
				continue
			}

			// Pseudo-random price/stock driven by the row index so they vary
			// across the catalog instead of cycling every 7.
			price := roundPrice(t.basePrice + float64((n*113+ti*37)%int(t.priceJitter)))
			stock := t.stockMin + uint((n*53+ti*17)%int(t.stockMax-t.stockMin+1))
			desc := fmt.Sprintf("%s Edition %d.", descriptionFor(t.descStyle, base), n+1)

			// A minority of items carry a strike-through "compare at" price so
			// the storefront's discount badges have something to show.
			var compareAt float64
			if (n+ti)%4 == 0 {
				compareAt = roundPrice(price * 1.25)
			}

			serial := ti*10000 + n + 1
			p := models.Product{
				Name:          name,
				Slug:          fmt.Sprintf("%s-%d", slugify(name), serial),
				SKU:           fmt.Sprintf("%s-%05d", strings.ToUpper(t.category[:min(4, len(t.category))]), serial),
				Brand:         pick(brands, n*3+ti),
				Category:      t.category,
				Subcategory:   t.subcategory,
				IndoorOutdoor: t.indoorOutdoor,
				Difficulty:    pick(difficulties, n+ti),
				Sunlight:      pick(sunlights, n*2+ti),
				Water:         pick(waterNeeds, n+ti*2),
				Humidity:      pick(waterNeeds, n*3+ti),
				PetFriendly:   (n+ti)%3 == 0,
				Price:         price,
				CompareAtPrice: compareAt,
				OfferLabel:    pick(offerLabels, n*5+ti*7),
				Stock:         stock,
				Description:   desc,
				// A curated, license-free catalogue photo chosen from the
				// product name + subcategory (see models.ProductImageURL and
				// frontend/public/images/products/catalog/).
				ImageURL: models.ProductImageURL(name, t.subcategory, t.category, uint(serial)),
			}
			if err := database.DB.Create(&p).Error; err != nil {
				log.Printf("insert %s failed: %v", name, err)
				continue
			}
			created++

			if (created+skipped)%200 == 0 {
				fmt.Printf("  ...%d/%d (created=%d skipped=%d)\n",
					created+skipped, total, created, skipped)
			}
		}
	}

	var inDB int64
	database.DB.Model(&models.Product{}).Count(&inDB)
	fmt.Printf("\nseedproducts: requested=%d created=%d skipped=%d (existing) total_in_db=%d\n",
		total, created, skipped, inDB)

	fbtAssigned := assignFBT()
	fmt.Printf("seedproducts: assigned fbt_ids to %d products\n", fbtAssigned)
}

// assignFBT backfills FbtIDs ("frequently bought together") for every
// in-stock product that doesn't have one yet: plants pair with a couple of
// care items + a planter, while care/decor items pair back with plants.
// Deterministic (id-driven), so it's idempotent and safe to re-run.
func assignFBT() int {
	var rows []models.Product
	database.DB.Where("stock > 0").Find(&rows)

	var plantIDs, careIDs, decorIDs []uint
	for _, p := range rows {
		switch p.Category {
		case "plant":
			plantIDs = append(plantIDs, p.ID)
		case "care":
			careIDs = append(careIDs, p.ID)
		case "decor":
			decorIDs = append(decorIDs, p.ID)
		}
	}

	pick := func(pool []uint, id uint, n int) []uint {
		if len(pool) == 0 {
			return nil
		}
		out := make([]uint, 0, n)
		start := int(id) % len(pool)
		step := len(pool) / (n + 1)
		if step < 1 {
			step = 1
		}
		for tries := 0; len(out) < n && tries < len(pool); tries++ {
			cand := pool[(start+tries*step)%len(pool)]
			if cand == id {
				continue
			}
			dup := false
			for _, x := range out {
				if x == cand {
					dup = true
					break
				}
			}
			if !dup {
				out = append(out, cand)
			}
		}
		return out
	}

	joinIDs := func(ids []uint) string {
		s := ""
		for i, id := range ids {
			if i > 0 {
				s += ","
			}
			s += strconv.FormatUint(uint64(id), 10)
		}
		return s
	}

	updated := 0
	for _, p := range rows {
		if p.FbtIDs != "" {
			continue
		}
		var fbt []uint
		switch p.Category {
		case "plant":
			fbt = append(pick(careIDs, p.ID, 2), pick(decorIDs, p.ID, 1)...)
		case "decor", "care":
			fbt = pick(plantIDs, p.ID, 2)
		}
		if len(fbt) == 0 {
			continue
		}
		database.DB.Model(&models.Product{}).Where("id = ?", p.ID).
			Update("fbt_ids", joinIDs(fbt))
		updated++
	}
	return updated
}
