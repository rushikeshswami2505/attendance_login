package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/joho/godotenv"
)

func sendNotification(title, message string) {
	// Run in background so it doesn't block the main program
	go func() {
		// Using a simpler PowerShell command for notification
		script := fmt.Sprintf(`$ws = New-Object -ComObject WScript.Shell; $ws.Popup("%s", 5, "%s", 64)`, message, title)
		cmd := exec.Command("powershell", "-Command", script)
		_ = cmd.Run()
	}()
}

func main() {
	// Root storage location on C: drive
	basePath := `C:\attendance_info`
	_ = os.MkdirAll(filepath.Join(basePath, "screenshots"), 0755)

	// 1. Setup Logging to the C: drive root location
	logPath := filepath.Join(basePath, "attendance.log")
	logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		mw := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(mw)
		os.Stdout = logFile
		os.Stderr = logFile
	}

	fmt.Printf("\n--- Run started at %s ---\n", time.Now().Format("2006-01-02 15:04:05"))

	// 2. Load .env file from the EXECUTABLE'S directory (makes it truly folder-independent)
	exePath, _ := os.Executable()
	exeDir := filepath.Dir(exePath)
	err = godotenv.Load(filepath.Join(exeDir, ".env"))
	if err != nil {
		log.Printf("Warning: Could not load .env from %s, trying local...", exeDir)
		_ = godotenv.Load() // Fallback to current dir
	}

	username := os.Getenv("username")
	password := os.Getenv("password")
	baseURL := os.Getenv("websiteurl")

	if username == "" || password == "" || baseURL == "" {
		log.Fatal("Missing credentials in .env file (username, password, websiteurl)")
	}

	// Remove any trailing slash to prevent double slashes in URL construction
	baseURL = strings.TrimSuffix(baseURL, "/")

	force := os.Getenv("FORCE_SIGNIN")

	// 2. Check if today is a weekend
	now := time.Now()
	if (now.Weekday() == time.Saturday || now.Weekday() == time.Sunday) && force != "true" {
		fmt.Printf("Today is %s. Skipping sign-in for the weekend. (Set FORCE_SIGNIN=true to bypass)\n", now.Weekday())
		return
	}

	headless := os.Getenv("HEADLESS") != "false" // Default to true (headless) for server/WSL
	chromePath := os.Getenv("CHROME_PATH")

	// 3. Setup browser context
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("window-size", "1920,1080"),
		// This flag prevents the "Chrome is being controlled by automated test software" header
		// which can sometimes interfere with headless rendering or triggers anti-bot scripts.
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		// Set a real-looking user agent so sites don't see "HeadlessChrome"
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
	)

	// If a custom chrome path is provided (e.g., from Windows), use it
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	// Set a timeout for the entire process
	ctx, cancel = context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	fmt.Println("Starting Greythr Auto Sign-In process...")

	var leaveStatus string
	todayStr := now.Format("02 Jan 2006") // Greythr format usually: 07 Mar 2026

	fmt.Println("Attempting to login...")
	err = chromedp.Run(ctx,
		// Navigate to login
		chromedp.Navigate(baseURL),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Page navigated. Waiting for login fields...")
			return nil
		}),
		chromedp.WaitVisible(`#username`, chromedp.ByQuery),
		chromedp.SendKeys(`#username`, username, chromedp.ByQuery),
		chromedp.SendKeys(`#password`, password, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Credentials entered. Clicking login...")
			return nil
		}),
		// Use JS to find and click the login button - more reliable
		chromedp.Evaluate(`
			(() => {
				const loginBtn = Array.from(document.querySelectorAll('button')).find(b => 
					b.innerText.includes('Login') || 
					b.type === 'submit' || 
					b.classList.contains('btn-primary')
				);
				if (loginBtn) {
					loginBtn.click();
					return "CLICKED";
				}
				return "NOT_FOUND";
			})()
		`, nil),
		
		// Wait for landing page or error
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Login click dispatched. Waiting for dashboard (up to 40s)...")
			
			// Create a sub-context with a shorter timeout for this specific wait
			waitCtx, cancel := context.WithTimeout(ctx, 40*time.Second)
			defer cancel()
			
			// Wait for either the home link OR a known error message
			// Greythr home link is usually in the sidebar or top nav
			err := chromedp.WaitVisible(`a[href*="home"], .error, .alert-danger, .toast-message`, chromedp.ByQuery).Do(waitCtx)
			if err != nil {
				fmt.Println("Login timeout! Taking diagnostic screenshot...")
				var buf []byte
				_ = chromedp.CaptureScreenshot(&buf).Do(ctx)
				debugPath := filepath.Join(basePath, "screenshots", "login_failed_debug.png")
				_ = os.WriteFile(debugPath, buf, 0644)
				fmt.Printf("DEBUG: Screenshot saved to %s\n", debugPath)
				return fmt.Errorf("could not reach dashboard: %v", err)
			}
			return nil
		}),
	)
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	fmt.Println("Logged in successfully. Navigating to Leave History...")

	err = chromedp.Run(ctx,
		// Step 4: Check Leave History
		chromedp.Navigate(baseURL+"/v3/portal/ess/leave/leave-workflow/history"),
		// Wait for either the history cards or a "No leave" message
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second), // Wait extra for Angular/Ajax to load data
		// Evaluate if today has an approved leave
		chromedp.Evaluate(fmt.Sprintf(`
			(() => {
				const cards = Array.from(document.querySelectorAll('div, tr, .leave-history-card, .leave-row'));
				const today = "%s";
				const onLeave = cards.some(c => {
					const text = c.innerText || "";
					// Look for approved leave that matches today's date
					return text.includes(today) && (text.includes("APPROVED") || text.includes("Approved"));
				});
				return onLeave ? "ON_LEAVE" : "NO_LEAVE";
			})()
		`, todayStr), &leaveStatus),
	)

	if err != nil {
		log.Printf("Leave check warning (continuing anyway): %v", err)
		leaveStatus = "NO_LEAVE"
	}

	fmt.Println("Leave check finished. Status:", leaveStatus)

	if leaveStatus == "ON_LEAVE" {
		fmt.Printf("Detected approved leave for today (%s). Skipping sign-in.\n", todayStr)
		sendNotification("Greythr Attendance", "Skipping: Approved leave detected.")
		return
	}

	fmt.Println("No leave detected. Proceeding to sign-in page...")

	// Step 5: Click Sign In on Home Page
	fmt.Println("Navigating to Home Dashboard...")
	err = chromedp.Run(ctx,
		chromedp.Navigate(baseURL+"/v3/portal/ess/home"),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Home page loaded. Waiting for attendance widget...")
			return nil
		}),
		chromedp.Sleep(10*time.Second), // Give plenty of time for dashboard widgets
		chromedp.ActionFunc(func(ctx context.Context) error {
			fmt.Println("Searching for sign-in button...")
			return nil
		}),
		
		chromedp.Evaluate(`
			(() => {
				function findButtonRecursive(root, searchTexts) {
					// 1. Check all buttons/links in the current root
					const candidates = Array.from(root.querySelectorAll('button, a, gt-button, [role="button"]'));
					for (const el of candidates) {
						const text = (el.innerText || el.textContent || "").trim().toUpperCase();
						for (const s of searchTexts) {
							if (text === s.toUpperCase()) return { element: el, text: text };
						}
					}

					// 2. Check shadow roots of all children
					const allChildren = root.querySelectorAll('*');
					for (const child of allChildren) {
						if (child.shadowRoot) {
							const found = findButtonRecursive(child.shadowRoot, searchTexts);
							if (found) return found;
						}
					}
					return null;
				}

				// First look for Sign In
				const signIn = findButtonRecursive(document, ["Sign In", "Mark Attendance"]);
				if (signIn) {
					signIn.element.click();
					return "CLICKED: " + signIn.text;
				}

				// Then look for Sign Out (already signed in)
				const signOut = findButtonRecursive(document, ["Sign Out", "MARK OUT"]);
				if (signOut) {
					return "ALREADY_SIGNED_IN";
				}

				// Debug: Collect all button texts seen to help
				const allBtns = Array.from(document.querySelectorAll('button, gt-button')).map(b => (b.innerText || b.textContent || "").trim());
				return "NOT_FOUND. Buttons: " + allBtns.join(", ");
			})()
		`, &leaveStatus),
	)

	if err != nil {
		log.Fatalf("Failed to execute sign-in logic: %v", err)
	}

	if strings.Contains(leaveStatus, "CLICKED") {
		fmt.Printf("Success: %s ✅\n", leaveStatus)
		sendNotification("Greythr Attendance", "Successfully signed in today's attendance! ✅")
		fmt.Println("Waiting for success banner to appear and then taking screenshot...")
		
		var buf []byte
		err = chromedp.Run(ctx,
			// Wait for the "Successfully Signed in" message, or at least wait up to 5 seconds
			chromedp.ActionFunc(func(ctx context.Context) error {
				// Try to wait for the banner with a shorter timeout so it doesn't fail the whole script if missing
				timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				
				// We wait for either the success toast, OR the "Sign Out" button
				// The banner in the screenshot contains "Successfully Signed in"
				_ = chromedp.WaitVisible(`//*[contains(text(), "Successfully Signed in")] | //button[contains(text(), "Sign Out")]`, chromedp.BySearch).Do(timeoutCtx)
				return nil // Ignore error if it times out, we'll take the screenshot anyway
			}),
			chromedp.Sleep(1*time.Second), // Final small wait to ensure UI is settled
			chromedp.CaptureScreenshot(&buf),
		)
		
		if err == nil && len(buf) > 0 {
			filename := fmt.Sprintf("signin_%s.png", time.Now().Format("2006-01-02_15-04-05"))
			fullPath := filepath.Join(basePath, "screenshots", filename)
			if writeErr := os.WriteFile(fullPath, buf, 0644); writeErr == nil {
				fmt.Printf("Screenshot successfully saved to %s\n", fullPath)
			} else {
				fmt.Printf("Warning: Could not save the screenshot file - %v\n", writeErr)
			}
		} else {
			fmt.Printf("Warning: Could not capture screenshot - %v\n", err)
		}
	} else if leaveStatus == "ALREADY_SIGNED_IN" {
		fmt.Println("Today's attendance is already signed. ✅")
		sendNotification("Greythr Attendance", "Attendance already signed for today! ✅")
	} else {
		fmt.Println("Debug Info:", leaveStatus)
		sendNotification("Greythr Attendance", "Failed to sign in today. Please check the logs. ❌")
		fmt.Printf("\nCould not find the button. Please check the logs at %s for more info.\n", logPath)
	}
}
