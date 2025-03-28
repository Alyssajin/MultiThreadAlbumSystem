package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Album struct {
	Artist    string
	Title     string
	Year      int
	Image     []byte
	ImageSize int64
}

type Profile struct {
	Artist string `json:"artist"`
	Title  string `json:"title"`
	Year   string `json:"year"`
}

var pendingResponses = struct {
	sync.Mutex
	m map[string]chan string
}{m: make(map[string]chan string)}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func processRabbitMqResponse(msgs <-chan amqp.Delivery) {
	go func() {
		for msg := range msgs {
			log.Printf("Received a message: %s", msg.Body)
			fmt.Print("msg_album.correlationid:", msg.CorrelationId)
			// fmt.Print("\ncorrID_album:", corrID_album)
			corrID := msg.CorrelationId

			pendingResponses.Lock()
			respChan, ok := pendingResponses.m[corrID]
			if ok {
				respChan <- string(msg.Body)
				delete(pendingResponses.m, corrID)
			}
			pendingResponses.Unlock()
			msg.Ack(false)

		}
	}()
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// Declare a queue for the post an album
	albumQueue, err := ch.QueueDeclare(
		"post_album", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Failed to declare a queue for post_album")

	// Declare a queue for the post an album like
	preferredQueue, err := ch.QueueDeclare(
		"post_preferred", // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	failOnError(err, "Failed to declare a queue for post_preferred")

	// Declare a queue for the response from post album
	resAlbumQueue, err := ch.QueueDeclare(
		"response_album_queue", // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	failOnError(err, "Failed to declare a queue for postRes")

	// Declare a queue for RPC: getting the album by ID
	retrieveAlbumQueue, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue for retrieveAlbum")

	// Declare a queue for RPC: getting the album review by ID
	retrieveReviewQueue, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue for retrieveReview")

	err = ch.Qos(
		50,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	// Consume messages from the album queue
	msgsAlbum, err := ch.Consume(
		resAlbumQueue.Name, // queue
		"",                 // consumer
		false,              // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	failOnError(err, "Failed to register a consumer for response post album")

	// Consume messages from retrieve album queue
	msgsRetrieveAlbum, err := ch.Consume(
		retrieveAlbumQueue.Name, // queue
		"",                      // consumer
		false,                   // auto-ack
		false,                   // exclusive
		false,                   // no-local
		false,                   // no-wait
		nil,                     // args
	)
	failOnError(err, "Failed to register a consumer for retrieve album")

	// Consume messages from retrieve review queue
	msgsRetrieveReview, err := ch.Consume(
		retrieveReviewQueue.Name, // queue
		"",                       // consumer
		false,                    // auto-ack
		false,                    // exclusive
		false,                    // no-local
		false,                    // no-wait
		nil,                      // args
	)
	failOnError(err, "Failed to register a consumer for retrieve review")

	go processRabbitMqResponse(msgsAlbum)
	go processRabbitMqResponse(msgsRetrieveAlbum)
	go processRabbitMqResponse(msgsRetrieveReview)

	r := gin.Default()

	r.POST("/albums", func(c *gin.Context) {
		profileStr := c.PostForm("profile")
		if profileStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing profile field"})
			return
		}

		var profile Profile
		if err := json.Unmarshal([]byte(profileStr), &profile); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid profile JSON"})
			return
		}

		file, err := c.FormFile("image")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing image file"})
			return
		}

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open image file"})
			return
		}
		defer src.Close()

		imageData, err := io.ReadAll(src)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image file"})
			return
		}
		imageSize := len(imageData)

		yearInt, err := strconv.Atoi(profile.Year)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year value"})
			return
		}

		albumData := map[string]interface{}{
			"artist":     profile.Artist,
			"title":      profile.Title,
			"year":       yearInt,
			"image":      imageData,
			"image_size": imageSize,
		}

		albumDataBytes, err := json.Marshal(albumData)
		failOnError(err, "Failed to marshal album data")

		corrID_album := uuid.New().String()
		// log.Printf("Correlation ID: %s", corrID_album)

		responseChan := make(chan string)
		pendingResponses.Lock()
		pendingResponses.m[corrID_album] = responseChan
		pendingResponses.Unlock()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = ch.PublishWithContext(ctx,
			"", // exchange
			// "post_album", // routing key
			albumQueue.Name, // Queue name
			false,           // mandatory
			false,           // immediate
			amqp.Publishing{
				ContentType:   "application/json",
				CorrelationId: corrID_album,
				// ReplyTo:       albumQueue.Name,
				ReplyTo: resAlbumQueue.Name,
				Body:    albumDataBytes,
			})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send album message to RabbitMQ"})
			failOnError(err, "Failed to send album message to RabbitMQ")
		}

		select {
		case albumId := <-responseChan:
			c.JSON(http.StatusOK, gin.H{"message": "Album created", "albumId": albumId, "imageSize": imageSize})
		case <-ctx.Done():
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Timeout waiting for album response"})
		}
	})

	r.POST("/review/:likeOrNot/:albumId", func(c *gin.Context) {
		log.Print("Inside like route")
		likeOrNot := c.Param("likeOrNot")
		albumId := c.Param("albumId")
		id, err := strconv.Atoi(albumId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid album value"})
		}

		var like bool
		if likeOrNot == "like" {
			like = true
		} else if likeOrNot == "dislike" {
			like = false
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid likeOrNot value"})
			return
		}

		preference := map[string]interface{}{
			"like":    like,
			"albumId": id,
		}

		preferenceBytes, err := json.Marshal(preference)
		failOnError(err, "Failed to marshal preference data")

		err = ch.Publish(
			"",                  // exchange
			preferredQueue.Name, // Queue name
			false,               // mandatory
			false,               // immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        preferenceBytes,
			})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send preference message to RabbitMQ"})
			return
		}

		fmt.Println("Like creation in progress. Please wait for the confirmation.")

		c.JSON(200, gin.H{"message": "Album like updated"})

	})

	r.GET("/albums/:albumId", func(c *gin.Context) {
		albumId := c.Param("albumId")
		_, err := strconv.Atoi(albumId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid album value"})
		}

		var albumObj Album

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		corrID := uuid.New().String()

		responseChan := make(chan string)
		pendingResponses.Lock()
		pendingResponses.m[corrID] = responseChan
		pendingResponses.Unlock()

		err = ch.PublishWithContext(ctx,
			"",               // exchange
			"retrieve_queue", // routing key
			false,            // mandatory
			false,            // immediate
			amqp.Publishing{
				ContentType:   "text/plain",
				CorrelationId: corrID,
				ReplyTo:       retrieveAlbumQueue.Name,
				Body:          []byte(albumId),
			})
		failOnError(err, "Failed to publish message for retrieving album")

		select {
		case albumInfo := <-responseChan:
			// Unmarshal the albumInfo JSON string into the albumObj
			log.Printf("Received album info: %s", albumInfo)
			err := json.Unmarshal([]byte(albumInfo), &albumObj)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal album info"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Review retrieved", "albumInfo": albumObj})
		case <-ctx.Done():
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Timeout waiting for album response"})
		}
	})

	r.GET("/review/:albumId", func(c *gin.Context) {
		albumId := c.Param("albumId")
		_, err := strconv.Atoi(albumId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid album value"})
		}

		var response map[string]interface{}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		corrID := uuid.New().String()
		responseChan := make(chan string)
		pendingResponses.Lock()
		pendingResponses.m[corrID] = responseChan
		pendingResponses.Unlock()
		err = ch.PublishWithContext(ctx,
			"",                      // exchange
			"retrieve_review_queue", // routing key
			false,                   // mandatory
			false,                   // immediate
			amqp.Publishing{
				ContentType:   "text/plain",
				CorrelationId: corrID,
				ReplyTo:       retrieveReviewQueue.Name,
				Body:          []byte(albumId),
			})
		failOnError(err, "Failed to publish message for retrieving album review")
		select {
		case reviewInfo := <-responseChan:
			log.Printf("Received review info: %s", reviewInfo)
			err := json.Unmarshal([]byte(reviewInfo), &response)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unmarshal review info"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"message": "Album review retrieved", "reviewInfo": response})
		case <-ctx.Done():
			c.JSON(http.StatusGatewayTimeout, gin.H{"error": "Timeout waiting for album review response"})
		}
	})

	// Health check route
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// // GET /count -> returns row count in "test_table"
	// r.GET("/count", func(c *gin.Context) {
	// 	var cnt int
	// 	row := db.QueryRow("SELECT COUNT(*) FROM test_table")
	// 	if err := row.Scan(&cnt); err != nil {
	// 		c.JSON(500, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	c.JSON(200, gin.H{"row_count": cnt})
	// })

	// Optionally, pass a port via environment variable or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	log.Printf("Server starting on port %s ...", port)
	r.Run(":" + port)

}
