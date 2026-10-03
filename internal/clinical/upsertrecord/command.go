package upsertrecord

type Command struct {
	Allergies          *string `json:"allergies"`
	CurrentMedications *string `json:"current_medications"`
	PastConditions     *string `json:"past_conditions"`
	DoctorNotes        *string `json:"doctor_notes"`
}
