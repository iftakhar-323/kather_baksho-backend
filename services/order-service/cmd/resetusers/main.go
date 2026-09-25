package main

import (
	"fmt"
	"log"

	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"
)

func main() {
	database.ConnectDatabase()

	// Reset known demo accounts to known passwords.
	type acc struct {
		Email    string
		Name     string
		Password string
		Role     string
	}
	accounts := []acc{
		{"admin@kather_baksho.com", "Site Admin", "Admin@12345", "admin"},
		{"staff@kather_baksho.com", "Sadia Rahman", "Staff@12345", "staff"},
		{"admin@katherbox.com", "Site Admin", "Admin@12345", "admin"},
		{"admin@demo.com", "Demo Admin", "Admin@12345", "admin"},
		{"admintest@test.com", "Admin Test", "Admin@12345", "admin"},
		{"ritu@test.com", "Ritu", "Admin@12345", "admin"},
		{"karim@test.com", "Karim Hossain", "Admin@12345", "admin"},
		{"customer@test.com", "Customer Test", "Customer@12345", "customer"},
		{"iftakhar@gmail.com", "Iftakhar Alam", "Customer@12345", "customer"},
		{"cust1@test.com", "Test Customer", "Customer@12345", "customer"},
		{"staff@katherbox.com", "Sadia Rahman", "Staff@12345", "staff"},
	}

	for _, a := range accounts {
		h, err := utils.HashPassword(a.Password)
		if err != nil {
			log.Fatalf("hash failed: %v", err)
		}
		var u models.User
		if err := database.DB.Where("email = ?", a.Email).First(&u).Error; err != nil {
			// Create the canonical demo accounts if they're missing (fresh DB).
			u = models.User{Name: a.Name, Email: a.Email, Password: h, Role: a.Role, EmailVerified: true}
			database.DB.Create(&u)
			fmt.Printf("  CREATE %s  -> password=%s role=%s\n", a.Email, a.Password, a.Role)
			continue
		}
		database.DB.Model(&u).Updates(map[string]interface{}{
			"password": h,
			"role":     a.Role,
		})
		fmt.Printf("  RESET %s  -> password=%s role=%s\n", a.Email, a.Password, a.Role)
	}
	fmt.Println("done.")
}
