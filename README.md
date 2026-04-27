# Attendance Auto Sign-In

Automates daily attendance sign-in on Greythr using headless Chrome. Skips weekends and approved leave days automatically.

## Prerequisites

- **Go** 1.19+ — [golang.org](https://golang.org/dl/)
- **Google Chrome** installed on the system

## Setup

### 1. Configure `.env`

Create a `.env` file in the project root:

```
username=your_username
password=your_password
websiteurl=https://your-company.greythr.com
HEADLESS=true
FORCE_SIGNIN=false
CHROME_PATH=C:\Program Files\Google\Chrome\Application\chrome.exe
```

> Keep `.env` secure — it contains your credentials.

### 2. Build

```powershell
go build -o attendance.exe .\main.go
```

### 3. Test

```powershell
.\attendance.exe
```

Logs are written to `C:\attendance_info\attendance.log`. Screenshots on successful sign-in are saved to `C:\attendance_info\screenshots\`.

## Scheduled Tasks

Two Windows Task Scheduler tasks run the sign-in automatically:

| Task | Time |
|------|------|
| AttendanceSignInMorning | 11:00 AM daily |
| AttendanceSignInAfternoon | 3:00 PM daily |

Both tasks invoke `C:\attendance_info\run_wrapper.bat` which launches the executable.

To manage tasks: open **Task Scheduler** → Task Scheduler Library.

## Troubleshooting

- **Sign-in fails** — check `C:\attendance_info\attendance.log` and verify `.env` credentials.
- **Chrome issues** — set `HEADLESS=false` in `.env` to watch the browser.
- **Weekend override** — set `FORCE_SIGNIN=true` in `.env`.
