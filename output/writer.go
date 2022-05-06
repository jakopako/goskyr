package output

import "sync"

type Writer interface {
	// if a writer encounters a fatal error it should call log.Fatalf
	// to prevent the crawler from uselessly continuing to run.
	Write(itemsList chan map[string]interface{}, wg *sync.WaitGroup)
}
