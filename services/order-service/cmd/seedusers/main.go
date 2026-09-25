// Standalone CLI: seeds a batch of realistic user accounts into katherbox.db.
//
//	go run ./cmd/seedusers            # default 60 customers + 4 staff + 1 extra admin
//	go run ./cmd/seedusers 150        # 150 customers
//
// Idempotent — only tops up to the requested count, never duplicates an
// email. Every seeded account uses the password:  Test@12345
package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"
)

const seedPassword = "Test@12345"

var (
	rng = rand.New(rand.NewSource(time.Now().UnixNano()))

	first = []string{
		"Aarav", "Sara", "Rafi", "Maya", "Imran", "Nadia", "Tariq", "Lamia", "Ravi", "Anika",
		"Sumi", "Karan", "Priya", "Hasan", "Rina", "Omar", "Faria", "Bilal", "Tania", "Sabbir",
		"Mehjabin", "Asif", "Tahsin", "Rownak", "Nazia", "Sadman", "Aditi", "Taseen", "Mahmood", "Zara",
		"Nabil", "Ishrat", "Fahim", "Sneha", "Arif", "Munia", "Shakib", "Tisha", "Rakib", "Prova",
	}
	last = []string{
		"Khan", "Rahman", "Hossain", "Akter", "Islam", "Chowdhury", "Ahmed", "Begum", "Sultana", "Miah",
		"Das", "Roy", "Sarkar", "Talukder", "Bhuiyan", "Haque", "Sheikh", "Biswas", "Saha", "Karim",
	}
	cities = []string{"Dhaka", "Chattogram", "Sylhet", "Rajshahi", "Khulna", "Barishal", "Rangpur", "Mymensingh"}
)

func phone() string {
	return fmt.Sprintf("+8801%d%07d", 3+rng.Intn(7), rng.Intn(10000000))
}

func address() string {
	return fmt.Sprintf("House %d, Road %d, %s %04d, Bangladesh",
		rng.Intn(90)+1, rng.Intn(30)+1, cities[rng.Intn(len(cities))], rng.Intn(9000)+1000)
}

func main() {
	database.ConnectDatabase()
	database.DB.AutoMigrate(&models.User{})

	customers := 60
	if len(os.Args) >= 2 {
		if n, err := strconv.Atoi(os.Args[1]); err == nil && n > 0 {
			customers = n
		}
	}

	// Hash once — every demo account shares the same password, and bcrypt
	// verification doesn't care that the salt is reused across rows.
	hash, err := utils.HashPassword(seedPassword)
	if err != nil {
		fmt.Println("hash failed:", err)
		os.Exit(1)
	}

	made := map[string]int{"customer": 0, "staff": 0, "admin": 0}
	skipped := 0

	create := func(name, email, role string, points uint, daysAgo int) {
		var existing models.User
		if err := database.DB.Where("email = ?", email).First(&existing).Error; err == nil {
			skipped++
			return
		}
		u := models.User{
			Name:          name,
			Email:         email,
			Password:      hash,
			Role:          role,
			Points:        points,
			Phone:         phone(),
			Address:       address(),
			EmailVerified: rng.Float64() > 0.15,
		}
		if err := database.DB.Create(&u).Error; err != nil {
			fmt.Printf("  create %s failed: %v\n", email, err)
			return
		}
		when := time.Now().AddDate(0, 0, -daysAgo).Add(-time.Duration(rng.Intn(24)) * time.Hour)
		database.DB.Model(&models.User{}).Where("id = ?", u.ID).
			Updates(map[string]interface{}{"created_at": when, "updated_at": when})
		made[role]++
	}

	used := map[string]bool{}
	uniqEmail := func(fn, ln string) string {
		base := strings.ToLower(fn + "." + ln)
		email := base + "@katherbox.test"
		for n := 1; used[email]; n++ {
			email = fmt.Sprintf("%s%d@katherbox.test", base, n)
		}
		used[email] = true
		return email
	}

	// Customers — spread sign-up dates across the last year, varied loyalty points.
	for i := 0; i < customers; i++ {
		fn, ln := first[rng.Intn(len(first))], last[rng.Intn(len(last))]
		name := fn + " " + ln
		create(name, uniqEmail(fn, ln), "customer", uint(rng.Intn(20))*50, rng.Intn(365)+1)
	}

	// A few staff accounts.
	for i := 0; i < 4; i++ {
		fn, ln := first[rng.Intn(len(first))], last[rng.Intn(len(last))]
		create(fn+" "+ln, fmt.Sprintf("staff%d@katherbox.com", i+1), "staff", 0, rng.Intn(200)+30)
	}

	// One extra admin besides admin@katherbox.com (created by cmd/makeadmin).
	create("Iftekhar Admin", "iftekhar@katherbox.com", "admin", 0, 400)

	fmt.Printf("seedusers: +%d customers, +%d staff, +%d admin (skipped %d existing)\n",
		made["customer"], made["staff"], made["admin"], skipped)

	var total int64
	database.DB.Model(&models.User{}).Count(&total)
	fmt.Printf("seedusers: %d users in db · demo password for every seeded account: %s\n", total, seedPassword)
}
