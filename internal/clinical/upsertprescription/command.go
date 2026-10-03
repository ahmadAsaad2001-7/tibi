package upsertprescription

type MedicationInput struct {
	Name         string  `json:"medication_name" validate:"required,min=1,max=200"`
	Dosage       string  `json:"dosage"          validate:"required,min=1,max=100"`
	Frequency    string  `json:"frequency"       validate:"required,min=1,max=100"`
	DurationDays int     `json:"duration_days"   validate:"required,min=1,max=365"`
	Notes        *string `json:"notes"           validate:"omitempty,max=500"`
}

type Command struct {
	Medications []MedicationInput `json:"medications" validate:"required,min=1,dive"`
}