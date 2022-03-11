package output

type Writer interface {
	Write(itemsList chan []map[string]interface{})
}
