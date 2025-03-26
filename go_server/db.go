package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	amqp "github.com/rabbitmq/amqp091-go"
)

var db *sql.DB

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
		"postAlbum", // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	failOnError(err, "Failed to declare a queue for postAlbum")

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

	// err = ch.Qos(
	// 	0,     // prefetch count
	// 	0,     // prefetch size
	// 	false, // global
	// )
	// failOnError(err, "Failed to set QoS")

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

	// var wg sync.WaitGroup
	// wg.Add(2)

	// go postAlbum(&wg, albumMsgs, ch, resAlbumQueue)
	// go postLike(&wg, preferredMsgs, ch)

	// var forever chan struct{}
	forever := make(chan struct{})

	worker := 10

	for w := 0; w < worker; w++ {
		go postAlbum(albumMsgs, ch, resAlbumQueue)
		// go func() {
		// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// 	defer cancel()

		// 	for d := range albumMsgs {
		// 		log.Print("Recieved a correlation ID from post album: ", d.CorrelationId)

		// 		var albumData map[string]interface{}
		// 		if err := json.Unmarshal(d.Body, &albumData); err != nil {
		// 			log.Printf("Failed to unmarshal album profile: %v", err)
		// 			continue
		// 		}

		// 		artist := albumData["artist"].(string)
		// 		title := albumData["title"].(string)
		// 		year := int(albumData["year"].(float64))
		// 		image := albumData["image"].(string)
		// 		imageSize := int(albumData["image_size"].(float64))

		// 		// Save to DB
		// 		res, err := db.Exec(`
		// 		INSERT INTO album (artist, title, year, image, image_size)
		// 		VALUES (?, ?, ?, ?, ?)
		// 		`, artist, title, year, image, imageSize)
		// 		if err != nil {
		// 			log.Printf("Failed to insert album: %v", err)
		// 			continue
		// 		}

		// 		primaryKey, _ := res.LastInsertId()
		// 		response := fmt.Sprintf("%d", primaryKey)

		// 		// Send response
		// 		err = ch.PublishWithContext(ctx,
		// 			"", // exchange
		// 			// d.ReplyTo, // routing key
		// 			resAlbumQueue.Name, // routing key
		// 			false,              // mandatory
		// 			false,              // immediate
		// 			amqp.Publishing{
		// 				ContentType:   "text/plain",
		// 				CorrelationId: d.CorrelationId,
		// 				Body:          []byte(response),
		// 			})
		// 		failOnError(err, "Failed to publish response")
		// 		d.Ack(false)
		// 	}

		// }()

		// go func() {
		// 	for d := range preferredMsgs {

		// 		var preferredData map[string]interface{}
		// 		if err := json.Unmarshal(d.Body, &preferredData); err != nil {
		// 			log.Printf("Failed to unmarshal preferred data: %v", err)
		// 			continue
		// 		}

		// 		albumId := int(preferredData["albumId"].(float64))
		// 		likeOrNot := preferredData["like"].(bool)

		// 		// Save to DB
		// 		_, err := db.Exec(`
		// 		INSERT INTO album_like (album_id, like_or_not)
		// 		VALUES (?, ?)
		// 		`, albumId, likeOrNot)
		// 		if err != nil {
		// 			log.Printf("Failed to insert album like: %v", err)
		// 			continue
		// 		}
		// 		d.Ack(false)
		// 	}
		// }()
	}

	for w := 0; w < worker; w++ {
		go postLike(preferredMsgs, ch)
	}

	<-forever
	// select {}
}
