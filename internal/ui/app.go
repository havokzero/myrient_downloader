// internal/ui/app.go
package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"awesomeProject1/internal/domain"
	"awesomeProject1/internal/download"
	"awesomeProject1/internal/scraper"
	"awesomeProject1/internal/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	//"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// selectableEntry wraps a FileEntry with a checkbox state.
type selectableEntry struct {
	Item     domain.FileEntry
	Selected bool
}

// startTitleMarquee runs a simple "marquee" effect in the window title.
// It rotates the given text forever until the app exits.
func startTitleMarquee(w fyne.Window, text string) {
	go func() {
		marquee := text + "   "

		for {
			w.SetTitle(marquee)

			if len(marquee) > 0 {
				marquee = marquee[1:] + marquee[:1]
			}
			time.Sleep(120 * time.Millisecond)
		}
	}()
}

func Run() {
	// Use a fixed app ID so Fyne prefs stop complaining.
	a := app.NewWithID("myrient-downloader")

	// Use embedded icon from icon.go
	a.SetIcon(AppIcon)

	baseTitle := "Myrient Download Manager 2025 - Merry Christmas!!"
	w := a.NewWindow(baseTitle)

	// Also set icon per-window (belt and suspenders)
	w.SetIcon(AppIcon)

	w.Resize(fyne.NewSize(800, 550))

	// Start scrolling the title bar text
	startTitleMarquee(w, baseTitle)

	httpIdx := scraper.NewHTTPIndex()

	// first loaded URL becomes the "root" for system detection
	rootURL := ""

	// base local directory where downloads will be saved
	baseDownloadDir := ""

	// default concurrency for bulk downloads
	maxConcurrent := 4

	// ---------- LOG CONSOLE ----------
	logOutput := widget.NewMultiLineEntry()
	logOutput.SetPlaceHolder("Download log…")
	logOutput.Wrapping = fyne.TextWrapWord
	logOutput.SetMinRowsVisible(8)
	logOutput.Disable()

	const maxLogChars = 20000

	appendLogUnsafe := func(msg string) {
		// This function assumes it's already running on the UI thread.
		line := msg + "\n"
		newText := logOutput.Text + line
		if len(newText) > maxLogChars {
			// keep last maxLogChars characters, cut at a newline boundary if possible
			newText = newText[len(newText)-maxLogChars:]
			if idx := strings.Index(newText, "\n"); idx != -1 {
				newText = newText[idx+1:]
			}
		}
		logOutput.SetText(newText)
	}

	console := download.NewConsole(func(msg string) {
		// Simple, no driver tricks – if this ever misbehaves,
		// we can later add a log channel + UI pump.
		appendLogUnsafe(msg)
	})

	dlMgr := download.NewManager(console)

	// ---------- TOP BAR ----------
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Enter index URL (e.g. https://myrient.erista.me/files/)")
	// Example default (SNES No-Intro):
	// urlEntry.SetText("https://myrient.erista.me/files/No-Intro/Nintendo%20-%20Super%20Nintendo%20Entertainment%20System/")

	loadBtn := widget.NewButton("Load", nil)

	header := container.NewBorder(
		nil,
		nil,
		widget.NewLabel("  HTTP Index Browser"),
		loadBtn,
		urlEntry,
	)

	// ---------- STATUS + PROGRESS ----------
	statusLabel := widget.NewLabel("Ready.")
	progressBar := widget.NewProgressBar()
	progressBar.Hide()

	// ---------- LEFT: DIRECTORY LIST + SEARCH ----------

	// allEntries = full set from current page
	// filteredIdx = indexes into allEntries that currently match the search term
	allEntries := []selectableEntry{}
	filteredIdx := []int{}
	selectedListIndex := -1 // index into filteredIdx

	// label for number of selected files
	selectedCountLabel := widget.NewLabel("Selected files: 0")

	var updateSelectedCount func()
	var applyFilter func(term string)

	// Search bar
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Search/filter files (e.g. 'Crash Bandicoot')")

	list := widget.NewList(
		func() int { return len(filteredIdx) },
		func() fyne.CanvasObject {
			// row = checkbox + label
			return container.NewHBox(
				widget.NewCheck("", nil),
				widget.NewLabel(""),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(filteredIdx) {
				return
			}
			row := o.(*fyne.Container)
			chk := row.Objects[0].(*widget.Check)
			lbl := row.Objects[1].(*widget.Label)

			globalIdx := filteredIdx[i]
			e := &allEntries[globalIdx]

			// Avoid firing OnChanged while we sync state
			chk.OnChanged = nil

			lbl.SetText(e.Item.Name)
			chk.SetChecked(e.Selected)

			iCopy := i
			chk.OnChanged = func(b bool) {
				globalIdx := filteredIdx[iCopy]
				allEntries[globalIdx].Selected = b
				if updateSelectedCount != nil {
					updateSelectedCount()
				}
			}
		},
	)

	// Track which item is selected (for single-file actions)
	list.OnSelected = func(id widget.ListItemID) {
		selectedListIndex = int(id)
	}
	list.OnUnselected = func(id widget.ListItemID) {
		if selectedListIndex == int(id) {
			selectedListIndex = -1
		}
	}

	// Count selected files across allEntries
	updateSelectedCount = func() {
		count := 0
		for _, e := range allEntries {
			if e.Selected && !e.Item.IsDir {
				count++
			}
		}
		selectedCountLabel.SetText(fmt.Sprintf("Selected files: %d", count))
	}

	applyFilter = func(term string) {
		term = strings.ToLower(strings.TrimSpace(term))
		filteredIdx = filteredIdx[:0]

		for i, e := range allEntries {
			if term == "" || strings.Contains(strings.ToLower(e.Item.Name), term) {
				filteredIdx = append(filteredIdx, i)
			}
		}

		selectedListIndex = -1
		list.Refresh()
		updateSelectedCount()

		if len(allEntries) > 0 {
			statusLabel.SetText(
				fmt.Sprintf("Showing %d of %d entries", len(filteredIdx), len(allEntries)),
			)
		}
	}

	searchEntry.OnChanged = func(s string) {
		applyFilter(s)
	}

	loadIndex := func() {
		u := urlEntry.Text
		if u == "" {
			dialog.ShowInformation("Info", "Please enter a URL first.", w)
			return
		}

		if rootURL == "" {
			// First URL loaded becomes the root for system detection.
			rootURL = u
			console.Log(fmt.Sprintf("Root URL set to: %s", rootURL))
		}

		statusLabel.SetText("Loading: " + u)

		// Synchronous request
		res, err := httpIdx.List(u)
		if err != nil {
			dialog.ShowError(err, w)
			statusLabel.SetText("Error: " + err.Error())
			return
		}

		allEntries = make([]selectableEntry, len(res))
		for i, fe := range res {
			allEntries[i] = selectableEntry{Item: fe, Selected: false}
		}
		// initial filtered view: everything
		filteredIdx = make([]int, len(allEntries))
		for i := range filteredIdx {
			filteredIdx[i] = i
		}

		selectedListIndex = -1
		list.Refresh()
		updateSelectedCount()

		statusLabel.SetText(
			fmt.Sprintf("Loaded %d entries from %s", len(allEntries), u),
		)
		console.Log(fmt.Sprintf("Page has %d entries (files + dirs).", len(allEntries)))
	}

	loadBtn.OnTapped = loadIndex

	// ---------- ACTION BUTTONS ----------

	// Navigate remote directory (Myrient side)
	openRemoteDirBtn := widget.NewButton("Open remote directory", func() {
		if selectedListIndex < 0 || selectedListIndex >= len(filteredIdx) {
			dialog.ShowInformation("Info", "Select a directory entry first.", w)
			return
		}
		globalIdx := filteredIdx[selectedListIndex]
		e := allEntries[globalIdx]

		if !e.Item.IsDir {
			dialog.ShowInformation("Info", "Selected item is not a directory.", w)
			return
		}
		urlEntry.SetText(e.Item.URL)
		loadIndex()
	})

	// Choose local download directory once
	setDownloadDirBtn := widget.NewButton("Set download folder…", func() {
		fd := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			baseDownloadDir = uri.Path()
			statusLabel.SetText("Download folder set to: " + baseDownloadDir)
			console.Log("Download folder set to: " + baseDownloadDir)
		}, w)
		fd.Show()
	})

	// Concurrency controls
	concurrencyLabel := widget.NewLabel(fmt.Sprintf("Concurrent downloads: %d", maxConcurrent))
	concurrencySlider := widget.NewSlider(1, 100)
	concurrencySlider.Step = 1
	concurrencySlider.SetValue(float64(maxConcurrent))
	concurrencySlider.OnChanged = func(v float64) {
		maxConcurrent = int(v)
		concurrencyLabel.SetText(fmt.Sprintf("Concurrent downloads: %d", maxConcurrent))
	}

	// Single-file download (uses baseDownloadDir) with byte progress + ETA
	downloadBtn := widget.NewButton("Download file…", func() {
		if baseDownloadDir == "" {
			dialog.ShowInformation("Info", "Set a download folder first.", w)
			return
		}
		if selectedListIndex < 0 || selectedListIndex >= len(filteredIdx) {
			dialog.ShowInformation("Info", "Select a file entry first.", w)
			return
		}

		globalIdx := filteredIdx[selectedListIndex]
		e := allEntries[globalIdx]

		if e.Item.IsDir {
			dialog.ShowInformation("Info", "Selected item is a directory.", w)
			return
		}

		baseTargetDir := baseDownloadDir

		// Figure out the system/console folder from URLs.
		systemName := util.GuessSystemFromURL(rootURL, e.Item.URL)
		targetDir := baseTargetDir
		if systemName != "" && systemName != "Unknown" {
			targetDir = filepath.Join(baseTargetDir, systemName)
			console.Log(fmt.Sprintf("Detected system: %s (target: %s)", systemName, targetDir))
		} else {
			console.Log("System could not be determined. Using base target directory.")
		}

		progressBar.SetValue(0)
		progressBar.Show()
		statusLabel.SetText("Starting download...")

		start := time.Now()

		if err := dlMgr.DownloadFile(e.Item.URL, targetDir, func(p download.Progress) {
			if p.Err != nil {
				statusLabel.SetText("Error: " + p.Err.Error())
				progressBar.Hide()
				return
			}
			if p.BytesTotal > 0 {
				ratio := float64(p.BytesDone) / float64(p.BytesTotal)
				if ratio < 0 {
					ratio = 0
				}
				if ratio > 1 {
					ratio = 1
				}
				progressBar.SetValue(ratio)
				eta := util.CalculateETA(p.BytesDone, p.BytesTotal, start)
				statusLabel.SetText(
					fmt.Sprintf(
						"%s / %s (ETA %s)",
						util.FormatBytes(p.BytesDone, 2),
						util.FormatBytes(p.BytesTotal, 2),
						eta,
					),
				)
			}
			if p.Done {
				progressBar.SetValue(1)
				statusLabel.SetText("Download complete.")
			}
		}); err != nil {
			statusLabel.SetText("Error: " + err.Error())
		}
	})

	// Select all / clear buttons
	selectAllBtn := widget.NewButton("Select all", func() {
		for i := range allEntries {
			allEntries[i].Selected = true
		}
		list.Refresh()
		updateSelectedCount()
	})

	clearSelectionBtn := widget.NewButton("Clear selection", func() {
		for i := range allEntries {
			allEntries[i].Selected = false
		}
		selectedListIndex = -1
		list.Refresh()
		updateSelectedCount()
	})

	// Bulk download of all checked files – WITH semaphores (concurrency + retry)
	downloadSelectedBtn := widget.NewButton("Download selected…", func() {
		if baseDownloadDir == "" {
			dialog.ShowInformation("Info", "Set a download folder first.", w)
			return
		}

		var toDownload []domain.FileEntry
		for _, e := range allEntries {
			if e.Selected && !e.Item.IsDir {
				toDownload = append(toDownload, e.Item)
			}
		}

		if len(toDownload) == 0 {
			dialog.ShowInformation("Info", "No files selected.", w)
			return
		}

		baseTargetDir := baseDownloadDir
		total := len(toDownload)

		type downloadResult struct {
			name string
			err  error
		}

		// Semaphore to limit concurrent downloads
		sem := make(chan struct{}, maxConcurrent)

		// Channel to receive completion events
		doneCh := make(chan downloadResult)

		progressBar.Show()
		progressBar.SetValue(0)

		console.Log(fmt.Sprintf("Starting bulk download of %d files with concurrency %d", total, maxConcurrent))

		// Kick off all jobs (goroutines are limited by sem)
		for _, f := range toDownload {
			f := f // capture loop variable

			systemName := util.GuessSystemFromURL(rootURL, f.URL)
			targetDir := baseTargetDir
			if systemName != "" && systemName != "Unknown" {
				targetDir = filepath.Join(baseTargetDir, systemName)
			}

			go func(name, url, td string) {
				sem <- struct{}{} // acquire slot
				err := dlMgr.DownloadFileWithRetry(url, td, nil, 3)
				<-sem // release slot
				doneCh <- downloadResult{name: name, err: err}
			}(f.Name, f.URL, targetDir)
		}

		// Collect results and update queue progress in the main goroutine
		completed := 0
		for i := 0; i < total; i++ {
			res := <-doneCh
			completed++
			ratio := float64(completed) / float64(total)
			if ratio < 0 {
				ratio = 0
			}
			if ratio > 1 {
				ratio = 1
			}
			progressBar.SetValue(ratio)

			initial := ""
			if len(res.name) > 0 {
				initial = strings.ToUpper(string(res.name[0]))
			}

			if res.err != nil {
				statusLabel.SetText(
					fmt.Sprintf(
						"Queue: %d / %d (%.1f%%, @ %s) – ERROR %s: %v",
						completed, total, ratio*100.0, initial, res.name, res.err,
					),
				)
				console.LogError(fmt.Sprintf("Error downloading %s: %v", res.name, res.err))
			} else {
				statusLabel.SetText(
					fmt.Sprintf(
						"Queue: %d / %d (%.1f%%, @ %s) – finished %s",
						completed, total, ratio*100.0, initial, res.name,
					),
				)
			}
		}

		statusLabel.SetText("All selected downloads completed.")
		console.Log("All selected downloads completed.")
	})

	// ---------- LEFT SIDE (search + list) ----------
	leftSide := container.NewBorder(
		searchEntry, // top
		nil,         // bottom
		nil,
		nil,
		list,
	)

	// ---------- RIGHT PANEL ----------
	rightSide := container.NewVBox(
		widget.NewLabelWithStyle("Actions & Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		concurrencyLabel,
		concurrencySlider,
		openRemoteDirBtn,
		setDownloadDirBtn,
		downloadBtn,
		downloadSelectedBtn,
		container.NewHBox(selectAllBtn, clearSelectionBtn),
		selectedCountLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Progress", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		progressBar,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Status", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		statusLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		logOutput,
	)

	// Split view: left = list, right = actions/log
	mainSplit := container.NewHSplit(leftSide, rightSide)
	mainSplit.SetOffset(0.6) // 60% left, 40% right

	content := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		mainSplit,
	)

	w.SetContent(content)
	w.ShowAndRun()
}
