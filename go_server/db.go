package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
)

var db *sql.DB

type Album struct {
	Artist    string
	Title     string
	Year      int
	Image     []byte
	ImageSize int64
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func postAlbum(albumMsgs <-chan amqp.Delivery, ch *amqp.Channel, resAlbumQueue amqp.Queue) {
	// defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for d := range albumMsgs {
		log.Print("Recieved a correlation ID from post album: ", d.CorrelationId)

		var albumData map[string]interface{}
		if err := json.Unmarshal(d.Body, &albumData); err != nil {
			log.Printf("Failed to unmarshal album profile: %v", err)
			continue
		}

		artist := albumData["artist"].(string)
		title := albumData["title"].(string)
		year := int(albumData["year"].(float64))
		image := albumData["image"].(string)
		imageSize := int(albumData["image_size"].(float64))

		// Save to DB
		res, err := db.Exec(`
			INSERT INTO album (artist, title, year, image, image_size)
			VALUES (?, ?, ?, ?, ?)
			`, artist, title, year, image, imageSize)
		if err != nil {
			log.Printf("Failed to insert album: %v", err)
			continue
		}

		primaryKey, _ := res.LastInsertId()
		response := fmt.Sprintf("%d", primaryKey)

		// Send response
		err = ch.PublishWithContext(ctx,
			"", // exchange
			// d.ReplyTo, // routing key
			resAlbumQueue.Name, // routing key
			false,              // mandatory
			false,              // immediate
			amqp.Publishing{
				ContentType:   "text/plain",
				CorrelationId: d.CorrelationId,
				Body:          []byte(response),
			})
		failOnError(err, "Failed to publish response")
		d.Ack(false)
	}

}

func postLike(preferredMsgs <-chan amqp.Delivery, ch *amqp.Channel) {
	// defer wg.Done()

	for d := range preferredMsgs {

		var preferredData map[string]interface{}
		if err := json.Unmarshal(d.Body, &preferredData); err != nil {
			log.Printf("Failed to unmarshal preferred data: %v", err)
			continue
		}

		albumId := int(preferredData["albumId"].(float64))
		likeOrNot := preferredData["like"].(bool)

		// Save to DB
		_, err := db.Exec(`
			INSERT INTO album_like (album_id, like_or_not)
			VALUES (?, ?)
			`, albumId, likeOrNot)
		if err != nil {
			log.Printf("Failed to insert album like: %v", err)
			continue
		}
		d.Ack(false)
	}
}

func retrieveAlbum(retrieveAlbumMsgs <-chan amqp.Delivery, ch *amqp.Channel) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for d := range retrieveAlbumMsgs {
		id, err := strconv.Atoi(string(d.Body))
		failOnError(err, "Failed to convert string to int")
		log.Printf("Received a request to retrieve album with ID: %d", id)

		var albumObj Album

		row := db.QueryRow("SELECT artist, title, year FROM album WHERE id = ?", id)

		var response = make(map[string]interface{})
		if err := row.Scan(&albumObj.Artist, &albumObj.Title, &albumObj.Year); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Album not found with ID: %d", id)
				response = map[string]interface{}{
					"error": "Album not found",
				}
			} else {
				log.Printf("Failed to query album data: %v", err)
				response = map[string]interface{}{
					"error": "Failed to query album data",
				}
			}
		} else {
			response = map[string]interface{}{
				"artist": albumObj.Artist,
				"title":  albumObj.Title,
				"year":   albumObj.Year,
			}
		}

		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Printf("Failed to marshal album data: %v", err)
			continue
		}

		err = ch.PublishWithContext(ctx,
			"",        // exchange
			d.ReplyTo, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType:   "Application/json",
				CorrelationId: d.CorrelationId,
				Body:          []byte(responseJSON),
			})
		failOnError(err, "Failed to publish response")
		d.Ack(false)
	}
}

