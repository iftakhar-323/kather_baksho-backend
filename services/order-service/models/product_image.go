package models

import "strings"

// catalogImageDir is the public path (served by the frontend) that holds the
// curated, license-free catalogue photos committed under
// frontend/public/images/products/catalog/.
const catalogImageDir = "/images/products/catalog/"

// nameKeyword maps a substring found in a product name to a catalogue image
// slug. First match wins, so more specific entries come first. An empty slug
// means "no good stock photo exists" — the frontend then draws its own
// illustration for that product (cleaner than a wrong photo).
var nameKeyword = []struct{ kw, slug string }{
	// indoor plants
	{"snake plant", "snake-plant"},
	{"zz plant", "zz-plant"},
	{"peace lily", "peace-lily"},
	{"monstera", "monstera"},
	{"rubber plant", "rubber-plant"},
	{"spider plant", "spider-plant"},
	{"anthurium", "anthurium"},
	{"philodendron", "philodendron"},
	{"fiddle leaf", "fiddle-leaf-fig"},
	{"calathea", "calathea"},
	{"areca", "areca-palm"},
	{"money plant", "money-plant"},
	{"aglaonema", "aglaonema"},
	{"maranta", "maranta"},
	{"boston fern", "boston-fern"},
	{"english ivy", "english-ivy"},
	{"lucky bamboo", "lucky-bamboo"},
	{"bromeliad", "bromeliad"},
	{"dieffenbachia", "dieffenbachia"},
	{"bamboo plant", "bamboo-plant"},
	{"pothos", "pothos"},
	// outdoor plants / flowers
	{"rose", "rose"},
	{"hibiscus", "hibiscus"},
	{"jasmine", "jasmine"},
	{"marigold", "marigold"},
	{"bougainvillea", "bougainvillea"},
	{"lantana", "lantana"},
	{"ixora", "ixora"},
	{"plumeria", "plumeria"},
	{"champa", "plumeria"},
	{"tecoma", "tecoma"},
	{"duranta", "duranta"},
	{"poinsettia", "poinsettia"},
	{"petunia", "petunia"},
	{"zinnia", "zinnia"},
	{"cosmos", "cosmos"},
	{"sunflower", "sunflower"},
	{"dahlia", "dahlia"},
	{"tulip", "tulip"},
	{"daffodil", "daffodil"},
	{"lily bulb", "lily"},
	// decor / tools
	{"terracotta", "terracotta-pot"},
	{"stoneware", "ceramic-planter"},
	{"cement pot", "ceramic-planter"},
	{"macram", "macrame-hanger"},
	{"watering can", "watering-can"},
	{"mist sprayer", "mister-spray"},
	{"self-watering globe", "mister-spray"},
	{"plant mister", "mister-spray"},
	{"grow light", "grow-light"},
	{"cloche", "glass-cloche"},
	{"trowel", "garden-trowel"},
	{"scoop", "garden-trowel"},
	{"pruning shears", "pruning-shears"},
	{"greenhouse", "greenhouse"},
	{"hanging", "hanging-planter"},
	// care inputs — with a real photo
	{"potting mix", "potting-soil"},
	{"succulent soil", "potting-soil"},
	{"perlite", "perlite"},
	{"vermiculite", "perlite"},
	{"diatomaceous", "perlite"},
	{"neem oil", "neem-oil"},
	// care inputs — no good stock photo (drawn illustration instead)
	{"wooden stand", ""},
	{"pebble tray", ""},
	{"moss mat", ""},
	{"coir pole", ""},
	{"bamboo tray", ""},
	{"neem cake", ""},
	{"bone meal", ""},
	{"epsom", ""},
	{"compost", ""},
	{"liquid plant food", ""},
	{"rooting hormone", ""},
	{"vitamin tonic", ""},
	{"pruning seal", ""},
	{"fungicide", ""},
	{"insecticidal", ""},
	{"anti-transpirant", ""},
	{"moisture meter", ""},
	{"ph test", ""},
}

// subcategoryPool is the fallback set used when a product name matches no
// keyword. The id picks one deterministically so similar products still vary.
// Every slug here has a real photo.
var subcategoryPool = map[string][]string{
	"indoor_plant":  {"monstera", "snake-plant", "pothos", "calathea", "philodendron", "peace-lily", "zz-plant"},
	"outdoor_plant": {"rose", "hibiscus", "marigold", "bougainvillea", "sunflower", "zinnia", "jasmine", "poinsettia", "tulip", "plumeria"},
	"decor":         {"terracotta-pot", "ceramic-planter", "watering-can", "macrame-hanger", "grow-light", "hanging-planter"},
	"soil":          {"potting-soil", "perlite"},
	// use available photos for boxes and care instead of drawn illustrations
	"plant_box":  {"succulent-arrangement", "herb-garden", "terrarium", "greenhouse"},
	"fertilizer": {"potting-soil", "perlite", "neem-oil"},
	"care_kit":   {"garden-trowel", "watering-can", "pruning-shears", "mister-spray"},
}

var categoryPool = map[string][]string{
	"plant": {"monstera", "snake-plant", "pothos", "rose", "hibiscus"},
	"decor": {"terracotta-pot", "ceramic-planter", "watering-can", "macrame-hanger"},
	"care":  {"potting-soil", "perlite"},
}

// ProductImageURL returns the public path of the best-matching catalogue photo,
// or "" when no genuine photo fits (the frontend then draws an illustration).
func ProductImageURL(name, subcategory, category string, id uint) string {
	lower := strings.ToLower(name)
	for _, e := range nameKeyword {
		if strings.Contains(lower, e.kw) {
			if e.slug == "" {
				return ""
			}
			return catalogImageDir + e.slug + ".jpg"
		}
	}
	if pool := subcategoryPool[subcategory]; len(pool) > 0 {
		return catalogImageDir + pool[int(id)%len(pool)] + ".jpg"
	}
	if _, ok := subcategoryPool[subcategory]; ok {
		return "" // known subcategory, deliberately no photo
	}
	if pool := categoryPool[category]; len(pool) > 0 {
		return catalogImageDir + pool[int(id)%len(pool)] + ".jpg"
	}
	return ""
}
