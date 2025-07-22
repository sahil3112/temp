package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// DatabaseManager provides utility functions for database operations
type DatabaseManager struct {
	db *sql.DB
}

// NewDatabaseManager creates a new database manager instance
func NewDatabaseManager(database *sql.DB) *DatabaseManager {
	return &DatabaseManager{db: database}
}

// GetUserByUsername retrieves a user by username with all fields
func (dm *DatabaseManager) GetUserByUsername(username string) (*User, error) {
	var user User
	var createdAt, updatedAt string
	
	query := `SELECT id, username, password, email, first_name, last_name, role, is_active, created_at, updated_at 
			  FROM users WHERE username = ?`
	
	err := dm.db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Password, &user.Email,
		&user.FirstName, &user.LastName, &user.Role, &user.IsActive,
		&createdAt, &updatedAt)
	
	if err != nil {
		return nil, err
	}
	
	// Parse timestamps
	user.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	user.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	
	return &user, nil
}

// GetCustomerByEmail retrieves a customer by email
func (dm *DatabaseManager) GetCustomerByEmail(email string) (*Customer, error) {
	var customer Customer
	var createdAt, updatedAt string
	
	query := `SELECT id, first_name, last_name, email, phone, company, status, created_at, updated_at 
			  FROM customers WHERE email = ?`
	
	err := dm.db.QueryRow(query, email).Scan(
		&customer.ID, &customer.FirstName, &customer.LastName, &customer.Email,
		&customer.Phone, &customer.Company, &customer.Status, &createdAt, &updatedAt)
	
	if err != nil {
		return nil, err
	}
	
	// Parse timestamps
	customer.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	customer.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
	
	return &customer, nil
}

// CreateCustomer creates a new customer record
func (dm *DatabaseManager) CreateCustomer(customer *Customer) error {
	query := `INSERT INTO customers (first_name, last_name, email, phone, company, status)
			  VALUES (?, ?, ?, ?, ?, ?)`
	
	result, err := dm.db.Exec(query, customer.FirstName, customer.LastName,
		customer.Email, customer.Phone, customer.Company, customer.Status)
	
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	customer.ID = int(id)
	customer.CreatedAt = time.Now()
	customer.UpdatedAt = time.Now()
	
	return nil
}

