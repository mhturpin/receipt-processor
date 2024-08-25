package main

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "fmt"
    "math"
    "regexp"
    "strconv"
    "strings"
    "time"
)

// Data type definition for receipt object
type receipt struct {
    Id                  uuid.UUID
    Retailer            string
    PurchaseDateTime    time.Time
    Items               []item
    Total               float64
    Points              int
}

// Data type definition for item object
type item struct {
    ShortDescription    string
    Price               float64
}

// Data type definitions for loading request body
type receiptJson struct {
    Retailer        string      `json:"retailer"`
    PurchaseDate    string      `json:"purchaseDate"`
    PurchaseTime    string      `json:"purchaseTime"`
    Items           []itemJson  `json:"items"`
    Total           string      `json:"total"`
}

type itemJson struct {
    ShortDescription    string  `json:"shortDescription"`
    Price               string  `json:"price"`
}

// initialize receipts slice
var receipts = []receipt{}

// Routing
func main() {
    router := gin.Default()
    router.POST("/receipts/process", processReceipt)
    router.GET("/receipts/:id/points", getPoints)

    router.Run("localhost:8080")
}

// Process and save a receipt
func processReceipt(c *gin.Context) {
    var body receiptJson

    // Pull out fields from requset body
    if err := c.BindJSON(&body); err != nil {
        c.IndentedJSON(400, gin.H{"message": "Invalid request"})
        return
    }

    // Parse requsest and add new receipt to the list
    newReceipt := parseReceipt(body)
    receipts = append(receipts, newReceipt)


    fmt.Println(newReceipt)


    // c.IndentedJSON(200, newReceipt.id)
    c.IndentedJSON(200, newReceipt)
}

func parseReceipt(receiptBody receiptJson) receipt {
    var newReceipt receipt

    // Set id
    newReceipt.Id = uuid.New()

    // Set retailer
    retailerRe := regexp.MustCompile(`^[\w\s\-&]+$`)
    newReceipt.Retailer = retailerRe.FindString(receiptBody.Retailer)
    // handle blank string

    // Set date/time
    const dateFormat = "2006-01-02 15:04:05"
    newReceipt.PurchaseDateTime, _ = time.Parse(dateFormat, fmt.Sprintf("%v %v:00", receiptBody.PurchaseDate, receiptBody.PurchaseTime))
    // handle error

    // Set items
    for _, i := range receiptBody.Items {
        newReceipt.Items = append(newReceipt.Items, parseItem(i))
        // handle error
    }

    // Set total
    totalRe := regexp.MustCompile(`^\d+\.\d{2}$`)
    totalString := totalRe.FindString(receiptBody.Total)
    // handle blank string
    total, _ := strconv.ParseFloat(totalString, 64)
    // handle error
    newReceipt.Total = total

    // Calculate points
    calculatePoints(newReceipt)

    return newReceipt
}

func parseItem(itemBody itemJson) item {
    var newItem item

    // Set shortDescription
    descriptionRe := regexp.MustCompile(`^[\w\s\-]+$`)
    newItem.ShortDescription = descriptionRe.FindString(itemBody.ShortDescription)
    // handle blank string

    // Set price
    priceRe := regexp.MustCompile(`^\d+\.\d{2}$`)
    priceString := priceRe.FindString(itemBody.Price)
    // handle blank string
    price, _ := strconv.ParseFloat(priceString, 64)
    // handle error
    newItem.Price = price

    return newItem
}

func calculatePoints(r receipt) {
    totalCents := int(r.Total*100)

    // One point for every alphanumeric character in the retailer name
    r.Points = len(r.Retailer)

    // 50 points if the total is a round dollar amount with no cents
    if totalCents % 100 == 0 {
        r.Points += 50
    }

    // 25 points if the total is a multiple of 0.25
    if totalCents % 25 == 0 {
        r.Points += 25
    }

    // 5 points for every two items on the r
    r.Points += int(len(r.Items)/2)*5

    // For each item:
    // If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer. The result is the number of points earned
    for _, i := range r.Items {
        if len(strings.TrimSpace(i.ShortDescription)) % 3 == 0 {
            r.Points += int(math.Round(i.Price*0.2 + 0.5))
        }
    }

    // 6 points if the day in the purchase date is odd
    if r.PurchaseDateTime.Day() % 2 == 1 {
        r.Points += 6
    }

    // 10 points if the time of purchase is after 2:00pm and before 4:00pm
    hour := r.PurchaseDateTime.Hour()

    if hour == 14 || hour == 15 {
        r.Points += 10
    }
}

// Return the number of points for the receipt with the given id
func getPoints(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))

    // Return 404 if the id was not able to be parsed
    if err != nil {
        c.IndentedJSON(404, gin.H{"message": fmt.Sprintf("No receipt found with id %v", c.Param("id"))})
        return
    }

    // Find the receipt with the given id, and return the number of points
    for _, r := range receipts {
        if r.Id == id {
            c.IndentedJSON(200, r.Points)
            return
        }
    }

    // Return 404 if receipt not found
    c.IndentedJSON(404, gin.H{"message": fmt.Sprintf("No receipt found with id %v", c.Param("id"))})
}
