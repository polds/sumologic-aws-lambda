package main

import (
	// "encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/apex/go-apex"
	"github.com/apex/go-apex/logs"
	apexlog "github.com/apex/log"
	apexjson "github.com/apex/log/handlers/json"
	"github.com/bitly/go-simplejson"
	"github.com/jmcvetta/napping"
)

var (
	log       *apexlog.Entry
	collector = os.Getenv("SUMOLOGIC_COLLECTOR")
)

type LogMessage struct {
	ID        string      `json:"id"`
	Timestamp int64       `json:"timestamp"`
	Message   interface{} `json:"message"`
	RequestID string      `json:"requestID"`
	LogStream string      `json:"logStream"`
	LogGroup  string      `json:"logGroup"`
}

func init() {
	apexlog.SetHandler(apexjson.New(os.Stderr))
}

func main() {
	logs.HandleFunc(func(event *logs.Event, ctx *apex.Context) error {
		log = apexlog.WithFields(apexlog.Fields{
			"RequestID": ctx.RequestID,
			"collector": collector,
		})
		// Double check we have a collector to post to
		if collector == "" {
			err := errors.New("no collector variable available in the environment")
			log.WithError(err).
				Error("Please be sure to add SUMOLOGIC_COLLECTOR to your environment section in project.json")
			return err
		}

		msg := LogMessage{
			RequestID: ctx.RequestID,
			LogGroup:  event.LogGroup,
			LogStream: event.LogStream,
		}

		for _, logEvent := range event.LogEvents {
			msg.ID = logEvent.ID
			msg.Timestamp = logEvent.Timestamp
			msg.Message = detectJSON(logEvent.Message)
			emitSumoEvent(msg)
		}
		return nil
	})
}

// detectJSON takes a string, asserts it to a []byte and attempts
// to determine if it's valid JSON, if it is it returns an interface
// of the JSON Object. If it's not it returns the string as is.
func detectJSON(str string) interface{} {
	data, err := simplejson.NewJson([]byte(str))
	if err != nil {
		return str
	}

	return data.Interface()
}

func emitSumoEvent(msg LogMessage) {
	e := struct {
		Message string
	}{}

	resp, err := napping.Put(collector, &msg, nil, &e)
	if err != nil {
		log.WithError(err).Error("could not PUT to collector")
		return
	}

	if resp.Status() != http.StatusOK {
		log.WithFields(apexlog.Fields{
			"message": e.Message,
			"status":  resp.Status(),
		}).Error("unexpected response from SumoLogic")
		return
	}
}
