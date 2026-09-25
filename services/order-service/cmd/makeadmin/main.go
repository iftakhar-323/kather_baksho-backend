package main

import (
	"fmt"
	"kather_baksho/database"
	"kather_baksho/models"
	"kather_baksho/utils"
)

func main() {
	database.ConnectDatabase()
	h, _ := utils.HashPassword("Admin@12345")
	var u models.User
	err := database.DB.Where("email = ?", "admin@kather_baksho.com").First(&u).Error
	if err != nil {
		u = models.User{Name: "Site Admin", Email: "admin@kather_baksho.com", Password: h, Role: "admin"}
		database.DB.Create(&u)
		fmt.Println("CREATED admin@kather_baksho.com")
	} else {
		database.DB.Model(&u).Updates(map[string]interface{}{"password": h, "role": "admin"})
		fmt.Println("UPDATED admin@kather_baksho.com")
	}
}
