package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
)

// Legacy Finance System (vulnerablePaymentsHandler)
func vulnerablePaymentsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in
	session, _ := store.Get(r, "session-name")
	authenticated, ok := session.Values["authenticated"].(bool)
	if !ok || !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" {
		// VULNERABLE: No CSRF protection
		action := r.FormValue("action")
		paymentIDStr := r.FormValue("payment_id")
		paymentID, err := strconv.Atoi(paymentIDStr)
		if err != nil {
			http.Error(w, "Invalid payment ID", http.StatusBadRequest)
			return
		}

		if action == "approve" {
			// Approve the payment
			_, err = db.Exec("UPDATE payments SET approved = TRUE, approved_at = datetime('now'), approved_by = ? WHERE id = ?", 
				session.Values["username"].(string), paymentID)
			if err != nil {
				http.Error(w, "Error approving payment", http.StatusInternalServerError)
				return
			}

			// Get the email and description for notification
			var email, description string
			err = db.QueryRow("SELECT email, description FROM payments WHERE id = ?", paymentID).Scan(&email, &description)
			if err == nil {
				fmt.Printf("Payment #%d approved by %s - %s - Notification sent to %s\n", 
					paymentID, session.Values["username"].(string), description, email)
			}
		} else if action == "disapprove" {
			// Disapprove the payment (reverse approval)
			_, err = db.Exec("UPDATE payments SET approved = FALSE, approved_at = NULL, approved_by = NULL WHERE id = ?", paymentID)
			if err != nil {
				http.Error(w, "Error disapproving payment", http.StatusInternalServerError)
				return
			}

			// Get the email and description for notification
			var email, description string
			err = db.QueryRow("SELECT email, description FROM payments WHERE id = ?", paymentID).Scan(&email, &description)
			if err == nil {
				fmt.Printf("Payment #%d disapproved by %s - %s - Status reset to pending\n", 
					paymentID, session.Values["username"].(string), description)
			}
		}
	}

	// Get pending payments with enhanced data
	rows, err := db.Query(`SELECT id, amount, email, approved, description, 
		created_at, approved_at, approved_by FROM payments ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, "Error fetching payments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var payment Payment
		var approvedAt, approvedBy *string
		err := rows.Scan(&payment.ID, &payment.Amount, &payment.Email, &payment.Approved, 
			&payment.Description, &payment.CreatedAt, &approvedAt, &approvedBy)
		if err != nil {
			continue
		}
		if approvedBy != nil {
			payment.ApprovedBy = *approvedBy
		}
		payments = append(payments, payment)
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Legacy Finance System - Globomantics CRM</title>
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
        .payment { 
            background: white; 
            padding: 20px; 
            margin-bottom: 15px; 
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .approved { background: #d4edda; border-left: 4px solid #28a745; }
        .pending { border-left: 4px solid #ffc107; }
        .approve-btn { 
            background: #dc3545; 
            color: white; 
            padding: 8px 16px; 
            border: none; 
            border-radius: 4px;
            cursor: pointer; 
        }
        .approve-btn:hover { background: #c82333; }
        .disapprove-btn { 
            background: #6c757d; 
            color: white; 
            padding: 8px 16px; 
            border: none; 
            border-radius: 4px;
            cursor: pointer; 
        }
        .disapprove-btn:hover { background: #545b62; }
        .notice {
            background: #fff3cd;
            border: 1px solid #ffeaa7;
            color: #856404;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .nav a { color: #007bff; text-decoration: none; margin-right: 20px; }
        .amount { font-size: 18px; font-weight: bold; color: #333; }
        .vendor { color: #666; }
        .status { margin-top: 10px; }
        .status.approved { color: #28a745; font-weight: bold; }
        .status.pending { color: #ffc107; font-weight: bold; }
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
        <h2>Legacy Finance System</h2>
        <p>Welcome, <strong>%s</strong>!</p>

        <div class="notice">
            <strong>Notice:</strong> This is our legacy finance system. 
            For enhanced security features, please use the <a href="/approvals">new approval system</a>.
        </div>

        <h3>Vendor Payments:</h3>`

	username := session.Values["username"].(string)

	for _, payment := range payments {
		fmt.Printf("DEBUG: Payment #%d - Amount: $%.2f - Approved: %t - Email: %s\n", 
			payment.ID, payment.Amount, payment.Approved, payment.Email)
		
		statusClass := "pending"
		statusText := "Pending Approval"
		if payment.Approved {
			statusClass = "approved"
			statusText = "Approved"
		}

		tmpl += fmt.Sprintf(`
        <div class="payment %s">
            <div class="amount">Payment #%d - $%.2f</div>
            <div class="vendor">Vendor: %s</div>
            <div class="status %s">%s</div>`, 
			statusClass, payment.ID, payment.Amount, payment.Email, statusClass, statusText)

		if !payment.Approved {
			tmpl += fmt.Sprintf(`
            <form method="post" style="margin-top: 15px;">
                <input type="hidden" name="payment_id" value="%d">
                <input type="hidden" name="action" value="approve">
                <button type="submit" class="approve-btn">Approve Payment</button>
            </form>`, payment.ID)
		} else {
			tmpl += fmt.Sprintf(`
            <form method="post" style="margin-top: 15px;">
                <input type="hidden" name="payment_id" value="%d">
                <input type="hidden" name="action" value="disapprove">
                <button type="submit" class="disapprove-btn">Disapprove Payment</button>
            </form>
            <!-- DEBUG: Payment #%d is APPROVED, showing disapprove button -->`, payment.ID, payment.ID)
		}

		tmpl += `</div>`
	}

	tmpl += `
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, tmpl, username)
}

// Modern Approval System (paymentsHandler)
func paymentsHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is logged in
	session, _ := store.Get(r, "session-name")
	authenticated, ok := session.Values["authenticated"].(bool)
	if !ok || !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Clean up expired tokens periodically
	cleanupExpiredTokens()

	if r.Method == "POST" {
		// SECURE: CSRF protection
		csrfToken := r.FormValue("csrf_token")
		if !validateCSRFToken(csrfToken) {
			http.Error(w, "Security validation failed. Please try again.", http.StatusForbidden)
			return
		}

		action := r.FormValue("action")
		paymentIDStr := r.FormValue("payment_id")
		paymentID, err := strconv.Atoi(paymentIDStr)
		if err != nil {
			http.Error(w, "Invalid payment ID", http.StatusBadRequest)
			return
		}

		if action == "approve" {
			// Approve the payment with audit trail
			_, err = db.Exec("UPDATE payments SET approved = TRUE, approved_at = datetime('now'), approved_by = ? WHERE id = ?", 
				session.Values["username"].(string), paymentID)
			if err != nil {
				http.Error(w, "Error approving payment", http.StatusInternalServerError)
				return
			}
		} else if action == "disapprove" {
			// Disapprove the payment (reverse approval) with audit trail
			_, err = db.Exec("UPDATE payments SET approved = FALSE, approved_at = NULL, approved_by = NULL WHERE id = ?", paymentID)
			if err != nil {
				http.Error(w, "Error disapproving payment", http.StatusInternalServerError)
				return
			}

			// Get the email and description for notification
			var email, description string
			err = db.QueryRow("SELECT email, description FROM payments WHERE id = ?", paymentID).Scan(&email, &description)
			if err == nil {
				fmt.Printf("Payment #%d disapproved by %s - %s - Status reset to pending\n", 
					paymentID, session.Values["username"].(string), description)
			}
		}
	}

	// Generate CSRF token for the form
	csrfToken := generateCSRFToken()

	// Get pending payments with enhanced data
	rows, err := db.Query(`SELECT id, amount, email, approved, description, 
		created_at, approved_at, approved_by FROM payments ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, "Error fetching payments", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		var payment Payment
		var approvedAt, approvedBy *string
		err := rows.Scan(&payment.ID, &payment.Amount, &payment.Email, &payment.Approved, 
			&payment.Description, &payment.CreatedAt, &approvedAt, &approvedBy)
		if err != nil {
			continue
		}
		if approvedBy != nil {
			payment.ApprovedBy = *approvedBy
		}
		fmt.Printf("DEBUG SECURE: Payment #%d - Amount: $%.2f - Approved: %t - Email: %s\n", 
			payment.ID, payment.Amount, payment.Approved, payment.Email)
		payments = append(payments, payment)
	}

	tmplStr := `
<!DOCTYPE html>
<html>
<head>
    <title>Payment Approvals - Globomantics CRM</title>
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
        .payment { 
            background: white; 
            padding: 20px; 
            margin-bottom: 15px; 
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .approved { background: #d4edda; border-left: 4px solid #28a745; }
        .pending { border-left: 4px solid #007bff; }
        .approve-btn { 
            background: #28a745; 
            color: white; 
            padding: 8px 16px; 
            border: none; 
            border-radius: 4px;
            cursor: pointer; 
        }
        .approve-btn:hover { background: #1e7e34; }
        .security-notice {
            background: #e7f3ff;
            border: 1px solid #b8daff;
            color: #004085;
            padding: 15px;
            border-radius: 4px;
            margin-bottom: 20px;
        }
        .nav a { color: #007bff; text-decoration: none; margin-right: 20px; }
        .amount { font-size: 18px; font-weight: bold; color: #333; }
        .vendor { color: #666; }
        .status { margin-top: 10px; }
        .status.approved { color: #28a745; font-weight: bold; }
        .status.pending { color: #007bff; font-weight: bold; }
        .security-token {
            font-family: monospace;
            background: #f8f9fa;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 12px;
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
        <h2>Payment Approval System</h2>
        <p>Welcome, <strong>{{.Username}}</strong>!</p>

        <div class="security-notice">
            <strong>Secure System:</strong> This approval system uses advanced security measures 
            to protect against unauthorized transactions. Each request includes validation token: 
            <span class="security-token">{{.CSRFToken}}</span>
        </div>

        <h3>Vendor Payments:</h3>
        {{range .Payments}}
        <div class="payment {{if .Approved}}approved{{else}}pending{{end}}">
            <div class="amount">Payment #{{.ID}} - ${{printf "%.2f" .Amount}}</div>
            <div class="vendor">Vendor: {{.Email}}</div>
            <div class="status {{if .Approved}}approved{{else}}pending{{end}}">
                {{if .Approved}}Approved{{else}}Pending Approval{{end}}
            </div>
            {{if not .Approved}}
            <form method="post" style="margin-top: 15px;">
                <input type="hidden" name="payment_id" value="{{.ID}}">
                <input type="hidden" name="action" value="approve">
                <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                <button type="submit" class="approve-btn">Approve Payment</button>
            </form>
            {{else}}
            <form method="post" style="margin-top: 15px;">
                <input type="hidden" name="payment_id" value="{{.ID}}">
                <input type="hidden" name="action" value="disapprove">
                <input type="hidden" name="csrf_token" value="{{$.CSRFToken}}">
                <button type="submit" class="disapprove-btn">Disapprove Payment</button>
            </form>
            <!-- DEBUG: Payment #{{.ID}} is APPROVED, showing disapprove button -->
            {{end}}
        </div>
        {{end}}

        <div style="margin-top: 30px; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
            <h3>System Information</h3>
            <p><strong>Security Level:</strong> Enhanced</p>
            <p><strong>Legacy System:</strong> <a href="/finance">Access legacy finance portal</a></p>
            <p><strong>Features:</strong> CSRF Protection, Session Validation, Audit Logging</p>
        </div>
    </div>
</body>
</html>`

	data := struct {
		Username  string
		Payments  []Payment
		CSRFToken string
	}{
		Username:  session.Values["username"].(string),
		Payments:  payments,
		CSRFToken: csrfToken,
	}

	tmpl := template.Must(template.New("payments").Parse(tmplStr))
	tmpl.Execute(w, data)
}