func retrieveReview(retrieveReviewMsgs <-chan amqp.Delivery, ch *amqp.Channel) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for d := range retrieveReviewMsgs {
		id, err := strconv.Atoi(string(d.Body))
		failOnError(err, "Failed to convert string to int")
		log.Printf("Received a request to retrieve review with ID: %d", id)

		var response = make(map[string]interface{})

		row := db.QueryRow(`SELECT 
				SUM(CASE WHEN like_or_not = 1 THEN 1 ELSE 0 END) AS like_count, 
				SUM(CASE WHEN like_or_not = 0 THEN 1 ELSE 0 END) AS dislike_count
				FROM album_like 
				WHERE album_id = ?;`, id)

		var likeCount, dislikeCount int

		if err := row.Scan(&likeCount, &dislikeCount); err != nil {
			if err == sql.ErrNoRows {
				log.Printf("No reviews found for album ID: %d", id)
				response = map[string]interface{}{
					"error": "No reviews found",
				}
			} else {
				log.Printf("Failed to query review data: %v", err)
				response = map[string]interface{}{
					"error": "Failed to query review data",
				}
			}
		} else {
			response = map[string]interface{}{
				"like_count":    likeCount,
				"dislike_count": dislikeCount,
			}
		}

		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Printf("Failed to marshal album data: %v", err)
			continue
		}

		err = ch.PublishWithContext(ctx,
			"",        // exchange
			d.ReplyTo, // routing key
			false,     // mandatory
			false,     // immediate
			amqp.Publishing{
				ContentType:   "Application/json",
				CorrelationId: d.CorrelationId,
				Body:          []byte(responseJSON),
			})
		failOnError(err, "Failed to publish response")
		d.Ack(false)
	}
}

func main() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	// export DB_DSN="root:111111@tcp(localhost:3306)/cs6650"
	dsn := os.Getenv("DB_DSN")

	if dsn == "" {
		log.Fatal("DB_DSN environment variable not set")
	}

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}

	// Test the DB connection quickly
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	// Create album table
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS album (
                id INT AUTO_INCREMENT PRIMARY KEY,
                artist VARCHAR(255) NOT NULL,
                title VARCHAR(255) NOT NULL,
                year INT NOT NULL,
                image LONGBLOB NOT NULL,
                image_size INT NOT NULL
        ) ENGINE=InnoDB;
        `)

	if err != nil {
		log.Fatalf("Failed to create album table: %v", err)
	}

	// Create album_like table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS album_like (
			id INT AUTO_INCREMENT PRIMARY KEY,
			album_id INT NOT NULL,
			like_or_not BOOLEAN NOT NULL,
			FOREIGN KEY (album_id) REFERENCES album(id)
			) ENGINE=InnoDB;
		`)

	if err != nil {
		log.Fatalf("Failed to create album_like table: %v", err)
	}

	albumQueue, err := ch.QueueDeclare(
		"post_album", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Failed to declare a queue for post_album")

	preferredQueue, err := ch.QueueDeclare(
		"post_preferred", // name
		true,             // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	failOnError(err, "Failed to declare a queue for post_preferred")

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
		"retrieve_queue", // name
		false,            // durable
		false,            // delete when unused
		false,            // exclusive
		false,            // no-wait
		nil,              // arguments
	)
	failOnError(err, "Failed to declare a queue for retrieveAlbum")

	retrieveReviewQueue, err := ch.QueueDeclare(
		"retrieve_review_queue", // name
		false,                   // durable
		false,                   // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		nil,                     // arguments
	)
	failOnError(err, "Failed to declare a queue for retrieveReview")

	err = ch.Qos(
		50,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	failOnError(err, "Failed to set QoS")

	albumMsgs, err := ch.Consume(
		albumQueue.Name, // queue
		"",              // consumer
		false,           // auto-ack
		false,           // exclusive
		false,           // no-local
		false,           // no-wait
		nil,             // args
	)
	failOnError(err, "Failed to register a album post consumer")

	preferredMsgs, err := ch.Consume(
		preferredQueue.Name, // queue
		"",                  // consumer
		false,               // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)
	failOnError(err, "Failed to register a preference post consumer")

	// Consume messages from the retrieve album response queue
	retrieveAlbumMsgs, err := ch.Consume(
		retrieveAlbumQueue.Name, // queue
		"",                      // consumer
		false,                   // auto-ack
		false,                   // exclusive
		false,                   // no-local
		false,                   // no-wait
		nil,                     // args
	)
	failOnError(err, "Failed to register a retrieve album consumer")

	// Consume messages from the retrieve review response queue
	retrieveReviewMsgs, err := ch.Consume(
		retrieveReviewQueue.Name, // queue
		"",                       // consumer
		false,                    // auto-ack
		false,                    // exclusive
		false,                    // no-local
		false,                    // no-wait
		nil,                      // args
	)
	failOnError(err, "Failed to register a retrieve review consumer")

	forever := make(chan struct{})

	worker := 10

	for w := 0; w < worker; w++ {
		go postAlbum(albumMsgs, ch, resAlbumQueue)
		go postLike(preferredMsgs, ch)
		go retrieveAlbum(retrieveAlbumMsgs, ch)
		go retrieveReview(retrieveReviewMsgs, ch)

	}

	<-forever
	// select {}
}