// LogActivity creates a new activity record
func (dm *DatabaseManager) LogActivity(activity *Activity) error {
	query := `INSERT INTO activities (customer_id, user_id, type, subject, notes)
			  VALUES (?, ?, ?, ?, ?)`
	
	result, err := dm.db.Exec(query, activity.CustomerID, activity.UserID,
		activity.Type, activity.Subject, activity.Notes)
	
	if err != nil {
		return err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	
	activity.ID = int(id)
	activity.CreatedAt = time.Now()
	
	return nil
}

// GetCustomerActivities retrieves all activities for a customer
func (dm *DatabaseManager) GetCustomerActivities(customerID int) ([]Activity, error) {
	query := `SELECT a.id, a.customer_id, a.user_id, a.type, a.subject, a.notes, a.created_at,
			         u.username, u.first_name, u.last_name
			  FROM activities a
			  JOIN users u ON a.user_id = u.id
			  WHERE a.customer_id = ?
			  ORDER BY a.created_at DESC`
	
	rows, err := dm.db.Query(query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var activities []Activity
	for rows.Next() {
		var activity Activity
		var createdAt string
		var username, firstName, lastName string
		
		err := rows.Scan(&activity.ID, &activity.CustomerID, &activity.UserID,
			&activity.Type, &activity.Subject, &activity.Notes, &createdAt,
			&username, &firstName, &lastName)
		
		if err != nil {
			continue
		}
		
		activity.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		activities = append(activities, activity)
	}
	
	return activities, nil
}

// AuditLog creates an audit log entry
func (dm *DatabaseManager) AuditLog(userID int, username, action, tableName string, recordID int, 
	oldValues, newValues interface{}, ipAddress, userAgent string) error {
	
	var oldJSON, newJSON string
	
	if oldValues != nil {
		if data, err := json.Marshal(oldValues); err == nil {
			oldJSON = string(data)
		}
	}
	
	if newValues != nil {
		if data, err := json.Marshal(newValues); err == nil {
			newJSON = string(data)
		}
	}
	
	query := `INSERT INTO audit_log (user_id, username, action, table_name, record_id, 
			  old_values, new_values, ip_address, user_agent)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := dm.db.Exec(query, userID, username, action, tableName, recordID,
		oldJSON, newJSON, ipAddress, userAgent)
	
	return err
}

// GetDashboardStats returns statistics for the dashboard
func (dm *DatabaseManager) GetDashboardStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Count active customers
	var activeCustomers int
	err := dm.db.QueryRow("SELECT COUNT(*) FROM customers WHERE status = 'active'").Scan(&activeCustomers)
	if err != nil {
		return nil, err
	}
	stats["active_customers"] = activeCustomers
	
	// Count pending payments
	var pendingPayments int
	err = dm.db.QueryRow("SELECT COUNT(*) FROM payments WHERE approved = FALSE").Scan(&pendingPayments)
	if err != nil {
		return nil, err
	}
	stats["pending_payments"] = pendingPayments
	
	// Calculate total pending amount
	var totalPending float64
	err = dm.db.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payments WHERE approved = FALSE").Scan(&totalPending)
	if err != nil {
		return nil, err
	}
	stats["pending_amount"] = totalPending
	
	// Calculate this month's approved payments
	var monthlyRevenue float64
	err = dm.db.QueryRow(`SELECT COALESCE(SUM(amount), 0) FROM payments 
						  WHERE approved = TRUE AND 
						  date(approved_at) >= date('now', 'start of month')`).Scan(&monthlyRevenue)
	if err != nil {
		return nil, err
	}
	stats["monthly_revenue"] = monthlyRevenue
	
	// Count total comments this month
	var monthlyComments int
	err = dm.db.QueryRow(`SELECT COUNT(*) FROM comments 
						  WHERE date(created) >= date('now', 'start of month')`).Scan(&monthlyComments)
	if err != nil {
		return nil, err
	}
	stats["monthly_comments"] = monthlyComments
	
	return stats, nil
}

// SearchCustomers searches customers by name, email, or company
func (dm *DatabaseManager) SearchCustomers(searchTerm string, limit int) ([]Customer, error) {
	query := `SELECT id, first_name, last_name, email, phone, company, status, created_at, updated_at
			  FROM customers 
			  WHERE first_name LIKE ? OR last_name LIKE ? OR email LIKE ? OR company LIKE ?
			  ORDER BY first_name, last_name
			  LIMIT ?`
	
	searchPattern := "%" + searchTerm + "%"
	rows, err := dm.db.Query(query, searchPattern, searchPattern, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var customers []Customer
	for rows.Next() {
		var customer Customer
		var createdAt, updatedAt string
		
		err := rows.Scan(&customer.ID, &customer.FirstName, &customer.LastName,
			&customer.Email, &customer.Phone, &customer.Company, &customer.Status,
			&createdAt, &updatedAt)
		
		if err != nil {
			continue
		}
		
		customer.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		customer.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedAt)
		customers = append(customers, customer)
	}
	
	return customers, nil
}

// CleanupDatabase performs maintenance tasks
func (dm *DatabaseManager) CleanupDatabase() error {
	// Clean up expired CSRF tokens
	_, err := dm.db.Exec("DELETE FROM csrf_tokens WHERE expires_at < datetime('now') OR used = TRUE")
	if err != nil {
		log.Printf("Error cleaning CSRF tokens: %v", err)
	}
	
	// Clean up old audit logs (keep last 30 days)
	_, err = dm.db.Exec("DELETE FROM audit_log WHERE created_at < datetime('now', '-30 days')")
	if err != nil {
		log.Printf("Error cleaning audit logs: %v", err)
	}
	
	// Clean up inactive sessions (older than 24 hours)
	_, err = dm.db.Exec("DELETE FROM sessions WHERE expires_at < datetime('now') OR created_at < datetime('now', '-24 hours')")
	if err != nil {
		log.Printf("Error cleaning sessions: %v", err)
	}
	
	// Vacuum database to reclaim space
	_, err = dm.db.Exec("VACUUM")
	if err != nil {
		log.Printf("Error vacuuming database: %v", err)
	}
	
	return nil
}

// BackupDatabase creates a simple backup by exporting data
func (dm *DatabaseManager) BackupDatabase() error {
	fmt.Println("Database backup would be implemented here...")
	fmt.Println("In production, this would:")
	fmt.Println("1. Create a full database dump")
	fmt.Println("2. Compress the backup file")
	fmt.Println("3. Store it in a secure location")
	fmt.Println("4. Rotate old backups")
	return nil
}

// ValidateDatabase checks database integrity
func (dm *DatabaseManager) ValidateDatabase() error {
	// Check for orphaned records
	var orphanedActivities int
	err := dm.db.QueryRow(`SELECT COUNT(*) FROM activities a 
						   LEFT JOIN customers c ON a.customer_id = c.id 
						   WHERE c.id IS NULL`).Scan(&orphanedActivities)
	if err != nil {
		return err
	}
	
	if orphanedActivities > 0 {
		fmt.Printf("Warning: Found %d orphaned activities\n", orphanedActivities)
	}
	
	var orphanedPayments int
	err = dm.db.QueryRow(`SELECT COUNT(*) FROM payments p 
						  WHERE p.approved_by IS NOT NULL AND p.approved_by NOT IN (SELECT username FROM users)`).Scan(&orphanedPayments)
	if err != nil {
		return err
	}
	
	if orphanedPayments > 0 {
		fmt.Printf("Warning: Found %d payments with invalid approved_by references\n", orphanedPayments)
	}
	
	fmt.Println("Database validation completed")
	return nil
}
