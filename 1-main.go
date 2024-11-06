package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

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
	// Connection string to PostgreSQL
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Open the connection to the database
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}
	defer db.Close()

	// Test the database connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging database: ", err)
	}

	// Set up the HTTP route and handler
	http.HandleFunc("/interaction", func(w http.ResponseWriter, r *http.Request) {
		// Get email or phone number from query parameters
		email := r.URL.Query().Get("email")
		phoneNumber := r.URL.Query().Get("phone")

		// Validate the inputs
		if email == "" && phoneNumber == "" {
			http.Error(w, "Either email or phone number must be provided", http.StatusBadRequest)
			return
		}

		// Check and insert/update the contact information
		err := insertOrUpdateContact(db, email, phoneNumber) // Change this line to use insertOrUpdateContact
		if err != nil {
			http.Error(w, fmt.Sprintf("Error handling contact info: %v", err), http.StatusInternalServerError)
			return
		}

		// Fetch and display the contact information
		displayContactInfo(db, w, email, phoneNumber)
	})

	// Start the HTTP server
	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
func insertOrUpdateContact(db *sql.DB, email, phoneNumber string) error {
	// Check if email exists
	var existingID int
	var existingPhone string
	var linkPrecedence sql.NullInt64 // Use NullInt64 to handle potential NULL
	var createdAt, updatedAt sql.NullString

	// Query for existing contact by email
	err := db.QueryRow(`
		SELECT id, phone_number, link_precedence, created_at, updated_at
		FROM contacts
		WHERE email = $1`, email).Scan(&existingID, &existingPhone, &linkPrecedence, &createdAt, &updatedAt)

	// If no record is found (email does not exist)
	if err == sql.ErrNoRows {
		// Insert a new contact
		_, err = db.Exec(`
			INSERT INTO contacts (email, phone_number, created_at, updated_at)
			VALUES ($1, $2, NOW(), NOW())`,
			email, phoneNumber)
		if err != nil {
			return fmt.Errorf("error inserting new contact: %v", err)
		}
		fmt.Println("Inserted a new contact.")
	} else if err != nil {
		return fmt.Errorf("error querying the database: %v", err)
	} else {
		// Use linkPrecedence.Int64 if it is not NULL, otherwise default to 1
		newLinkPrecedence := linkPrecedence.Int64 + 1
		if !linkPrecedence.Valid {
			newLinkPrecedence = 1
		}

		// If the email exists, insert a new row with the LinkedID pointing to the existing record
		_, err = db.Exec(`
			INSERT INTO contacts (email, phone_number, linked_id, link_precedence, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())`,
			email, phoneNumber, existingID, newLinkPrecedence)
		if err != nil {
			return fmt.Errorf("error inserting linked contact: %v", err)
		}
		fmt.Println("Inserted a new row linked to existing contact.")
	}
	return nil
}

// Function to display contact information for a given email or phone number
func displayContactInfo(db *sql.DB, w http.ResponseWriter, email, phoneNumber string) {
	var id int
	var emailAddress, linkPrecedence, createdAt, updatedAt, deletedAt sql.NullString
	var linkedID sql.NullInt64

	// Slice to hold phone numbers
	var phoneNumbers []string

	// Query the contact details from the database for the given email
	rows, err := db.Query(`
		SELECT id, email, phone_number, linked_id, link_precedence, created_at, updated_at, deleted_at
		FROM contacts
		WHERE email = $1 OR phone_number = $2
		ORDER BY created_at DESC`, email, phoneNumber)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying the database: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Process the rows
	for rows.Next() {
		var phoneNum string
		err := rows.Scan(&id, &emailAddress, &phoneNum, &linkedID, &linkPrecedence, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error scanning database row: %v", err), http.StatusInternalServerError)
			return
		}

		// Add phone number to the list
		phoneNumbers = append(phoneNumbers, phoneNum)
	}

	// Check if any contact details were found
	if len(phoneNumbers) == 0 {
		http.Error(w, "No contact found with the provided email or phone number", http.StatusNotFound)
		return
	}

	// Display the contact details as HTML
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>Contact Information</h1>")
	fmt.Fprintf(w, "<p><strong>Email:</strong> %s</p>", emailAddress.String)

	// Display phone numbers as an array
	fmt.Fprintf(w, "<p><strong>Phone Numbers:</strong> [%s]</p>", strings.Join(phoneNumbers, ", "))

	// Display details from the latest row
	fmt.Fprintf(w, "<p><strong>ID:</strong> %d</p>", id)
	fmt.Fprintf(w, "<p><strong>Linked ID:</strong> %d</p>", linkedID.Int64)
	fmt.Fprintf(w, "<p><strong>Link Precedence:</strong> %s</p>", linkPrecedence.String)
	fmt.Fprintf(w, "<p><strong>Created At:</strong> %s</p>", createdAt.String)
	fmt.Fprintf(w, "<p><strong>Updated At:</strong> %s</p>", updatedAt.String)
	fmt.Fprintf(w, "<p><strong>Deleted At:</strong> %s</p>", deletedAt.String)
}
