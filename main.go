package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	preQuerySetNum  = 50
	starsNumInSet   = 20
	refreshInterval = time.Second * 20

	postQueueSize = 10000
)

var db *gorm.DB
var starCache = [preQuerySetNum]*[starsNumInSet]Star{}
var postChannel = make(chan *Star, postQueueSize)

type Star struct {
	ID      uint   `gorm:"primaryKey" json:"-"`
	Name    string `json:"name"`
	Message string `json:"message"`
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

	if err != nil {
		log.Panic(err)
	}

	fmt.Println("Connected to PostgreSQL")
}

func loadEnv() {
	// local development
	err := godotenv.Load(".env.local")
	if err != nil {
		log.Panic(err)
	}
}

func main() {
	loadEnv()
	initDB()
	setupCache()
	r := setupRouter()

	r.Run(":8080")
}

func setupCache() {
	populateGetStarCache()   // Populate the cache initially
	go refreshGetStarCache() // Refresh the cache periodically in the background

	go processeCreateStar()
}

func setupRouter() *gin.Engine {

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.GET("/api/message", getStar)
	r.POST("/api/message", createStar)

	return r
}

/* GET star sections */
func getRandomWordsFromDB() (*[starsNumInSet]Star, error) {
	var stars [starsNumInSet]Star
	if err := db.Raw("SELECT * FROM stars ORDER BY RANDOM() LIMIT ?", starsNumInSet).Find(&stars).Error; err != nil {
		return nil, err
	}
	return &stars, nil
}

func populateGetStarCache() {
	for i := 0; i < preQuerySetNum; i++ {
		stars, err := getRandomWordsFromDB()
		if err != nil {
			log.Printf("Error querying random words from DB: %v", err)
			continue
		}

		starCache[i] = stars
	}
}

func refreshGetStarCache() {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		populateGetStarCache()
		log.Println("Word cache refreshed.")
	}
}

func getStar(c *gin.Context) {
	randomIndex := rand.Intn(preQuerySetNum)

	if starCache[randomIndex] == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to get stars: caching don't work, please resend a request",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   *starCache[randomIndex],
	})
}

/* POST star sections */
func processeCreateStar() { // Worker goroutine
	for star := range postChannel { // Range over the channel (blocks until data)
		if err := db.Create(&star).Error; err != nil {
			log.Println("Database error:", err)
		}
	}
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

	select {
	case postChannel <- &star: // Try to send the task
		// Task sent successfully

	case <-time.After(time.Second): // Timeout if the queue is full
		// Channel is full (or at least was for the timeout duration)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unavaliable",
			"message": "the server is under heavy load, please resend soon",
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status":  "accept",
		"message": "your star will be created soon",
	})
}
