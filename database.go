package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "mamlesh"
	dbname   = "IdentifyContacts"
)

func main() {
	// Connection string
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open the connection to the database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// Test the connection
	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")

	// Create a table query
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100),
		email VARCHAR(100) UNIQUE
	);`

	// Execute the CREATE TABLE query
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatal("Error creating table: ", err)
	}

	fmt.Println("Table 'users' created successfully!")
}
