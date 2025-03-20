package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

type Star struct {
	ID      uint   `gorm:"primaryKey" json:"-"`
	Name    string `json:"name"`
	Message string `json:"message"`
}

type Donation struct {
	ID           uint    `gorm:"primaryKey" json:"-"`
	Name         string  `json:"name"`
	Amount       float32 `json:"amount"`
	TaxDeduction bool    `json:"tax_deduction"`
	NationalID   *string `json:"national_id"`
	Fullname     *string `json:"fullname"`
	Email        *string `json:"email"`
}

func initDB() {

	var err error
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Bangkok",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
		os.Getenv("POSTGRES_PORT"))

	/* WARNING -> := declare local vaiable, so the line below WILL NOT WORK */
	/* db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})				*/

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	// Migrate the schema
	db.AutoMigrate(&Star{})
	db.AutoMigrate(&Donation{})

	if err != nil {
		log.Panic(err)
	}

	// Set connection pool options (important for concurrency)
	//db.SetMaxOpenConns(100) // Adjust as needed
	//db.SetMaxIdleConns(10)  // Adjust as needed
	//db.SetConnMaxLifetime(time.Minute * 5) // Adjust as needed

	// Test connection
	// if err := db.Ping(); err != nil {
	// 	log.Fatal(err)
	// }

	fmt.Println("Connected to PostgreSQL")
}

func LoadEnv() {
	// local development
	err := godotenv.Load(".env.local")
	if err != nil {
		log.Panic(err)
	}
}

func getStar(c *gin.Context) {

	var stars []Star
	db.Limit(10).Find(&stars)

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stars,
	})
}

func createStar(c *gin.Context) {

	var star Star
	if err := c.BindJSON(&star); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "binding error",
		})
		return
	}

	if star.Name == "" || star.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "either name or message is empty",
		})
		return
	}

	result := db.Create(&star)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "create error",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"name":    star.Name,
		"message": star.Message,
	})
}

func SetupRouter() *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.GET("/api/message", getStar)
	r.POST("/api/message", createStar)

	return r
}

func main() {
	LoadEnv()
	initDB()
	r := SetupRouter()

	r.Run(":8080")
}
