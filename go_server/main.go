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

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	albumQueue, err := ch.QueueDeclare(
		"postAlbum", // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	failOnError(err, "Failed to declare a queue for postAlbum")

	// albumQueue, err := ch.QueueDeclare(
	// 	"",    // name
	// 	false, // durable
	// 	false, // delete when unused
	// 	false, // exclusive
	// 	false, // no-wait
	// 	nil,   // arguments
	// )
	// failOnError(err, "Failed to declare a queue for postAlbum")

	// msgs_album, err := ch.Consume(
	// 	albumQueue.Name, // queue
	// 	"",              // consumer
	// 	true,            // auto-ack
	// 	false,           // exclusive
	// 	false,           // no-local
	// 	false,           // no-wait
	// 	nil,             // args
	// )
	// failOnError(err, "Failed to register a consumer for response post album")

	preferredQueue, err := ch.QueueDeclare(
		"postPreferred", // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	failOnError(err, "Failed to declare a queue for postPreferred")

	resAlbumQueue, err := ch.QueueDeclare(
		"response_album_queue", // name
		true,                   // durable
		false,                  // delete when unused
		false,                  // exclusive
		false,                  // no-wait
		nil,                    // arguments
	)
	failOnError(err, "Failed to declare a queue for postRes")

	msgs_album, err := ch.Consume(
		resAlbumQueue.Name, // queue
		"",                 // consumer
		true,               // auto-ack
		false,              // exclusive
		false,              // no-local
		false,              // no-wait
		nil,                // args
	)
	failOnError(err, "Failed to register a consumer for response post album")

	r := gin.Default()

	r.POST("/album", func(c *gin.Context) {
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

		// Channel to receive the albumId from the background goroutine
		albumIDChannel := make(chan string)

		corrID_album := uuid.New().String()
		log.Printf("Correlation ID: %s", corrID_album)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = ch.PublishWithContext(ctx,
			"", // exchange
			// "postAlbum", // routing key
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

		go func() {
			for msg := range msgs_album {
				log.Printf("Received a message: %s", msg.Body)
				fmt.Print("msg_album.correlationid:", msg.CorrelationId)
				fmt.Print("\ncorrID_album:", corrID_album)
				if corrID_album == msg.CorrelationId {
					albumId := string(msg.Body)
					log.Print("Album data  %v published to RabbitMQ successfully", albumId)
					albumIDChannel <- albumId // Send the albumId to the channel
					break
				}
			}
		}()

		fmt.Println("Album creation in progress. Please wait for the confirmation.")

		// Wait for the albumId to be received from the channel
		albumId := <-albumIDChannel

		// Now that we have the albumId, send the final response back to the client
		c.JSON(200, gin.H{"message": "Album created", "albumId": albumId, "imageSize": imageSize})

		// c.JSON(200, gin.H{"created": "Album created"})
	})

	r.POST("/album/:likeOrNot/:albumId", func(c *gin.Context) {
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

	// r.GET("/album/:albumId", func(c *gin.Context) {
	// 	albumId := c.Param("albumId")
	// 	id, err := strconv.Atoi(albumId)
	// 	if err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid album value"})
	// 	}

	// 	var albumObj Album
	// 	row := db.QueryRow("SELECT artist, title, year FROM album WHERE id = ?", id)
	// 	if err := row.Scan(&albumObj.Artist, &albumObj.Title, &albumObj.Year); err != nil {
	// 		if err == sql.ErrNoRows {
	// 			c.JSON(http.StatusNotFound, gin.H{"error": "Album not found"})
	// 		} else {
	// 			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 		}
	// 		return
	// 	}
	// 	c.JSON(200, albumObj)
	// })

	// Optionally, pass a port via environment variable or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	log.Printf("Server starting on port %s ...", port)
	r.Run(":" + port)

}
