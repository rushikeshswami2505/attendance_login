# Attendance Auto Sign-In

Automates daily attendance sign-in on Greythr using headless Chrome. Skips weekends and approved leave days automatically.

## Prerequisites

- **Go** 1.19+ — [golang.org](https://golang.org/dl/)
- **Google Chrome** installed on the system

---

## Setup (any machine, any location)

### 1. Place the exe

Copy `attendance.exe` to any folder on your machine. Example:

```
C:\Users\yourname\attendance_login\attendance.exe
```

> You can put it anywhere — just use that path consistently in Step 3.

---

### 2. Create `.env`

Create a `.env` file **in the same folder as the exe**:

```
username=KSCPL_your_number_
password=your_password
websiteurl="https://kanaka-software.greythr.com/"
HEADLESS=true
FORCE_SIGNIN=false
CHROME_PATH="C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
```

> Keep `.env` secure — it contains your credentials. Never commit it to git.

---

### 3. Create Scheduled Tasks (11:00 AM and 3:00 PM)

Open **PowerShell as Administrator** and run the block below.
**Change the first line to your actual exe path.**

Open **PowerShell as Administrator**, replace the path, and run:

```powershell
$Action = New-ScheduledTaskAction -Execute "C:\Users\yourname\attendance_login\attendance.exe"

Register-ScheduledTask -TaskName "AttendanceMorning"   -Action $Action -Trigger (New-ScheduledTaskTrigger -Daily -At 11am) -Force
Register-ScheduledTask -TaskName "AttendanceAfternoon" -Action $Action -Trigger (New-ScheduledTaskTrigger -Daily -At 3pm)  -Force
```

> Run this once. It creates both tasks. If you need to update the path later, run it again — `-Force` will overwrite.

---

### 4. Test immediately

```powershell
Start-ScheduledTask -TaskName "AttendanceSignInMorning"
```

Then check the log:

```
C:\attendance_info\attendance.log
```

And screenshots (on success):

```
C:\attendance_info\screenshots\
```

---

## Build from source

If you want to build the exe yourself:

```powershell
go build -o attendance.exe .\main.go
```

---

## Troubleshooting

| Symptom | Fix |
|---------|-----|
| No log entry, task exits with code 1 | Exe path in the task is wrong — re-run Step 3 with the correct `$exePath` |
| Login fails | Check credentials in `.env` |
| Chrome not found | Update `CHROME_PATH` in `.env` |
| Want to see the browser | Set `HEADLESS=false` in `.env` |
| Need to sign in on a weekend | Set `FORCE_SIGNIN=true` in `.env` |

To manage tasks: open **Task Scheduler** → Task Scheduler Library → look for `AttendanceSignIn*`.
