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
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Error opening database: ", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal("Error pinging database: ", err)
	}

	http.HandleFunc("/interaction", func(w http.ResponseWriter, r *http.Request) {
		email := r.URL.Query().Get("email")
		phoneNumber := r.URL.Query().Get("phone")

		if email == "" && phoneNumber == "" {
			http.Error(w, "Either email or phone number must be provided", http.StatusBadRequest)
			return
		}

		err := insertOrUpdateContact(db, email, phoneNumber)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error handling contact info: %v", err), http.StatusInternalServerError)
			return
		}

		displayContactInfo(db, w, email, phoneNumber)
	})
	http.HandleFunc("/seeDetails", func(w http.ResponseWriter, r *http.Request) {
		seeDetails(db, w, r)
	})

	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func insertOrUpdateContact(db *sql.DB, email, phoneNumber string) error {
	var existingID int
	var existingPhone string
	var linkPrecedence string
	var createdAt, updatedAt sql.NullString

	err := db.QueryRow(`
		SELECT id, phone_number, link_precedence, created_at, updated_at
		FROM contacts
		WHERE email = $1`, email).Scan(&existingID, &existingPhone, &linkPrecedence, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		_, err = db.Exec(`
			INSERT INTO contacts (email, phone_number, link_precedence, created_at, updated_at)
			VALUES ($1, $2, 'primary', NOW(), NOW())`,
			email, phoneNumber)
		if err != nil {
			return fmt.Errorf("error inserting new contact: %v", err)
		}
		fmt.Println("Inserted a new primary contact.")
	} else if err != nil {
		return fmt.Errorf("error querying the database: %v", err)
	} else {
		_, err = db.Exec(`
			INSERT INTO contacts (email, phone_number, linked_id, link_precedence, created_at, updated_at)
			VALUES ($1, $2, $3, 'secondary', NOW(), NOW())`,
			email, phoneNumber, existingID)
		if err != nil {
			return fmt.Errorf("error inserting secondary contact: %v", err)
		}
		fmt.Println("Inserted a secondary contact linked to existing primary contact.")
	}
	return nil
}

func displayContactInfo(db *sql.DB, w http.ResponseWriter, email, phoneNumber string) {
	var id int
	var emailAddress, linkPrecedence, createdAt, updatedAt, deletedAt sql.NullString
	var linkedID sql.NullInt64

	var phoneNumbers []string

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

	for rows.Next() {
		var phoneNum string
		err := rows.Scan(&id, &emailAddress, &phoneNum, &linkedID, &linkPrecedence, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error scanning database row: %v", err), http.StatusInternalServerError)
			return
		}

		phoneNumbers = append(phoneNumbers, phoneNum)
	}

	if len(phoneNumbers) == 0 {
		http.Error(w, "No contact found with the provided email or phone number", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>Contact Information</h1>")
	fmt.Fprintf(w, "<p><strong>Email:</strong> %s</p>", emailAddress.String)
	fmt.Fprintf(w, "<p><strong>Phone Numbers:</strong> [%s]</p>", strings.Join(phoneNumbers, ", "))
	fmt.Fprintf(w, "<p><strong>ID:</strong> %d</p>", id)
	fmt.Fprintf(w, "<p><strong>Linked ID:</strong> %d</p>", linkedID.Int64)
	fmt.Fprintf(w, "<p><strong>Link Precedence:</strong> %s</p>", linkPrecedence.String)
	fmt.Fprintf(w, "<p><strong>Created At:</strong> %s</p>", createdAt.String)
	fmt.Fprintf(w, "<p><strong>Updated At:</strong> %s</p>", updatedAt.String)
	fmt.Fprintf(w, "<p><strong>Deleted At:</strong> %s</p>", deletedAt.String)
}

func seeDetails(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	phoneNumber := r.URL.Query().Get("phone")

	if email == "" && phoneNumber == "s" {
		http.Error(w, "Email parameter is required", http.StatusBadRequest)
		return
	}

	var id int
	var emailAddress, linkPrecedence, createdAt, updatedAt, deletedAt sql.NullString
	var linkedID sql.NullInt64
	var phoneNumbers []string

	rows, err := db.Query(`
		SELECT id, email, phone_number, linked_id, link_precedence, created_at, updated_at, deleted_at
		FROM contacts
		WHERE email = $1 AND phone_number = $2
		ORDER BY created_at DESC`, email, phoneNumber)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error querying the database: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var phoneNum string
		err := rows.Scan(&id, &emailAddress, &phoneNum, &linkedID, &linkPrecedence, &createdAt, &updatedAt, &deletedAt)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error scanning database row: %v", err), http.StatusInternalServerError)
			return
		}

		phoneNumbers = append(phoneNumbers, phoneNum)
	}

	if len(phoneNumbers) == 0 {
		http.Error(w, "No contact found with the provided email", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<h1>Contact Details for Email: %s</h1>", email)
	fmt.Fprintf(w, "<p><strong>Email:</strong> %s</p>", emailAddress.String)
	fmt.Fprintf(w, "<p><strong>Phone Numbers:</strong> %v</p>", phoneNumbers)
	fmt.Fprintf(w, "<p><strong>ID:</strong> %d</p>", id)
	fmt.Fprintf(w, "<p><strong>Linked ID:</strong> %d</p>", linkedID.Int64)
	fmt.Fprintf(w, "<p><strong>Link Precedence:</strong> %s</p>", linkPrecedence.String)
	fmt.Fprintf(w, "<p><strong>Created At:</strong> %s</p>", createdAt.String)
	fmt.Fprintf(w, "<p><strong>Updated At:</strong> %s</p>", updatedAt.String)
	fmt.Fprintf(w, "<p><strong>Deleted At:</strong> %s</p>", deletedAt.String)
}
