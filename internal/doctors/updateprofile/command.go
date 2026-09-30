package updateprofile

type Command struct {
	Bio                  *string  `json:"bio"`
	ConsultationFee      *string  `json:"consultation_fee"      validate:"omitempty,decimal_positive"`
	Currency             *string  `json:"currency"              validate:"omitempty,len=3,uppercase"`
	ClinicName           *string  `json:"clinic_name"           validate:"omitempty,max=200"`
	ClinicAddress        *string  `json:"clinic_address"        validate:"omitempty,max=500"`
	MedicalLicenseNumber *string  `json:"medical_license_number" validate:"omitempty,max=100"`
	SpecialtyIDs         *[]int64 `json:"specialty_ids"         validate:"omitempty,dive,gt=0"`
}