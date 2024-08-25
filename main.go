package main

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "fmt"
    "strings"
    "math"
    "time"
    "regexp"
)

type Receipt struct {
    id              uuid.UUID   `json:"id"`
    Retailer        string      `json:"retailer"`
    PurchaseDate    string      `json:"purchaseDate"`
    PurchaseTime    string      `json:"purchaseTime"`
    Items           []Item      `json:"items"`
    Total           string      `json:"total"`
    points          int         `json:"points"`
}

type Item struct {
    shortDescription    string
    price               string
}

// initialize receipts slice
var receipts = []Receipt{}

// Routing
func main() {
    router := gin.Default()
    router.POST("/receipts/process", processReceipt)
    router.GET("/receipts/:id/points", getPoints)

    router.Run("localhost:8080")
}

// Process and save a receipt
func processReceipt(c *gin.Context) {
    var newReceipt Receipt

    if err := c.BindJSON(&newReceipt); err != nil {
        return
    }

    newReceipt.id = uuid.New()
    calculatePoints(newReceipt)
    receipts = append(receipts, newReceipt)

    c.IndentedJSON(http.StatusOK, newReceipt.id)
}

func calculatePoints(receipt Receipt) {
    totalCents := int(float32(receipt.Total)*100)

    // One point for every alphanumeric character in the retailer name
    receipt.points = len(receipt.Retailer)

    // 50 points if the total is a round dollar amount with no cents
    if totalCents % 100 == 0 {
        receipt.points += 50
    }

    // 25 points if the total is a multiple of 0.25
    if totalCents % 25 == 0 {
        receipt.points += 25
    }

    // 5 points for every two items on the receipt
    receipt.points += int(len(receipt.Items)/2)*5

    // For each item:
    // If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer. The result is the number of points earned
    for _, i := range receipt.Items {
        if len(strings.TrimSpace(i.shortDescription)) % 3 == 0 {
            receipt.points += math.Round(float32(i.price*0.2) + 0.5)
        }
    }

    // 6 points if the day in the purchase date is odd
    re := regexp.MustCompile(`\d{4}-\d{2}-(\d{2})`)
    date := re.FindStringSubmatch(receipt.PurchaseDate)

    if date[1] % 2 == 1 {
        receipt.points += 6
    }

    // 10 points if the time of purchase is after 2:00pm and before 4:00pm
    re = regexp.MustCompile(`(\d{2}):\d{2}`)
    date := re.FindStringSubmatch(receipt.PurchaseTime)

    if date[1] == 14 || date[1] == 15 {
        receipt.points += 10
    }
}

// Return the number of points for the receipt with the given id
func getPoints(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))

    // Return 404 if the id was not able to be parsed
    if err != nil {
        c.IndentedJSON(http.StatusNotFound, gin.H{"message": fmt.Sprintf("No receipt found with id %v", c.Param("id"))})
        return
    }

    // Find the receipt with the given id, and return the number of points
    for _, r := range receipts {
        if r.id == id {
            c.IndentedJSON(http.StatusOK, r.points)
            return
        }
    }

    // Return 404 if receipt not found
    c.IndentedJSON(http.StatusNotFound, gin.H{"message": fmt.Sprintf("No receipt found with id %v", c.Param("id"))})
}
