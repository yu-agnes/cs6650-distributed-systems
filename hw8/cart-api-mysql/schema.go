package main

import "database/sql"

// createTables runs the schema migration on startup.
// This is the Go equivalent of Spring Boot's schema.sql or JPA auto-DDL.
// Using IF NOT EXISTS so it's safe to run every time the service starts.
func createTables(db *sql.DB) error {
	// ==================== carts table ====================
	// - id: auto-increment primary key (fast lookups by cart ID)
	// - customer_id: indexed for "find all carts by customer" queries
	// - created_at/updated_at: automatic timestamps
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS carts (
			id          INT AUTO_INCREMENT PRIMARY KEY,
			customer_id INT NOT NULL,
			created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			
			INDEX idx_customer_id (customer_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	// ==================== cart_items table ====================
	// - cart_id: foreign key to carts.id with CASCADE delete
	//   (deleting a cart automatically removes all its items → no orphans)
	// - UNIQUE(cart_id, product_id): same product can't appear twice in one cart
	//   (adding same product again should UPDATE quantity, not INSERT new row)
	//   This index also speeds up "get all items for a cart" queries
	// - quantity: must be at least 1
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cart_items (
			id         INT AUTO_INCREMENT PRIMARY KEY,
			cart_id    INT NOT NULL,
			product_id INT NOT NULL,
			quantity   INT NOT NULL DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE,
			UNIQUE KEY uk_cart_product (cart_id, product_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
	`)
	if err != nil {
		return err
	}

	return nil
}
