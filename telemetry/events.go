package telemetry

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/newrelic/newrelic-telemetry-sdk-go/internal"
)

const eventTypeName string = "events"

// Event is a unique set of data that happened at a specific point in time
type Event struct {
	// Required Fields:
	//
	// EventType is the name of the event
	EventType string
	// Timestamp is when this event happened.  If Timestamp is not set, it
	// will be assigned to time.Now() in Harvester.RecordEvent.
	Timestamp time.Time

	// Recommended Fields:
	//
	// Attributes is a map of user specified data on this event.  The map
	// values can be any of bool, number, or string.
	Attributes map[string]interface{}
	// AttributesJSON is a json.RawMessage of attributes for this metric. It
	// will only be sent if Attributes is nil.
	AttributesJSON json.RawMessage
}

func (e *Event) writeJSON(buf *bytes.Buffer) {
	w := internal.JSONFieldsWriter{Buf: buf}
	buf.WriteByte('{')

	w.StringField("eventType", e.EventType)
	w.IntField("timestamp", e.Timestamp.UnixNano()/(1000*1000))

	internal.AddAttributes(&w, e.Attributes)

	buf.WriteByte('}')
}

// eventGroup represents a single batch of events to report to New Relic.
type eventGroup struct {
	Events []Event
}

// split will split the eventGroup into 2 equally sized batches.
// If the number of events in the original is 0 or 1 then nil is returned.
func (group *eventGroup) split() []splittablePayloadEntry {
	if len(group.Events) < 2 {
		return nil
	}

	half := len(group.Events) / 2
	b1 := *group
	b1.Events = group.Events[:half]
	b2 := *group
	b2.Events = group.Events[half:]

	return []splittablePayloadEntry{&b1, &b2}
}

func (group *eventGroup) writeJSON(buf *bytes.Buffer) {
	for idx, s := range group.Events {
		if idx > 0 {
			buf.WriteByte(',')
		}
		s.writeJSON(buf)
	}
}

// Type returns the type of data contained in this MapEntry.
func (group *eventGroup) DataTypeKey() string {
	return eventTypeName
}

// WriteBytes writes the json serialized bytes of the MapEntry to the buffer.
func (group *eventGroup) WriteDataEntry(buf *bytes.Buffer) *bytes.Buffer {
	group.writeJSON(buf)
	return buf
}

// NewEventGroup creates a new MapEntry representing a group of events in a batch.
func NewEventGroup(events []Event) MapEntry {
	return &eventGroup{Events: events}
}
