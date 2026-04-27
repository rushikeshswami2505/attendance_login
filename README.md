# Attendance Auto Sign-In

This Go program automates attendance sign-in for Greythr or similar systems using Chrome automation. It runs headless by default and can be scheduled to execute daily.

## Prerequisites

- **Go**: Install Go from [golang.org](https://golang.org/dl/). Version 1.19+ recommended.
- **Chrome Browser**: Ensure Google Chrome is installed on the system.
- **Windows**: This setup is tailored for Windows (uses PowerShell for notifications and Task Scheduler).

## Setup Instructions

### 1. Clone or Download the Project

Copy the project files to a directory on your machine, e.g., `C:\attendance_login`.

### 2. Configure Environment Variables

Create a `.env` file in the project root with your credentials:

```
username=your_username
password=your_password
websiteurl=https://your-company.greythr.com
HEADLESS=true  # Set to false for visible browser (default: true)
FORCE_SIGNIN=false  # Set to true to force sign-in on weekends (default: false)
CHROME_PATH=  # Optional: Path to Chrome executable if not in default location
```

**Security Note**: Keep the `.env` file secure and do not commit it to version control.

### 3. Build the Executable

Open a PowerShell terminal in the project directory and run:

```powershell
go build -o attendance.exe .\main.go
```

This creates `attendance.exe` in the current directory.

### 4. Test the Program

Run the executable to verify it works:

```powershell
.\attendance.exe
```

Check the logs in `C:\attendance_info\attendance.log` for output.

### 5. Schedule Daily Execution

Use Windows Task Scheduler to run the program at specific times. Run these commands in PowerShell (as Administrator if needed):

#### Morning Sign-In (11:00 AM)
```powershell
schtasks /create /tn "AttendanceSignInMorning" /tr "C:\path\to\attendance.exe" /sc daily /st 11:00
```

#### Afternoon Sign-In (3:00 PM)
```powershell
schtasks /create /tn "AttendanceSignInAfternoon" /tr "C:\path\to\attendance.exe" /sc daily /st 15:00
```

Replace `C:\path\to\attendance.exe` with the full path to your `attendance.exe` file.

### 6. Manage Scheduled Tasks

- Open **Task Scheduler** (search in Windows Start menu).
- Tasks appear under **Task Scheduler Library**.
- To modify: Right-click a task > Properties.
- To delete: Right-click > Delete.

### Troubleshooting

- **Build Errors**: Ensure Go is installed and `go.mod` dependencies are downloaded (`go mod tidy`).
- **Sign-In Failures**: Check logs in `C:\attendance_info\attendance.log`. Verify `.env` credentials and website URL.
- **Chrome Issues**: If headless mode fails, set `HEADLESS=false` in `.env` to run with visible browser.
- **Task Scheduler**: Run PowerShell as Administrator when creating tasks. Ensure the executable path is correct.

### Logs and Screenshots

- Logs: `C:\attendance_info\attendance.log`
- Screenshots: `C:\attendance_info\screenshots\` (captured on sign-in)

### Notes

- The program skips weekends unless `FORCE_SIGNIN=true`.
- Notifications use Windows PowerShell popups.
- For non-Windows systems, adapt the scheduling and notification code accordingly.