package download

import "fmt"

// Logger is a function that accepts a log line (GUI will provide this).
type Logger func(string)

// Console is equivalent to the JS DownloadConsole, but in Go.
type Console struct {
	log Logger
}

func NewConsole(log Logger) *Console {
	if log == nil {
		log = func(string) {}
	}
	return &Console{log: log}
}

func (c *Console) Log(msg string) {
	c.log(msg)
}

func (c *Console) LogComplete() {
	c.log("Download complete!")
}

func (c *Console) LogCancelled() {
	c.log("Download cancelled!")
}

func (c *Console) LogError(msg string) {
	c.log("ERROR: " + msg)
}

func (c *Console) LogTotalSize(size string) {
	c.log("Total download size: " + size + ".")
}

func (c *Console) LogResuming(filename string, bytes int64) {
	c.log(fmt.Sprintf("Resuming download for %s from %d bytes.", filename, bytes))
}
