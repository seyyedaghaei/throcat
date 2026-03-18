package logx

import (
	"encoding/json"
	"io"
	"log"
	"time"
)

type Logger struct {
	JSON bool
	Out  io.Writer
}

func (l Logger) Event(m map[string]interface{}) {
	if m == nil {
		m = map[string]interface{}{}
	}
	m["time"] = time.Now().Format(time.RFC3339)

	if l.JSON {
		out := l.Out
		if out == nil {
			out = log.Writer()
		}
		_ = json.NewEncoder(out).Encode(m)
		return
	}

	event, _ := m["event"].(string)
	switch event {
	case "listen":
		log.Printf("listening on %s", m["addr"])
	case "connection":
		if m["direction"] == "open" {
			log.Printf("connection from %s", m["remote"])
		} else {
			log.Printf("connection closed %s", m["remote"])
		}
	case "accept_error":
		log.Printf("accept: %v", m["error"])
	case "dial_error":
		log.Printf("dial %s: %v", m["upstream"], m["error"])
	default:
		log.Printf("%v", m)
	}
}

