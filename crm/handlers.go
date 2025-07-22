	package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

// generateCSRFToken generates a random CSRF token and stores it in database
func generateCSRFToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := base64.URLEncoding.EncodeToString(bytes)
	
	// Store token in database with 1 hour expiry
	expiresAt := time.Now().Add(time.Hour)
	_, err := db.Exec("INSERT INTO csrf_tokens (token, expires_at) VALUES (?, ?)", token, expiresAt)
	if err != nil {
		// If database insert fails, fall back to memory storage temporarily
		fmt.Printf("Error storing CSRF token in database: %v\n", err)
	}
	
	return token
}

// validateCSRFToken validates a CSRF token against database
func validateCSRFToken(token string) bool {
	var expiresAt time.Time
	var used bool
	
	err := db.QueryRow("SELECT expires_at, used FROM csrf_tokens WHERE token = ?", token).Scan(&expiresAt, &used)
	if err != nil {
		return false
	}
	
	// Check if token is valid and not expired
	if used || time.Now().After(expiresAt) {
		// Clean up invalid token
		db.Exec("DELETE FROM csrf_tokens WHERE token = ?", token)
		return false
	}
	
	// Mark token as used (single-use)
	_, err = db.Exec("UPDATE csrf_tokens SET used = TRUE WHERE token = ?", token)
	if err != nil {
		return false
	}
	
	return true
}

// cleanupExpiredTokens removes expired CSRF tokens from database
func cleanupExpiredTokens() {
	_, err := db.Exec("DELETE FROM csrf_tokens WHERE expires_at < datetime('now') OR used = TRUE")
	if err != nil {
		fmt.Printf("Error cleaning up expired CSRF tokens: %v\n", err)
	}
}

