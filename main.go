package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"

	"github.com/robfig/cron"
)

type Email struct {
	Email  string `json:"email"`
	Posted bool   `json:"posted"`
}

func StartCron() {
	c := cron.New()
	_ = c.AddFunc("1 * * ? * *", post)
	c.Start()
}

var db *sql.DB
var err error
var lastBatchTime time.Time

func main() {
	log.Println("started........")
	// Open SQLite3 database

	db, err = sql.Open("sqlite3", "./subscribers.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Create table if not exists
	createTable := `
	CREATE TABLE IF NOT EXISTS subscribers (
  		email TEXT PRIMARY KEY,
   		posted BOOLEAN,
		batch_id INTEGER DEFAULT 0,
		FOREIGN KEY (batch_id) REFERENCES batch(batch_id)
	);  
	`
	_, err = db.Exec(createTable)

	createTable1 := `
	CREATE TABLE IF NOT EXISTS batch (
		batch_id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(createTable1)

	alterTable := `
	ALTER TABLE IF NOT EXISTS subscribers
	ADD COLUMN posting INTEGER DEFAULT 0;
	`
	_, err = db.Exec(alterTable)

	e := echo.New()

	// Endpoint to handle subscribers
	e.POST("/subscribe", func(c echo.Context) error {
		// Parse request body
		var email Email
		if err := c.Bind(&email); err != nil {
			return err
		}

		// Insert email into database
		_, err := db.Exec("INSERT INTO subscribers (email,posted) VALUES (?, ?)", email.Email, email.Posted)

		if err != nil {
			fmt.Println("Error inserting into database:", err)
			return c.String(http.StatusConflict, "Email already subscribed")
		}

		StartCron()
		return c.String(http.StatusCreated, "Subscribed successfully")
	})

	// Start server
	port := ":8080"
	fmt.Println("Server is running on port", port)
	e.Logger.Fatal(e.Start(port))
}

// @cron(run every 1 min)
func post() {

	// Create a new batch
	var newBatchID int64
	
	newBatchID, err = insertBatch()
	if err != nil {
		log.Println("Error creating batch:", err)
		return
	}
	log.Printf("New batch created: %d", newBatchID)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Println(err)
	}
	_, err = db.Exec("UPDATE subscribers SET batch_id = ? WHERE posted = false", newBatchID)
	stmt, err := db.Prepare("SELECT email FROM subscribers WHERE posted = false AND batch_id = ?")
	if err != nil {
		log.Println("error*****")
		panic(err)
	}
	mail, err := stmt.Query(newBatchID)
	if err != nil {
		log.Println("error")
		panic(err)
	}
	defer mail.Close()
	var emailIds []string

	// Iterate through the mail
	for mail.Next() {
		var email string
		if err := mail.Scan(&email); err != nil {
			log.Fatal(err)
		}
		emailIds = append(emailIds, email)
	}

	if len(emailIds) > 0 {
		sendMail(emailIds)
	}

	_, err = db.Exec("UPDATE subscribers SET posted = true WHERE posted = false AND batch_id = ?", newBatchID)

	if err != nil {
		log.Fatal(err)
	}
}

func sendMail(whoSubscribed []string) {
	//send email(admin@netxd.com,whoSubscribed)
	log.Println(whoSubscribed)
}

func insertBatch() (int64, error) {
	result, err := db.Exec("INSERT INTO batch DEFAULT VALUES")
	if err != nil {
		return -1, err
	}
	batchID, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}
	return batchID, nil
}
