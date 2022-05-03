package output

import "sync"

type Writer interface {
	Write(itemsList chan map[string]interface{}, wg *sync.WaitGroup)
}
