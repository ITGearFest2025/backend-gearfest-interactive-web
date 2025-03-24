package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	preQuerySetNum  = 30               // number of cache in one time
	starsNumInSet   = 20               // number of word inside one cache
	refreshInterval = time.Second * 20 // refresh interval for cache

	postQueueSize = 500 // createStar queue size
)

var db *gorm.DB
var starCache = [preQuerySetNum]*[starsNumInSet]Star{}
var postChannel = make(chan *Star, postQueueSize)

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
	Bill         *string `json:"bill"`
}

func initDB() {

	var err error
	dsn := fmt.Sprintf("host=gearfest-db user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Bangkok",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_DBNAME"),
		os.Getenv("DB_PORT"))

	/* WARNING -> := declare local vaiable, so the line below WILL NOT WORK */
	/* db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})				*/

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	// Migrate the schema
	db.AutoMigrate(&Star{})

	migrator := db.Migrator()

	// Check if the table exists.
	if !migrator.HasTable(&Donation{}) {
		fmt.Println("Table 'Donation' does not exist. Auto-migrating.")
		if err := migrator.AutoMigrate(&Donation{}); err != nil {
			panic(err)
		}
		fmt.Println("Table 'Donation' created successfully.")

	} else {
		fmt.Println("Table 'Donation' exists. Checking for 'bill' column.")

		// Check if the 'Image' column exists.
		if !migrator.HasColumn(&Donation{}, "bill") {
			fmt.Println("Image column does not exist. Adding it.")
			if err := migrator.AddColumn(&Donation{}, "bill"); err != nil {
				panic(err)
			}
			fmt.Println("bill column added successfully.")
		}
	}

	if err != nil {
		log.Panic(err)
	}

	log.Println("connected to postgresql")
}

func main() {

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

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong")
	})

	r.GET("/api/message", getStar)
	r.POST("/api/message", createStar)
	r.POST("/api/donate", createDonation)
	r.GET("/api/top-donate", getTopDonate)
	r.GET("/api/total-donate", getTotalDonation)

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
			"message": "body binding error: " + err.Error(),
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

/* POST donation */
func createDonation(c *gin.Context) {
	var donation Donation
	if err := c.BindJSON(&donation); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "body binding error: " + err.Error(),
		})
		return
	}

	if donation.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "name must not be null",
		})
		return
	}

	if donation.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "amount must be greater than 0",
		})
		return
	}

	if donation.TaxDeduction && (!valStr(donation.Fullname) || !valStr(donation.Email) || !valStr(donation.NationalID)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "fullname, email, and national_id must not be empty or null",
		})
		return
	}

	if err := db.Create(&donation).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failed",
			"message": "creation error: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "donation created",
	})

}

type TopDonationHolder struct {
	Name          string  `json:"name"`
	TotalDonation float32 `json:"total_donation"`
}

func getTopDonate(c *gin.Context) {
	var topDonationHolder [10]TopDonationHolder
	queryString := "SELECT name, sum(amount) AS total_donation FROM donations GROUP BY name ORDER BY total_donation DESC LIMIT 10"

	if err := db.Raw(queryString).Find(&topDonationHolder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to get top donation: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   topDonationHolder,
	})
}

func getTotalDonation(c *gin.Context) {
	var totalDonation float32
	queryString := "SELECT sum(amount) AS total_donation FROM donations"

	if err := db.Raw(queryString).Find(&totalDonation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failed",
			"message": "failed to get total donation: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   totalDonation,
	})
}

func valStr(in *string) bool {
	return in != nil && *in != ""
}
