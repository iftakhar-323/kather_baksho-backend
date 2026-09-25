// Standalone CLI: seeds 30 dummy orders for the demo customer.
// Run with:
//
//	go run ./cmd/seedorders
//
// Idempotent — wipes any previous orders/items for the target customer
// before inserting. Statuses are spread across the whole delivery flow so
// the Orders page shows realistic variety.
package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"kather_baksho/database"
	"kather_baksho/models"
)

const targetEmail = "customer@test.com"

var statuses = []string{
	"Pending",
	"Processing",
	"Packed",
	"On the Way",
	"Delivered",
}

var gifts = []bool{false, false, true}

func main() {
	rand.Seed(time.Now().UnixNano())

	database.ConnectDatabase()

	var user models.User
	if err := database.DB.Where("email = ?", targetEmail).First(&user).Error; err != nil {
		log.Fatalf("target user %s not found — run cmd/makeadmin / cmd/resetusers first: %v", targetEmail, err)
	}

	var products []models.Product
	if err := database.DB.Limit(40).Find(&products).Error; err != nil {
		log.Fatalf("no products to seed against: %v", err)
	}
	if len(products) == 0 {
		log.Fatalf("no products in DB — run cmd/seedproducts first")
	}

	// Clean any prior demo orders for this user so re-runs stay at 30.
	if err := database.DB.Where("user_id = ?", user.ID).
		Delete(&models.Order{}).Error; err != nil {
		log.Fatalf("clear old orders: %v", err)
	}

	const N = 30
	for i := 0; i < N; i++ {
		status := statuses[i%len(statuses)]
		itemCount := 1 + rand.Intn(3) // 1..3 items
		var items []models.OrderItem
		var total float64

		used := map[uint]bool{}
		for j := 0; j < itemCount; j++ {
			p := products[rand.Intn(len(products))]
			if used[p.ID] {
				p = products[(rand.Intn(len(products))+1)%len(products)]
			}
			used[p.ID] = true

			qty := 1 + rand.Intn(3)
			items = append(items, models.OrderItem{
				ProductID: p.ID,
				Quantity:  uint(qty),
				Price:     p.Price,
			})
			total += p.Price * float64(qty)
		}

		// Stamp a varied creation date (oldest = 60 days ago, newest = today).
		createdAt := time.Now().AddDate(0, 0, -rand.Intn(60))

		o := models.Order{
			UserID:     user.ID,
			TotalPrice: total,
			Status:     status,
			GiftWrap:   gifts[rand.Intn(len(gifts))],
			Items:      items,
		}
		o.CreatedAt = createdAt
		o.UpdatedAt = createdAt
		if err := database.DB.Create(&o).Error; err != nil {
			log.Fatalf("insert order %d: %v", i, err)
		}
	}

	fmt.Printf("OK — %d dummy orders inserted for %s\n", N, targetEmail)
}