// Legacy Authentication System (vulnerableLoginHandler)
func vulnerableLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Legacy Auth System - Globomantics CRM</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            background: #f5f5f5;
        }
        .container { max-width: 500px; margin: 50px auto; padding: 20px; }
        .login-card {
            background: white;
            padding: 40px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { font-size: 24px; font-weight: bold; color: #333; margin-bottom: 10px; }
        .subtitle { color: #666; font-size: 14px; }
        .form-group { margin-bottom: 20px; }
        label { display: block; margin-bottom: 8px; font-weight: 500; color: #333; }
        input[type="text"], input[type="password"] { 
            width: 100%; 
            padding: 12px; 
            border: 1px solid #ddd; 
            border-radius: 4px; 
            font-size: 16px;
            box-sizing: border-box;
        }
        .login-btn { 
            background: #007bff; 
            color: white; 
            padding: 12px 30px; 
            border: none; 
            border-radius: 4px; 
            font-size: 16px; 
            cursor: pointer; 
            width: 100%;
        }
        .login-btn:hover { background: #0056b3; }
        .nav { margin-bottom: 20px; text-align: center; }
        .nav a { text-decoration: none; color: #007bff; }
        .notice {
            background: #fff3cd;
            border: 1px solid #ffeaa7;
            color: #856404;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="nav">
            <a href="/">← Back to Main Site</a>
        </div>
        
        <div class="login-card">
            <div class="header">
                <div class="logo">Globomantics</div>
                <div class="subtitle">Legacy Authentication System</div>
            </div>
            
            <div class="notice">
                <strong>Notice:</strong> This is our legacy authentication system. 
                For improved security, please use the <a href="/login">new login portal</a>.
            </div>

            <form method="post">
                <div class="form-group">
                    <label>Employee ID:</label>
                    <input type="text" name="username" placeholder="Enter your employee ID" required>
                </div>
                <div class="form-group">
                    <label>Password:</label>
                    <input type="password" name="password" placeholder="Enter your password" required>
                </div>
                <button type="submit" class="login-btn">Access Legacy System</button>
            </form>

            <div style="margin-top: 30px; text-align: center; font-size: 14px; color: #666;">
                <p>Having trouble? <a href="/login">Try the new system</a></p>
            </div>
        </div>
    </div>
</body>
</html>`

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(tmpl))
		return
	}

	// POST - Vulnerable implementation using string concatenation
	username := r.FormValue("username")
	password := r.FormValue("password")

	// VULNERABLE: Direct string concatenation (SQL Injection vulnerability)
	query := fmt.Sprintf("SELECT id, username, email FROM users WHERE username = '%s' AND password = '%s'", username, password)
	
	fmt.Printf("Executing vulnerable query: %s\n", query) // For demonstration

	row := db.QueryRow(query)
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Email)

	if err != nil {
		http.Error(w, "Login failed: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Set session
	session, _ := store.Get(r, "session-name")
	session.Values["authenticated"] = true
	session.Values["username"] = user.Username
	session.Save(r, w)
	
	fmt.Printf("DEBUG: Vulnerable login stored username: %s\n", user.Username)

	// Success response
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Access Granted - Legacy System</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 40px; 
            background: #f5f5f5;
        }
        .container { max-width: 600px; margin: 0 auto; }
        .success { 
            background: #d4edda; 
            color: #155724; 
            padding: 20px; 
            border: 1px solid #c3e6cb; 
            border-radius: 8px;
            margin-bottom: 20px;
        }
        .user-info {
            background: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .nav a { text-decoration: none; color: #007bff; margin-right: 15px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="success">
            <h2>✅ Authentication Successful</h2>
            <p>Welcome to the legacy system</p>
        </div>
        
        <div class="user-info">
            <h3>Employee Information</h3>
            <p><strong>Employee ID:</strong> %s</p>
            <p><strong>Email:</strong> %s</p>
            <p><strong>System:</strong> Legacy Authentication Portal</p>
        </div>
        
        <div style="margin-top: 20px;">
            <a href="/dashboard">Go to Dashboard</a> | 
            <a href="/logout">Logout</a> |
            <a href="/auth">Back to Legacy Login</a> |
            <a href="/">Home</a>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, tmpl, user.Username, user.Email)
}

// Main Login System (Secure)
func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Employee Portal - Globomantics CRM</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
        }
        .login-container {
            background: white;
            padding: 40px;
            border-radius: 15px;
            box-shadow: 0 15px 35px rgba(0,0,0,0.1);
            max-width: 400px;
            width: 100%;
        }
        .header { text-align: center; margin-bottom: 30px; }
        .logo { 
            font-size: 28px; 
            font-weight: bold; 
            color: #333; 
            margin-bottom: 10px; 
        }
        .logo::before { content: ""; margin-right: 10px; }
        .subtitle { color: #666; font-size: 16px; }
        .form-group { margin-bottom: 20px; }
        label { 
            display: block; 
            margin-bottom: 8px; 
            font-weight: 500; 
            color: #333; 
        }
        input[type="text"], input[type="password"] { 
            width: 100%; 
            padding: 12px; 
            border: 2px solid #e1e5e9; 
            border-radius: 8px; 
            font-size: 16px;
            box-sizing: border-box;
            transition: border-color 0.3s;
        }
        input[type="text"]:focus, input[type="password"]:focus {
            outline: none;
            border-color: #667eea;
        }
        .login-btn { 
            background: #667eea; 
            color: white; 
            padding: 12px 30px; 
            border: none; 
            border-radius: 8px; 
            font-size: 16px; 
            cursor: pointer; 
            width: 100%;
            margin-bottom: 15px;
            transition: background 0.3s;
        }
        .login-btn:hover { background: #5a6fd8; }
        .nav { text-align: center; margin-bottom: 20px; }
        .nav a { text-decoration: none; color: rgba(255,255,255,0.9); }
        .help-links {
            text-align: center;
            margin-top: 20px;
            font-size: 14px;
        }
        .help-links a { color: #667eea; text-decoration: none; margin: 0 10px; }
    </style>
</head>
<body>
    <div>
        <div class="nav">
            <a href="/">← Back to Main Site</a>
        </div>
        
        <div class="login-container">
            <div class="header">
                <div class="logo">Globomantics</div>
                <div class="subtitle">Employee Portal</div>
            </div>

            <form method="post">
                <div class="form-group">
                    <label>Employee ID:</label>
                    <input type="text" name="username" placeholder="Enter your employee ID" required>
                </div>
                <div class="form-group">
                    <label>Password:</label>
                    <input type="password" name="password" placeholder="Enter your password" required>
                </div>
                <button type="submit" class="login-btn">Sign In</button>
            </form>

            <div class="help-links">
                <a href="/auth">Legacy System</a> |
                <a href="#">Forgot Password?</a> |
                <a href="#">Help</a>
            </div>
        </div>
    </div>
</body>
</html>`

		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(tmpl))
		return
	}

	// POST - Secure implementation using parameterized queries
	username := r.FormValue("username")
	password := r.FormValue("password")

	// SECURE: Using parameterized queries
	query := "SELECT id, username, email FROM users WHERE username = ? AND password = ?"
	row := db.QueryRow(query, username, password)
	
	var user User
	err := row.Scan(&user.ID, &user.Username, &user.Email)

	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Set session
	session, _ := store.Get(r, "session-name")
	session.Values["authenticated"] = true
	session.Values["username"] = user.Username
	session.Save(r, w)
	
	fmt.Printf("DEBUG: Login stored username: %s\n", user.Username)

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// Customer Feedback System (vulnerableCommentsHandler)
func vulnerableCommentsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in
	session, _ := store.Get(r, "session-name")
	authenticated, ok := session.Values["authenticated"].(bool)
	if !ok || !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		content := r.FormValue("content")

		_, err := db.Exec("INSERT INTO comments (username, content) VALUES (?, ?)", username, content)
		if err != nil {
			http.Error(w, "Error saving feedback", http.StatusInternalServerError)
			return
		}
	}

	// Get comments
	rows, err := db.Query("SELECT id, username, content, created FROM comments ORDER BY created DESC")
	if err != nil {
		http.Error(w, "Error fetching feedback", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.Username, &comment.Content, &comment.Created)
		if err != nil {
			continue
		}
		comments = append(comments, comment)
	}

	// VULNERABLE: Direct output without encoding
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Customer Feedback - Globomantics CRM</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            background: #f8f9fa;
        }
        .header {
            background: white;
            padding: 20px 0;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        .header-content {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 20px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .logo { font-size: 20px; font-weight: bold; color: #333; }
        .container { max-width: 800px; margin: 0 auto; padding: 0 20px; }
        .feedback-form {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        .feedback-item { 
            background: white; 
            padding: 20px; 
            margin-bottom: 15px; 
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border-left: 4px solid #007bff; 
        }
        .form-group { margin-bottom: 20px; }
        label { 
            display: block; 
            margin-bottom: 8px; 
            font-weight: 500; 
            color: #333;
        }
        input, textarea { 
            width: 100%; 
            padding: 12px; 
            border: 1px solid #ddd; 
            border-radius: 4px;
            font-size: 16px;
            box-sizing: border-box;
        }
        textarea { min-height: 100px; resize: vertical; }
        .submit-btn { 
            background: #007bff; 
            color: white; 
            padding: 12px 30px; 
            border: none; 
            border-radius: 4px;
            cursor: pointer; 
            font-size: 16px;
        }
        .submit-btn:hover { background: #0056b3; }
        .nav a { color: #007bff; text-decoration: none; margin-right: 20px; }
        .timestamp { color: #666; font-size: 14px; }
        .customer-name { font-weight: bold; color: #333; }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <div class="logo">Globomantics CRM</div>
            <div>
                <a href="/dashboard">Dashboard</a>
                <a href="/logout">Logout</a>
            </div>
        </div>
    </div>

    <div class="container">
        <h2>Customer Feedback Portal</h2>
        <p>Collect and review customer feedback to improve our services.</p>
        
        <div class="feedback-form">
            <h3>Submit New Feedback</h3>
            <form method="post">
                <div class="form-group">
                    <label>Customer Name:</label>
                    <input type="text" name="username" placeholder="Enter customer name" required>
                </div>
                <div class="form-group">
                    <label>Feedback:</label>
                    <textarea name="content" placeholder="Enter customer feedback..." required></textarea>
                </div>
                <button type="submit" class="submit-btn">Submit Feedback</button>
            </form>
        </div>

        <h3>Recent Customer Feedback:</h3>`

	for _, comment := range comments {
		// VULNERABLE: Direct output without HTML encoding
		tmpl += fmt.Sprintf(`
        <div class="feedback-item">
            <div class="customer-name">%s</div>
            <div class="timestamp">%s</div>
            <div style="margin-top: 10px;">%s</div>
        </div>`, comment.Username, comment.Created.Format("January 2, 2006 at 3:04 PM"), comment.Content)
	}

	tmpl += `
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(tmpl))
}

// Team Communication System (commentsHandler)
func commentsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in
	session, _ := store.Get(r, "session-name")
	authenticated, ok := session.Values["authenticated"].(bool)
	if !ok || !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		content := r.FormValue("content")

		_, err := db.Exec("INSERT INTO comments (username, content) VALUES (?, ?)", username, content)
		if err != nil {
			http.Error(w, "Error saving message", http.StatusInternalServerError)
			return
		}
	}

	// Get comments
	rows, err := db.Query("SELECT id, username, content, created FROM comments ORDER BY created DESC")
	if err != nil {
		http.Error(w, "Error fetching messages", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.Username, &comment.Content, &comment.Created)
		if err != nil {
			continue
		}
		comments = append(comments, comment)
	}

	// SECURE: Using html/template for proper encoding
	tmplStr := `
<!DOCTYPE html>
<html>
<head>
    <title>Team Chat - Globomantics CRM</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            background: #f8f9fa;
        }
        .header {
            background: white;
            padding: 20px 0;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        .header-content {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 20px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .logo { font-size: 20px; font-weight: bold; color: #333; }
        .container { max-width: 800px; margin: 0 auto; padding: 0 20px; }
        .chat-form {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        .message { 
            background: white; 
            padding: 20px; 
            margin-bottom: 15px; 
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border-left: 4px solid #28a745; 
        }
        .form-group { margin-bottom: 20px; }
        label { 
            display: block; 
            margin-bottom: 8px; 
            font-weight: 500; 
            color: #333;
        }
        input, textarea { 
            width: 100%; 
            padding: 12px; 
            border: 1px solid #ddd; 
            border-radius: 4px;
            font-size: 16px;
            box-sizing: border-box;
        }
        textarea { min-height: 100px; resize: vertical; }
        .submit-btn { 
            background: #28a745; 
            color: white; 
            padding: 12px 30px; 
            border: none; 
            border-radius: 4px;
            cursor: pointer; 
            font-size: 16px;
        }
        .submit-btn:hover { background: #1e7e34; }
        .nav a { color: #007bff; text-decoration: none; margin-right: 20px; }
        .timestamp { color: #666; font-size: 14px; }
        .employee-name { font-weight: bold; color: #333; }
        .online-indicator {
            display: inline-block;
            width: 8px;
            height: 8px;
            background: #28a745;
            border-radius: 50%;
            margin-right: 8px;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <div class="logo">Globomantics CRM</div>
            <div>
                <a href="/dashboard">Dashboard</a>
                <a href="/logout">Logout</a>
            </div>
        </div>
    </div>

    <div class="container">
        <h2>Team Communication</h2>
        <p>Internal messaging system for team collaboration and project updates.</p>
        
        <div class="chat-form">
            <h3>Post Team Message</h3>
            <form method="post">
                <div class="form-group">
                    <label>Your Name:</label>
                    <input type="text" name="username" placeholder="Enter your name" required>
                </div>
                <div class="form-group">
                    <label>Message:</label>
                    <textarea name="content" placeholder="Type your message..." required></textarea>
                </div>
                <button type="submit" class="submit-btn">Send Message</button>
            </form>
        </div>

        <h3>Team Messages:</h3>
        {{range .}}
        <div class="message">
            <div>
                <span class="online-indicator"></span>
                <span class="employee-name">{{.Username}}</span>
                <span class="timestamp">{{.Created.Format "January 2, 2006 at 3:04 PM"}}</span>
            </div>
            <div style="margin-top: 10px;">{{.Content}}</div>
        </div>
        {{end}}
    </div>
</body>
</html>`

	tmpl := template.Must(template.New("teamchat").Parse(tmplStr))
	tmpl.Execute(w, comments)
}

// Secure Customer Feedback System (secureCommentsHandler)
func secureCommentsHandler(w http.ResponseWriter, r *http.Request) {
	// Add CSP headers for enhanced XSS protection
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	
	// Check if user is logged in
	session, _ := store.Get(r, "session-name")
	authenticated, ok := session.Values["authenticated"].(bool)
	if !ok || !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		username := r.FormValue("username")
		content := r.FormValue("content")

		_, err := db.Exec("INSERT INTO comments (username, content) VALUES (?, ?)", username, content)
		if err != nil {
			http.Error(w, "Error saving feedback", http.StatusInternalServerError)
			return
		}
	}

	// Get comments
	rows, err := db.Query("SELECT id, username, content, created FROM comments ORDER BY created DESC")
	if err != nil {
		http.Error(w, "Error fetching feedback", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.Username, &comment.Content, &comment.Created)
		if err != nil {
			continue
		}
		comments = append(comments, comment)
	}

	// SECURE: Using html/template for proper encoding
	tmplStr := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Secure Customer Feedback - Globomantics CRM</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            background: #f8f9fa;
        }
        .header {
            background: white;
            padding: 20px 0;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        .header-content {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 20px;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .logo { font-size: 20px; font-weight: bold; color: #333; }
        .container { max-width: 800px; margin: 0 auto; padding: 0 20px; }
        .feedback-form {
            background: white;
            padding: 30px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            margin-bottom: 30px;
        }
        .feedback-item { 
            background: white; 
            padding: 20px; 
            margin-bottom: 15px; 
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            border-left: 4px solid #28a745; 
        }
        .form-group { margin-bottom: 20px; }
        label { 
            display: block; 
            margin-bottom: 8px; 
            font-weight: 500; 
            color: #333;
        }
        input, textarea { 
            width: 100%; 
            padding: 12px; 
            border: 1px solid #ddd; 
            border-radius: 4px;
            font-size: 16px;
            box-sizing: border-box;
        }
        textarea { min-height: 100px; resize: vertical; }
        .submit-btn { 
            background: #28a745; 
            color: white; 
            padding: 12px 30px; 
            border: none; 
            border-radius: 4px;
            cursor: pointer; 
            font-size: 16px;
        }
        .submit-btn:hover { background: #1e7e34; }
        .nav a { color: #007bff; text-decoration: none; margin-right: 20px; }
        .timestamp { color: #666; font-size: 14px; }
        .customer-name { font-weight: bold; color: #333; }
        .security-notice {
            background: #d4edda;
            border: 1px solid #c3e6cb;
            color: #155724;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <div class="logo">Globomantics CRM</div>
            <div>
                <a href="/dashboard">Dashboard</a>
                <a href="/logout">Logout</a>
            </div>
        </div>
    </div>

    <div class="container">
        <h2>Secure Customer Feedback Portal</h2>
        <div class="security-notice">
            <strong>🛡️ Security Enhanced:</strong> This feedback system uses secure input handling to prevent XSS attacks.
            User input is properly escaped and sanitized.
        </div>
        
        <div class="feedback-form">
            <h3>Submit New Feedback</h3>
            <form method="post">
                <div class="form-group">
                    <label>Customer Name:</label>
                    <input type="text" name="username" placeholder="Enter customer name" required>
                </div>
                <div class="form-group">
                    <label>Feedback:</label>
                    <textarea name="content" placeholder="Enter customer feedback..." required></textarea>
                </div>
                <button type="submit" class="submit-btn">Submit Feedback</button>
            </form>
        </div>

        <h3>Recent Customer Feedback:</h3>
        {{range .}}
        <div class="feedback-item">
            <div class="customer-name">{{.Username}}</div>
            <div class="timestamp">{{.Created.Format "January 2, 2006 at 3:04 PM"}}</div>
            <div style="margin-top: 10px;">{{.Content}}</div>
        </div>
        {{end}}
    </div>
</body>
</html>`

	tmpl := template.Must(template.New("securefeedback").Parse(tmplStr))
	tmpl.Execute(w, comments)
}
