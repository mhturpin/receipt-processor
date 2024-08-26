package main

import (
    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "errors"
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

// Initialize receipts slice
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
    newReceipt, err := parseReceipt(body)

    if err != nil {
        c.IndentedJSON(400, gin.H{"message": err.Error()})
        return
    }

    receipts = append(receipts, newReceipt)

    // Return Id
    c.IndentedJSON(200, gin.H{"id": newReceipt.Id})
}

// Parse and validate info for a receipt
func parseReceipt(receiptBody receiptJson) (receipt, error) {
    var newReceipt receipt
    var err error

    // Set Id
    newReceipt.Id = uuid.New()

    // Set Retailer
    retailerRe := regexp.MustCompile(`^[\w\s\-&]+$`)
    newReceipt.Retailer = retailerRe.FindString(receiptBody.Retailer)

    if newReceipt.Retailer == "" {
        return receipt{}, errors.New("Invalid retailer")
    }

    // Set PurchaseDateTime
    const dateFormat = "2006-01-02 15:04:05"
    newReceipt.PurchaseDateTime, err = time.Parse(dateFormat, fmt.Sprintf("%v %v:00", receiptBody.PurchaseDate, receiptBody.PurchaseTime))

    if err != nil {
        return receipt{}, errors.New("Invalid purchaseDate or purchaseTime")
    }

    // Set Items
    for _, i := range receiptBody.Items {
        parsedItem, err := parseItem(i)

        if err != nil {
            return receipt{}, err
        }

        newReceipt.Items = append(newReceipt.Items, parsedItem)
    }

    // Set Total
    totalRe := regexp.MustCompile(`^\d+\.\d{2}$`)
    totalString := totalRe.FindString(receiptBody.Total)

    if totalString == "" {
        return receipt{}, errors.New("Invalid total")
    }

    newReceipt.Total, err = strconv.ParseFloat(totalString, 64)

    if err != nil {
        return receipt{}, errors.New("Invalid total")
    }

    // Calculate Points
    newReceipt.Points = calculatePoints(newReceipt)

    return newReceipt, nil
}

// Parse and validate info for an item
func parseItem(itemBody itemJson) (item, error) {
    var newItem item

    // Set ShortDescription
    descriptionRe := regexp.MustCompile(`^[\w\s\-]+$`)
    newItem.ShortDescription = descriptionRe.FindString(itemBody.ShortDescription)

    if newItem.ShortDescription == "" {
        return item{}, errors.New("Invalid shortDescription")
    }

    // Set Price
    priceRe := regexp.MustCompile(`^\d+\.\d{2}$`)
    priceString := priceRe.FindString(itemBody.Price)

    if priceString == "" {
        return item{}, errors.New("Invalid price")
    }

    price, err := strconv.ParseFloat(priceString, 64)
    newItem.Price = price

    if err != nil {
        return item{}, errors.New("Invalid price")
    }

    return newItem, nil
}

// Calculate the number of points awarded for a receipt
func calculatePoints(r receipt) int {
    totalCents := int(r.Total*100)

    // One point for every alphanumeric character in the retailer name
    alphaNumRe := regexp.MustCompile(`[A-Za-z0-9]`)
    charList := alphaNumRe.FindAllString(r.Retailer, -1)
    points := len(charList)

    // 50 points if the total is a round dollar amount with no cents
    if totalCents % 100 == 0 {
        points += 50
    }

    // 25 points if the total is a multiple of 0.25
    if totalCents % 25 == 0 {
        points += 25
    }

    // 5 points for every two items on the r
    points += int(len(r.Items)/2)*5

    // For each item:
    // If the trimmed length of the item description is a multiple of 3, multiply the price by 0.2 and round up to the nearest integer. The result is the number of points earned
    for _, i := range r.Items {
        if len(strings.TrimSpace(i.ShortDescription)) % 3 == 0 {
            // Add 0.499 and round to round up to the nearest integer
            points += int(math.Round(i.Price*0.2 + 0.499))
        }
    }

    // 6 points if the day in the purchase date is odd
    if r.PurchaseDateTime.Day() % 2 == 1 {
        points += 6
    }

    // 10 points if the time of purchase is after 2:00pm and before 4:00pm
    hour := r.PurchaseDateTime.Hour()

    if hour == 14 || hour == 15 {
        points += 10
    }

    return points
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
            c.IndentedJSON(200, gin.H{"points": r.Points})
            return
        }
    }

    // Return 404 if receipt not found
    c.IndentedJSON(404, gin.H{"message": fmt.Sprintf("No receipt found with id %v", c.Param("id"))})
}
