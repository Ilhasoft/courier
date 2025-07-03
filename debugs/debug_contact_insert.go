package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver
	"github.com/nyaruka/gocommon/uuids"
)

const insertContactSQL = `
INSERT INTO 
	contacts_contact(org_id, is_active, status, uuid, created_on, modified_on, created_by_id, modified_by_id, name, ticket_count) 
              VALUES($1, TRUE, 'A', $2, $3, $4, $5, $6, $7, 0)
RETURNING id
`

func main() {
	// Get database connection string from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://localhost/courier_test?sslmode=disable"
	}

	// Connect to database
	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test basic connectivity
	fmt.Println("Testing database connectivity...")
	if err := db.Ping(); err != nil {
		log.Fatalf("Database ping failed: %v", err)
	}
	fmt.Println("✓ Database connection successful")

	// Test contact insertion
	fmt.Println("\nTesting contact insertion...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Create test contact data
	orgID := 1
	contactUUID := uuids.New()
	now := time.Now()
	name := "Test Contact"
	createdBy := 1
	modifiedBy := 1

	fmt.Printf("Inserting contact with:\n")
	fmt.Printf("  org_id: %d\n", orgID)
	fmt.Printf("  uuid: %s\n", contactUUID)
	fmt.Printf("  name: %s\n", name)
	fmt.Printf("  created_by_id: %d\n", createdBy)
	fmt.Printf("  modified_by_id: %d\n", modifiedBy)

	// Execute the insert
	rows, err := tx.Query(insertContactSQL,
		orgID, contactUUID, now, now, createdBy, modifiedBy, name)
	if err != nil {
		log.Fatalf("Insert query failed: %v", err)
	}
	defer rows.Close()

	// Check if we got a result
	if rows.Next() {
		var contactID int
		if err := rows.Scan(&contactID); err != nil {
			log.Fatalf("Failed to scan contact ID: %v", err)
		}
		fmt.Printf("✓ Contact inserted successfully with ID: %d\n", contactID)
	} else {
		fmt.Println("✗ No rows returned from insert - this is the problem!")

		// Check if there were any errors
		if err := rows.Err(); err != nil {
			log.Fatalf("Error iterating rows: %v", err)
		}
	}

	// Test with NamedQuery (how the actual code does it)
	fmt.Println("\nTesting with NamedQuery...")

	type testContact struct {
		OrgID      int       `db:"org_id"`
		UUID       string    `db:"uuid"`
		CreatedOn  time.Time `db:"created_on"`
		ModifiedOn time.Time `db:"modified_on"`
		CreatedBy  int       `db:"created_by_id"`
		ModifiedBy int       `db:"modified_by_id"`
		Name       string    `db:"name"`
	}

	contact := &testContact{
		OrgID:      orgID,
		UUID:       string(uuids.New()),
		CreatedOn:  now,
		ModifiedOn: now,
		CreatedBy:  createdBy,
		ModifiedBy: modifiedBy,
		Name:       "Test Contact 2",
	}

	const namedInsertSQL = `
INSERT INTO 
	contacts_contact(org_id, is_active, status, uuid, created_on, modified_on, created_by_id, modified_by_id, name, ticket_count) 
              VALUES(:org_id, TRUE, 'A', :uuid, :created_on, :modified_on, :created_by_id, :modified_by_id, :name, 0)
RETURNING id
`

	rows2, err := tx.NamedQuery(namedInsertSQL, contact)
	if err != nil {
		log.Fatalf("Named insert query failed: %v", err)
	}
	defer rows2.Close()

	if rows2.Next() {
		var contactID int
		if err := rows2.Scan(&contactID); err != nil {
			log.Fatalf("Failed to scan contact ID from named query: %v", err)
		}
		fmt.Printf("✓ Contact inserted successfully with NamedQuery, ID: %d\n", contactID)
	} else {
		fmt.Println("✗ No rows returned from NamedQuery insert!")
		if err := rows2.Err(); err != nil {
			log.Fatalf("Error iterating named query rows: %v", err)
		}
	}

	// Check table constraints
	fmt.Println("\nChecking table constraints...")
	constraintQuery := `
SELECT 
    conname,
    pg_get_constraintdef(oid) as constraint_def
FROM pg_constraint 
WHERE conrelid = 'contacts_contact'::regclass
`

	constraintRows, err := db.Query(constraintQuery)
	if err != nil {
		log.Printf("Failed to query constraints: %v", err)
	} else {
		defer constraintRows.Close()
		fmt.Println("Table constraints:")
		for constraintRows.Next() {
			var name, def string
			if err := constraintRows.Scan(&name, &def); err != nil {
				log.Printf("Error scanning constraint: %v", err)
				continue
			}
			fmt.Printf("  %s: %s\n", name, def)
		}
	}

	fmt.Println("\n✓ Debug script completed successfully")
}
