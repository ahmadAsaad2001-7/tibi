package ws

// Event is the server → client envelope. Payload is JSON-encoded by the writer.
type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

// ClientMessage is a client → server message.
type ClientMessage struct {
	Action          string `json:"action"`
	ConsultationID  int64  `json:"consultation_id,omitempty"`
	ClinicSessionID int64  `json:"clinic_session_id,omitempty"`
	Signal          any    `json:"signal,omitempty"`
}

const (
	ActionJoinConsultation  = "join_consultation"
	ActionLeaveConsultation = "leave_consultation"
	ActionJoinQueue         = "join_queue"
	ActionLeaveQueue        = "leave_queue"
	ActionWebRTCSignal      = "webrtc_signal"
	ActionPing              = "ping"
)