// Dashboard handler
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	authenticated, ok := session.Values["authenticated"].(bool)
	if !ok || !authenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Safely extract username with proper type checking
	var username string
	if usernameVal, exists := session.Values["username"]; exists {
		if usernameStr, ok := usernameVal.(string); ok {
			username = usernameStr
		} else {
			username = "User"
			fmt.Printf("DEBUG: Username exists but wrong type: %T\n", usernameVal)
		}
	} else {
		username = "User"
		fmt.Printf("DEBUG: Username not found in session\n")
	}
	
	fmt.Printf("DEBUG: Dashboard username: %s\n", username)

	// Use a simple template without format specifiers to avoid issues
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Dashboard - Globomantics CRM</title>
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
        .user-info { color: #666; }
        .container { max-width: 1200px; margin: 0 auto; padding: 0 20px; }
        .welcome-section {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 40px;
            border-radius: 12px;
            margin-bottom: 30px;
            text-align: center;
        }
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 30px;
        }
        .stat-card {
            background: white;
            padding: 25px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            text-align: center;
        }
        .stat-number {
            font-size: 36px;
            font-weight: bold;
            color: #333;
            margin-bottom: 10px;
        }
        .stat-label {
            color: #666;
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 1px;
        }
        .quick-actions {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }
        .action-card {
            background: white;
            padding: 25px;
            border-radius: 8px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .action-card h3 {
            margin-top: 0;
            color: #333;
            display: flex;
            align-items: center;
        }
        .action-card .icon {
            font-size: 24px;
            margin-right: 10px;
        }
        .action-btn {
            background: #007bff;
            color: white;
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            text-decoration: none;
            display: inline-block;
            margin-top: 15px;
            cursor: pointer;
        }
        .action-btn:hover { background: #0056b3; }
        .nav a { color: #007bff; text-decoration: none; margin-right: 20px; }
    </style>
</head>
<body>
    <div class="header">
        <div class="header-content">
            <div class="logo">Globomantics CRM</div>
            <div>
                <span class="user-info">Welcome, ` + username + `</span>
                <a href="/logout" style="margin-left: 20px; color: #007bff; text-decoration: none;">Logout</a>
            </div>
        </div>
    </div>

    <div class="container">
        <div class="welcome-section">
            <h1>Welcome back, ` + username + `!</h1>
            <p>Here's what's happening with your customer relationships today.</p>
        </div>
        
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-number">847</div>
                <div class="stat-label">Active Customers</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">12</div>
                <div class="stat-label">Pending Approvals</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">$45,230</div>
                <div class="stat-label">Monthly Revenue</div>
            </div>
            <div class="stat-card">
                <div class="stat-number">96%</div>
                <div class="stat-label">Customer Satisfaction</div>
            </div>
        </div>

        <div class="quick-actions">
            <div class="action-card">
                <h3><span class="icon">PAY</span>Payment Approvals</h3>
                <p>Review and approve pending vendor payments and customer refunds.</p>
                <a href="/approvals" class="action-btn">Modern System</a>
                <a href="/finance" class="action-btn" style="background: #6c757d;">Legacy System</a>
            </div>
            
            <div class="action-card">
                <h3><span class="icon">FEED</span>Customer Feedback</h3>
                <p>Collect and manage customer feedback through our feedback systems.</p>
                <a href="/feedback" class="action-btn" style="background: #dc3545;">Legacy Feedback Form</a>
                <a href="/feedback-secure" class="action-btn" style="background: #28a745;">Modern Feedback Form</a>
            </div>
            
            <div class="action-card">
                <h3><span class="icon">RPT</span>Reports & Analytics</h3>
                <p>Generate reports and analyze customer data to drive business decisions.</p>
                <a href="#" class="action-btn">View Reports</a>
            </div>
            
            <div class="action-card">
                <h3><span class="icon">CRM</span>Customer Management</h3>
                <p>Access customer profiles, interaction history, and relationship data.</p>
                <a href="#" class="action-btn">Manage Customers</a>
            </div>
        </div>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	// Ensure we have a valid username
	if username == "" {
		username = "User"
		fmt.Printf("DEBUG: Empty username, using default\n")
	}
	
	fmt.Printf("DEBUG: About to render template with username: '%s'\n", username)
	fmt.Fprint(w, tmpl)
}

// Logout handler
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session-name")
	session.Values["authenticated"] = false
	delete(session.Values, "username")
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
