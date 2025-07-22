package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
)

// Global variables
var (
	db    *sql.DB
	store = sessions.NewCookieStore([]byte("super-secret-key"))
)

// User represents a user in the system
type User struct {
	ID        int
	Username  string
	Password  string
	Email     string
	FirstName string
	LastName  string
	Role      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Comment represents a comment in the system
type Comment struct {
	ID        int
	Username  string
	Content   string
	Created   time.Time
	IsPublic  bool
	MessageType string
}

// Payment represents a payment approval
type Payment struct {
	ID          int
	Amount      float64
	Approved    bool
	Email       string
	Description string
	CreatedAt   time.Time
	ApprovedAt  *time.Time
	ApprovedBy  string
}

// Customer represents a customer in the CRM
type Customer struct {
	ID          int
	FirstName   string
	LastName    string
	Email       string
	Phone       string
	Company     string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Activity represents customer activity tracking
type Activity struct {
	ID         int
	CustomerID int
	UserID     int
	Type       string
	Subject    string
	Notes      string
	CreatedAt  time.Time
}

// CSRFToken represents CSRF tokens for security
type CSRFToken struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	Used      bool
}

func main() {
	// Initialize database
	initDB()
	defer db.Close()

	// Setup routes
	router := mux.NewRouter()

	// Static files
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))

	// Routes
	router.HandleFunc("/", homeHandler).Methods("GET")
	router.HandleFunc("/login", loginHandler).Methods("GET", "POST")
	router.HandleFunc("/auth", vulnerableLoginHandler).Methods("GET", "POST") // Legacy auth system
	router.HandleFunc("/dashboard", dashboardHandler).Methods("GET")
	router.HandleFunc("/feedback", vulnerableCommentsHandler).Methods("GET", "POST") // Customer feedback
	router.HandleFunc("/feedback-secure", secureCommentsHandler).Methods("GET", "POST") // Secure customer feedback
	router.HandleFunc("/approvals", paymentsHandler).Methods("GET", "POST")
	router.HandleFunc("/finance", vulnerablePaymentsHandler).Methods("GET", "POST") // Legacy finance system
	router.HandleFunc("/logout", logoutHandler).Methods("GET")
	// Note: /partners route moved to separate malicious-server.go on port 9999

	fmt.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./crm.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create users table with enhanced schema
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		first_name TEXT NOT NULL DEFAULT '',
		last_name TEXT NOT NULL DEFAULT '',
		role TEXT NOT NULL DEFAULT 'user',
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create customers table
	createCustomersTable := `
	CREATE TABLE IF NOT EXISTS customers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		first_name TEXT NOT NULL,
		last_name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		phone TEXT,
		company TEXT,
		status TEXT NOT NULL DEFAULT 'active',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	// Create comments table with enhanced schema
	createCommentsTable := `
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		content TEXT NOT NULL,
		created DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_public BOOLEAN DEFAULT TRUE,
		message_type TEXT DEFAULT 'general',
		FOREIGN KEY (username) REFERENCES users(username)
	);`

	// Create payments table with enhanced schema
	createPaymentsTable := `
	CREATE TABLE IF NOT EXISTS payments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		amount REAL NOT NULL,
		approved BOOLEAN DEFAULT FALSE,
		email TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		approved_at DATETIME,
		approved_by TEXT,
		FOREIGN KEY (approved_by) REFERENCES users(username)
	);`

	// Create activities table for customer tracking
	createActivitiesTable := `
	CREATE TABLE IF NOT EXISTS activities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		customer_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		type TEXT NOT NULL,
		subject TEXT NOT NULL,
		notes TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (customer_id) REFERENCES customers(id),
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	// Create CSRF tokens table for security
	createCSRFTokensTable := `
	CREATE TABLE IF NOT EXISTS csrf_tokens (
		token TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		used BOOLEAN DEFAULT FALSE
	);`

	// Create sessions table for session management
	createSessionsTable := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		expires_at DATETIME NOT NULL,
		is_active BOOLEAN DEFAULT TRUE,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (username) REFERENCES users(username)
	);`

	// Create audit_log table for security tracking
	createAuditLogTable := `
	CREATE TABLE IF NOT EXISTS audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		username TEXT,
		action TEXT NOT NULL,
		table_name TEXT,
		record_id INTEGER,
		old_values TEXT,
		new_values TEXT,
		ip_address TEXT,
		user_agent TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);`

	// Execute table creation
	tables := []string{
		createUsersTable,
		createCustomersTable,
		createCommentsTable,
		createPaymentsTable,
		createActivitiesTable,
		createCSRFTokensTable,
		createSessionsTable,
		createAuditLogTable,
	}

	for _, tableSQL := range tables {
		_, err = db.Exec(tableSQL)
		if err != nil {
			log.Fatalf("Error creating table: %v", err)
		}
	}

	// Create indexes for better performance
	createIndexes()

	// Insert sample data
	insertSampleData()
}

func createIndexes() {
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_users_username ON users(username)",
		"CREATE INDEX IF NOT EXISTS idx_users_email ON users(email)",
		"CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email)",
		"CREATE INDEX IF NOT EXISTS idx_customers_company ON customers(company)",
		"CREATE INDEX IF NOT EXISTS idx_comments_username ON comments(username)",
		"CREATE INDEX IF NOT EXISTS idx_comments_created ON comments(created)",
		"CREATE INDEX IF NOT EXISTS idx_payments_email ON payments(email)",
		"CREATE INDEX IF NOT EXISTS idx_payments_approved ON payments(approved)",
		"CREATE INDEX IF NOT EXISTS idx_activities_customer_id ON activities(customer_id)",
		"CREATE INDEX IF NOT EXISTS idx_activities_user_id ON activities(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_csrf_tokens_expires_at ON csrf_tokens(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id)",
		"CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at)",
	}

	for _, indexSQL := range indexes {
		_, err := db.Exec(indexSQL)
		if err != nil {
			log.Printf("Error creating index: %v", err)
		}
	}
}

func insertSampleData() {
	// Insert sample users with enhanced data
	users := []User{
		{Username: "sarah.connor", Password: "Welcome123!", Email: "sarah.connor@globomantics.com", FirstName: "Sarah", LastName: "Connor", Role: "admin"},
		{Username: "john.doe", Password: "Password2024", Email: "john.doe@globomantics.com", FirstName: "John", LastName: "Doe", Role: "manager"},
		{Username: "mary.smith", Password: "Secure456", Email: "mary.smith@globomantics.com", FirstName: "Mary", LastName: "Smith", Role: "user"},
		{Username: "admin", Password: "GlobalAdmin2024", Email: "admin@globomantics.com", FirstName: "System", LastName: "Administrator", Role: "admin"},
		{Username: "demo", Password: "Demo2024!", Email: "demo@globomantics.com", FirstName: "Demo", LastName: "User", Role: "user"},
	}

	for _, user := range users {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", user.Username).Scan(&exists)
		if err != nil {
			log.Printf("Error checking user existence: %v", err)
			continue
		}

		if !exists {
			_, err = db.Exec(`INSERT INTO users (username, password, email, first_name, last_name, role) 
				VALUES (?, ?, ?, ?, ?, ?)`,
				user.Username, user.Password, user.Email, user.FirstName, user.LastName, user.Role)
			if err != nil {
				log.Printf("Error inserting user: %v", err)
			}
		}
	}

	// Insert sample customers
	customers := []Customer{
		{FirstName: "Alice", LastName: "Johnson", Email: "alice.johnson@techcorp.com", Phone: "555-0101", Company: "TechCorp Industries", Status: "active"},
		{FirstName: "Bob", LastName: "Williams", Email: "bob.williams@innovatetech.com", Phone: "555-0102", Company: "InnovateTech Solutions", Status: "active"},
		{FirstName: "Carol", LastName: "Davis", Email: "carol.davis@digitalsolutions.com", Phone: "555-0103", Company: "Digital Solutions LLC", Status: "active"},
		{FirstName: "David", LastName: "Brown", Email: "david.brown@startupventures.com", Phone: "555-0104", Company: "Startup Ventures", Status: "prospect"},
		{FirstName: "Eva", LastName: "Miller", Email: "eva.miller@futuretech.com", Phone: "555-0105", Company: "FutureTech Corp", Status: "active"},
	}

	for _, customer := range customers {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM customers WHERE email = ?)", customer.Email).Scan(&exists)
		if err != nil {
			log.Printf("Error checking customer existence: %v", err)
			continue
		}

		if !exists {
			_, err = db.Exec(`INSERT INTO customers (first_name, last_name, email, phone, company, status) 
				VALUES (?, ?, ?, ?, ?, ?)`,
				customer.FirstName, customer.LastName, customer.Email, customer.Phone, customer.Company, customer.Status)
			if err != nil {
				log.Printf("Error inserting customer: %v", err)
			}
		}
	}

	// Insert sample payments with enhanced data
	payments := []Payment{
		{Amount: 15750.00, Email: "accounting@techcorp.com", Description: "Q4 Software License Renewal - Enterprise Package"},
		{Amount: 8250.50, Email: "billing@innovatetech.com", Description: "Consulting Services - System Integration Project"},
		{Amount: 12000.75, Email: "payments@digitalsolutions.com", Description: "Hardware Procurement - Server Infrastructure"},
		{Amount: 6500.00, Email: "finance@startupventures.com", Description: "Training and Certification Program"},
		{Amount: 22300.25, Email: "procurement@futuretech.com", Description: "Custom Development - CRM Module Enhancement"},
	}

	for _, payment := range payments {
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM payments WHERE amount = ? AND email = ?)", payment.Amount, payment.Email).Scan(&exists)
		if err != nil {
			log.Printf("Error checking payment existence: %v", err)
			continue
		}

		if !exists {
			_, err = db.Exec(`INSERT INTO payments (amount, email, description) VALUES (?, ?, ?)`,
				payment.Amount, payment.Email, payment.Description)
			if err != nil {
				log.Printf("Error inserting payment: %v", err)
			}
		}
	}

	// Insert sample activities
	activities := []string{
		"INSERT OR IGNORE INTO activities (customer_id, user_id, type, subject, notes) VALUES (1, 1, 'call', 'Initial consultation call', 'Discussed requirements for new CRM system')",
		"INSERT OR IGNORE INTO activities (customer_id, user_id, type, subject, notes) VALUES (2, 2, 'email', 'Follow-up proposal', 'Sent detailed proposal for integration services')",
		"INSERT OR IGNORE INTO activities (customer_id, user_id, type, subject, notes) VALUES (3, 1, 'meeting', 'Project kickoff meeting', 'Met with stakeholders to discuss project timeline')",
		"INSERT OR IGNORE INTO activities (customer_id, user_id, type, subject, notes) VALUES (4, 3, 'call', 'Discovery call', 'Initial needs assessment for potential client')",
		"INSERT OR IGNORE INTO activities (customer_id, user_id, type, subject, notes) VALUES (5, 2, 'demo', 'Product demonstration', 'Showcased CRM features and capabilities')",
	}

	for _, activitySQL := range activities {
		_, err := db.Exec(activitySQL)
		if err != nil {
			log.Printf("Error inserting activity: %v", err)
		}
	}

	// Insert sample comments for team chat
	comments := []string{
		"INSERT OR IGNORE INTO comments (username, content, message_type) VALUES ('sarah.connor', 'Welcome to the team chat! Please use this for internal communications.', 'announcement')",
		"INSERT OR IGNORE INTO comments (username, content, message_type) VALUES ('john.doe', 'Great work on the Q4 sales numbers everyone!', 'general')",
		"INSERT OR IGNORE INTO comments (username, content, message_type) VALUES ('mary.smith', 'Don''t forget about the client meeting tomorrow at 2 PM.', 'reminder')",
	}

	for _, commentSQL := range comments {
		_, err := db.Exec(commentSQL)
		if err != nil {
			log.Printf("Error inserting comment: %v", err)
		}
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in
	session, _ := store.Get(r, "session-name")
	authenticated, ok := session.Values["authenticated"].(bool)
	if ok && authenticated {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Globomantics CRM - Customer Relationship Management</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .header {
            background: rgba(255,255,255,0.95);
            padding: 20px 0;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .header-content {
            max-width: 1200px;
            margin: 0 auto;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 20px;
        }
        .logo {
            display: flex;
            align-items: center;
            font-size: 24px;
            font-weight: bold;
            color: #333;
        }
        .logo::before {
            content: "";
            margin-right: 10px;
            font-size: 28px;
        }
        .main-content {
            max-width: 1200px;
            margin: 50px auto;
            padding: 0 20px;
        }
        .hero {
            text-align: center;
            color: white;
            margin-bottom: 50px;
        }
        .hero h1 {
            font-size: 48px;
            margin-bottom: 20px;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.3);
        }
        .hero p {
            font-size: 20px;
            margin-bottom: 30px;
            opacity: 0.9;
        }
        .login-card {
            background: white;
            padding: 40px;
            border-radius: 15px;
            box-shadow: 0 15px 35px rgba(0,0,0,0.1);
            max-width: 400px;
            margin: 0 auto;
        }
        .login-btn {
            background: #667eea;
            color: white;
            padding: 15px 30px;
            border: none;
            border-radius: 8px;
            font-size: 16px;
            cursor: pointer;
            width: 100%;
            margin-bottom: 15px;
            transition: background 0.3s;
        }
        .login-btn:hover {
            background: #5a6fd8;
        }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 30px;
            margin-top: 60px;
        }
        .feature-card {
            background: rgba(255,255,255,0.95);
            padding: 30px;
            border-radius: 15px;
            text-align: center;
            box-shadow: 0 10px 25px rgba(0,0,0,0.1);
        }
        .feature-icon {
            font-size: 48px;
            margin-bottom: 20px;
        }
        .feature-card h3 {
            color: #333;
            margin-bottom: 15px;
            font-size: 22px;
        }
        .feature-card p {
            color: #666;
            line-height: 1.6;
        }
        .footer {
            text-align: center;
            color: rgba(255,255,255,0.8);
            margin-top: 60px;
            padding: 20px;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <div class="logo">Globomantics CRM</div>
            <div>
                <a href="/login" style="text-decoration: none; color: #667eea; font-weight: 500;">Employee Login</a>
            </div>
        </div>
    </div>

    <div class="main-content">
        <div class="hero">
            <h1>Welcome to Globomantics</h1>
            <p>Your Complete Customer Relationship Management Solution</p>
            
            <div class="login-card">
                <h3 style="margin-top: 0; color: #333;">Employee Access</h3>
                <p style="color: #666; margin-bottom: 30px;">Sign in to access customer data, manage payments, and collaborate with your team.</p>
                <a href="/login">
                    <button class="login-btn">Access Employee Portal</button>
                </a>
                <p style="font-size: 14px; color: #888; margin-top: 20px;">
                    Need help? Contact IT Support
                </p>
            </div>
        </div>

        <div class="features">
            <div class="feature-card">
                <div class="feature-icon">CRM</div>
                <h3>Customer Management</h3>
                <p>Comprehensive customer profiles, interaction history, and relationship tracking to help you build stronger connections with your clients.</p>
            </div>
            
            <div class="feature-card">
                <div class="feature-icon">PAY</div>
                <h3>Payment Processing</h3>
                <p>Secure payment approval workflows, transaction monitoring, and financial reporting to streamline your billing operations.</p>
            </div>
            
            <div class="feature-card">
                <div class="feature-icon">CHAT</div>
                <h3>Team Collaboration</h3>
                <p>Internal messaging, project notes, and team communication tools to keep everyone aligned on customer initiatives.</p>
            </div>
        </div>
    </div>

    <div class="footer">
        <p>&copy; 2024 Globomantics Corporation. All rights reserved. | Privacy Policy | Terms of Service</p>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(tmpl))
}
