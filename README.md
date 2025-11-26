# 🎄 Myrient Downloader 2025
A modern graphical download manager for browsing and downloading files from the Myrient HTTP archive indexes — featuring filtering, bulk selection, concurrent downloads, automatic system categorization, progress indicators, ETA calculation, and a scrolling festive window title.

---

## ✨ Features

### ✅ Index Browser
- Load any Myrient directory URL
- Navigate folders like a file explorer
- Displays both files and directories

### ✅ Search & Filtering
- Instant text filtering
- Works on large lists
- Keeps checkbox state when filtering

### ✅ File Selection Controls
- Click-to-select individual files
- **Select All / Clear All**
- Displays selected file count

### ✅ Smart Downloading
- Choose a target folder once
- Downloads files into:
  ```
  /YourFolder/<Detected System>/
  ```
- Auto system detection based on URL path
- Avoids duplicates (skip existing)

### ✅ Single File Download Mode
- Byte-accurate progress bar
- Human readable sizes (MiB/GiB)
- ETA calculation
- Status updates

### ✅ Bulk Download Mode
- Configure concurrency (1–100 workers)
- Automatic retry handling
- Queue progress tracking
- Logs errors per file

### ✅ Integrated Log Console
- Timestamped events
- Truncated automatically to avoid memory bloat

### ✅ UI Enhancements
- Embedded icon using `go:embed`
- Title bar marquee animation
- 1200×800 default window layout

### ✅ Additional Notes 
- The max download limit will slide to 100 but the default is 4
- the 100 concurrent connections is for testing only
- please do not overload the website
- the program will download and unzip files automatically, however the folder will still end with ".zip" 

---

## 📥 Installation

### 🐧 Linux (Recommended)
```
git clone https://github.com/havokzero/myrient_downloader.git
cd myrient_downloader
go build -o myrient-downloader
./myrient-downloader
```

### 🪟 Windows ~ Untested
```
git clone https://github.com/havokzero/myrient_downloader.git
cd myrient_downloader
go build -o myrient-downloader.exe
myrient-downloader.exe
```

### 🍎 macOS ~ Untested
```
git clone https://github.com/havokzero/myrient_downloader.git
cd myrient_downloader
go build -o myrient-downloader
./myrient-downloader
```

---

## 🧱 Requirements
- Go 1.22+
- Fyne v2
- Internet connection capable of handling large downloads

---

## 📂 Project Structure
```
main.go
internal/
  ui/        → GUI, icon embed, window, list, download control
  scraper/   → HTTP index parsing
  download/  → download engine + concurrency + retry
  domain/    → file metadata model
  util/      → system detection, ETA, formatting helpers
```

---

## 🖼 Icon Handling

### ✅ Embedded in binary
Located at:
```
internal/ui/icon.png
```

Embedded via:
```go
//go:embed icon.png
var iconBytes []byte
```

Used in app:
```go
a.SetIcon(ui.AppIcon)
w.SetIcon(ui.AppIcon)
```

### ⚠ Linux Dock Behavior
GNOME sometimes does not show embedded icons in the dock.

Workaround (optional):
```
cp myrient-downloader /usr/local/bin/
cp internal/ui/icon.png /usr/share/icons/hicolor/256x256/apps/myrient-downloader.png
```

---

## 🚀 Roadmap
### ✅ Completed
- Search & checkbox persistence
- ETA & progress reporting
- Bulk download concurrency
- System-based auto-sorting
- Embedded icon
- Marquee title animation

### 🔜 Coming Soon ~ maybe
- Pause / Resume downloads
- Hash verification
- Save / restore selections
- Dark theme toggle
- Parallel directory walking
- Multi-mirror support

---

## 🐞 Known Issues
| Area | Status |
|------|--------|
| Ubuntu dock icon not showing | cosmetic, depends on desktop shell |
| Some Myrient indexes use unusual encodings | parser handles most cases |
| Large logs can push terminal output | auto truncation helps |

---

## 🖼 Screenshots

### Main Application Window
![Main UI](docs/image.png)

### Filtering, Selection, and System Detection
![Filtering and Selection](docs/image2.png)

### Bulk Download with Concurrency and Progress
![Bulk Download](docs/image3.png)


---

## 🤝 Contributions
PRs, feature suggestions, and bug reports are welcome.

---

## 📜 License
MIT — Free to modify, distribute, improve.

---

