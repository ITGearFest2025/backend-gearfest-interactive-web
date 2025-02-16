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

func initDB() {

	var err error
	connStr := fmt.Sprintf("host=localhost port=%s user=%s password=%s dbname=%s sslmode=verify-full sslrootcert=./postgresql/certs/CA.crt sslkey=./postgresql/certs/postgresdb.key sslcert=./postgresql/certs/postgresdb.crt",
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"))

	db, err = gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to PostgreSQL with GORM and SSL successful")

	// AutoMigrate will create tables if they don't exist
	err = db.AutoMigrate(&Star{})
	if err != nil {
		log.Fatal("Error during AutoMigrate:", err)
	}
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
