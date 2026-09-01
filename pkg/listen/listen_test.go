package listen

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

const testRawEvent = `{"version":"0","id":"event-id","detail-type":"order created","source":"orders","account":"123456789012","time":"2024-01-02T03:04:05Z","region":"eu-west-1","resources":["resource-arn"],"detail":{"orderId":"order-1","count":2}}`

func TestPrintMessageLogsFullEventByDefault(t *testing.T) {
	msg := newReceivedEvent(t)
	var logOut bytes.Buffer
	s := newOutputTestSQS(&logOut, false, false)

	var out bytes.Buffer
	if err := s.printMessage(&out, msg); err != nil {
		t.Fatalf("printMessage() error = %v", err)
	}

	if out.Len() != 0 {
		t.Fatalf("printMessage() stdout = %q, want empty", out.String())
	}

	entry := readLogEntry(t, logOut.String())
	if entry["msg"] != "received event" {
		t.Fatalf("log msg = %v, want received event", entry["msg"])
	}
	if entry["event"] != testRawEvent {
		t.Fatalf("log event = %v, want %s", entry["event"], testRawEvent)
	}
	if _, ok := entry["detail"]; ok {
		t.Fatalf("log detail was set in full-event mode: %v", entry["detail"])
	}
}

func TestPrintMessagePrettyPrintsFullEventWhenVerbose(t *testing.T) {
	msg := newReceivedEvent(t)
	var logOut bytes.Buffer
	s := newOutputTestSQS(&logOut, false, true)

	var out bytes.Buffer
	if err := s.printMessage(&out, msg); err != nil {
		t.Fatalf("printMessage() error = %v", err)
	}

	if got, want := out.String(), prettyJSON(t, testRawEvent); got != want {
		t.Fatalf("printMessage() stdout = %q, want %q", got, want)
	}
	if logOut.Len() != 0 {
		t.Fatalf("printMessage() log = %q, want empty", logOut.String())
	}
}

func TestPrintMessagePrintsRawTextWhenVerbose(t *testing.T) {
	var logOut bytes.Buffer
	s := newOutputTestSQS(&logOut, false, true)

	var out bytes.Buffer
	if err := s.printMessage(&out, receivedEvent{raw: json.RawMessage("hello from SNS")}); err != nil {
		t.Fatalf("printMessage() error = %v", err)
	}

	if got, want := out.String(), "hello from SNS\n"; got != want {
		t.Fatalf("printMessage() stdout = %q, want %q", got, want)
	}
}

func TestPrintMessagePrintsMalformedJSONUnchanged(t *testing.T) {
	var logOut bytes.Buffer
	s := newOutputTestSQS(&logOut, false, true)

	var out bytes.Buffer
	if err := s.printMessage(&out, receivedEvent{raw: json.RawMessage(`{"message":`)}); err != nil {
		t.Fatalf("printMessage() error = %v", err)
	}

	if got, want := out.String(), "{\"message\":\n"; got != want {
		t.Fatalf("printMessage() stdout = %q, want %q", got, want)
	}
}

func TestPrintMessageAddsDividerBetweenVerboseMessages(t *testing.T) {
	msg := newReceivedEvent(t)
	var logOut bytes.Buffer
	s := newOutputTestSQS(&logOut, false, true)

	var out bytes.Buffer
	if err := s.printMessage(&out, msg); err != nil {
		t.Fatalf("first printMessage() error = %v", err)
	}
	if err := s.printMessage(&out, msg); err != nil {
		t.Fatalf("second printMessage() error = %v", err)
	}

	pretty := prettyJSON(t, testRawEvent)
	if got, want := out.String(), pretty+"---\n"+pretty; got != want {
		t.Fatalf("printMessage() stdout = %q, want %q", got, want)
	}
	if logOut.Len() != 0 {
		t.Fatalf("printMessage() log = %q, want empty", logOut.String())
	}
}

func TestPrintMessageLogsOnlyDetail(t *testing.T) {
	msg := newReceivedEvent(t)
	var logOut bytes.Buffer
	s := newOutputTestSQS(&logOut, true, false)

	var out bytes.Buffer
	if err := s.printMessage(&out, msg); err != nil {
		t.Fatalf("printMessage() error = %v", err)
	}

	if out.Len() != 0 {
		t.Fatalf("printMessage() stdout = %q, want empty", out.String())
	}

	entry := readLogEntry(t, logOut.String())
	if entry["id"] != "event-id" {
		t.Fatalf("log id = %v, want event-id", entry["id"])
	}
	if entry["detail-type"] != "order created" {
		t.Fatalf("log detail-type = %v, want order created", entry["detail-type"])
	}
	if entry["detail"] != `{"orderId":"order-1","count":2}` {
		t.Fatalf("log detail = %v, want detail json", entry["detail"])
	}
	if _, ok := entry["event"]; ok {
		t.Fatalf("log event was set in only-detail mode: %v", entry["event"])
	}
}

func TestPrintMessagePrettyPrintsOnlyDetailWhenVerbose(t *testing.T) {
	msg := newReceivedEvent(t)
	var logOut bytes.Buffer
	s := newOutputTestSQS(&logOut, true, true)

	var out bytes.Buffer
	if err := s.printMessage(&out, msg); err != nil {
		t.Fatalf("printMessage() error = %v", err)
	}

	if got, want := out.String(), prettyJSON(t, `{"orderId":"order-1","count":2}`); got != want {
		t.Fatalf("printMessage() stdout = %q, want %q", got, want)
	}
	if logOut.Len() != 0 {
		t.Fatalf("printMessage() log = %q, want empty", logOut.String())
	}
}

func newOutputTestSQS(logOut *bytes.Buffer, onlyDetail bool, verbose bool) *SQS {
	return &SQS{
		log:        slog.New(slog.NewJSONHandler(logOut, nil)),
		onlyDetail: onlyDetail,
		verbose:    verbose,
	}
}

func newReceivedEvent(t *testing.T) receivedEvent {
	t.Helper()

	var event Event
	if err := json.Unmarshal([]byte(testRawEvent), &event); err != nil {
		t.Fatalf("failed to unmarshal test event: %v", err)
	}

	return receivedEvent{
		event: event,
		raw:   json.RawMessage(testRawEvent),
	}
}

func readLogEntry(t *testing.T, line string) map[string]any {
	t.Helper()

	var entry map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &entry); err != nil {
		t.Fatalf("failed to unmarshal log entry %q: %v", line, err)
	}

	return entry
}

func prettyJSON(t *testing.T, raw string) string {
	t.Helper()

	var out bytes.Buffer
	if err := json.Indent(&out, []byte(raw), "", "  "); err != nil {
		t.Fatalf("failed to indent json: %v", err)
	}
	out.WriteByte('\n')

	return out.String()
}
