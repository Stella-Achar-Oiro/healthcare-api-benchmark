package models

import (
	"fmt"
	"math/rand"
	"time"
)

// Patient represents a healthcare patient record with realistic medical data.
// This structure mirrors typical EHR (Electronic Health Record) systems.
type Patient struct {
	ID                 string    `json:"id"`
	MedicalRecordNumber string    `json:"medical_record_number"`
	FirstName          string    `json:"first_name"`
	LastName           string    `json:"last_name"`
	DateOfBirth        time.Time `json:"date_of_birth"`
	Gender             string    `json:"gender"`
	DiagnosisCodes     []string  `json:"diagnosis_codes"`
	Medications        []string  `json:"medications"`
	Allergies          []string  `json:"allergies"`
	LastVisitDate      time.Time `json:"last_visit_date"`
	PrimaryPhysician   string    `json:"primary_physician"`
	InsuranceProvider  string    `json:"insurance_provider"`
	BloodType          string    `json:"blood_type"`
}

// PatientResponse represents the API response structure for patient queries.
// This is what gets serialized and sent to API clients.
//
// Note: In a real healthcare system, this would include additional metadata
// such as FHIR compliance markers, audit trails, and consent flags.
type PatientResponse struct {
	Success   bool      `json:"success"`
	Patient   *Patient  `json:"patient,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	RequestID string    `json:"request_id"`
}

var (
	// Sample data pools for generating realistic patient records
	firstNames = []string{
		"James", "Mary", "John", "Patricia", "Robert", "Jennifer",
		"Michael", "Linda", "William", "Elizabeth", "David", "Barbara",
		"Richard", "Susan", "Joseph", "Jessica", "Thomas", "Sarah",
	}

	lastNames = []string{
		"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia",
		"Miller", "Davis", "Rodriguez", "Martinez", "Hernandez", "Lopez",
		"Wilson", "Anderson", "Thomas", "Taylor", "Moore", "Jackson",
	}

	// Common ICD-10 diagnosis codes (simplified)
	diagnosisCodes = []string{
		"E11.9",  // Type 2 diabetes mellitus without complications
		"I10",    // Essential (primary) hypertension
		"J44.9",  // COPD, unspecified
		"E78.5",  // Hyperlipidemia, unspecified
		"M54.5",  // Low back pain
		"F41.9",  // Anxiety disorder, unspecified
		"K21.9",  // GERD without esophagitis
		"J45.909", // Unspecified asthma
	}

	medications = []string{
		"Metformin 500mg", "Lisinopril 10mg", "Atorvastatin 20mg",
		"Omeprazole 20mg", "Albuterol inhaler", "Levothyroxine 75mcg",
		"Amlodipine 5mg", "Gabapentin 300mg", "Sertraline 50mg",
	}

	allergies = []string{
		"Penicillin", "Sulfa drugs", "Aspirin", "Iodine",
		"Latex", "Shellfish", "No known allergies",
	}

	physicians = []string{
		"Dr. Anderson", "Dr. Patel", "Dr. Chen", "Dr. Williams",
		"Dr. Johnson", "Dr. Rodriguez", "Dr. Kim", "Dr. Thompson",
	}

	insuranceProviders = []string{
		"Blue Cross Blue Shield", "UnitedHealthcare", "Aetna",
		"Cigna", "Humana", "Kaiser Permanente", "Medicare", "Medicaid",
	}

	bloodTypes = []string{
		"A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-",
	}
)

// GeneratePatient creates a realistic patient record with random but plausible data.
// This is used for testing and simulation purposes.
//
// In a production healthcare system, this would be replaced with actual database queries.
// The random data generation helps create realistic load patterns for benchmarking.
func GeneratePatient(id string) *Patient {
	rand.Seed(time.Now().UnixNano() + hashString(id))

	// Generate a realistic age (18-90 years old)
	yearsOld := rand.Intn(72) + 18
	dob := time.Now().AddDate(-yearsOld, -rand.Intn(12), -rand.Intn(28))

	// Generate 1-3 diagnosis codes
	diagnosisCount := rand.Intn(3) + 1
	selectedDiagnoses := make([]string, diagnosisCount)
	for i := 0; i < diagnosisCount; i++ {
		selectedDiagnoses[i] = diagnosisCodes[rand.Intn(len(diagnosisCodes))]
	}

	// Generate 0-4 medications
	medCount := rand.Intn(5)
	selectedMeds := make([]string, medCount)
	for i := 0; i < medCount; i++ {
		selectedMeds[i] = medications[rand.Intn(len(medications))]
	}

	// Generate 0-2 allergies
	allergyCount := rand.Intn(3)
	if allergyCount == 0 {
		allergyCount = 1 // Everyone gets at least "No known allergies"
	}
	selectedAllergies := make([]string, allergyCount)
	for i := 0; i < allergyCount; i++ {
		selectedAllergies[i] = allergies[rand.Intn(len(allergies))]
	}

	// Last visit within the past year
	lastVisit := time.Now().AddDate(0, -rand.Intn(12), -rand.Intn(28))

	gender := "Male"
	if rand.Intn(2) == 0 {
		gender = "Female"
	}

	return &Patient{
		ID:                 id,
		MedicalRecordNumber: fmt.Sprintf("MRN-%07d", rand.Intn(9999999)),
		FirstName:          firstNames[rand.Intn(len(firstNames))],
		LastName:           lastNames[rand.Intn(len(lastNames))],
		DateOfBirth:        dob,
		Gender:             gender,
		DiagnosisCodes:     selectedDiagnoses,
		Medications:        selectedMeds,
		Allergies:          selectedAllergies,
		LastVisitDate:      lastVisit,
		PrimaryPhysician:   physicians[rand.Intn(len(physicians))],
		InsuranceProvider:  insuranceProviders[rand.Intn(len(insuranceProviders))],
		BloodType:          bloodTypes[rand.Intn(len(bloodTypes))],
	}
}

// Validate performs basic validation on patient data.
// In a real healthcare system, this would be much more comprehensive
// and include checks for data integrity, consent, and authorization.
func (p *Patient) Validate() error {
	if p.ID == "" {
		return fmt.Errorf("patient ID is required")
	}
	if p.FirstName == "" || p.LastName == "" {
		return fmt.Errorf("patient name is required")
	}
	if p.DateOfBirth.After(time.Now()) {
		return fmt.Errorf("date of birth cannot be in the future")
	}
	return nil
}

// GetAge calculates the patient's current age in years.
func (p *Patient) GetAge() int {
	now := time.Now()
	age := now.Year() - p.DateOfBirth.Year()

	// Adjust if birthday hasn't occurred this year
	if now.Month() < p.DateOfBirth.Month() ||
		(now.Month() == p.DateOfBirth.Month() && now.Day() < p.DateOfBirth.Day()) {
		age--
	}

	return age
}

// hashString creates a simple hash of a string to use as a seed.
// This ensures the same patient ID generates the same random data consistently.
func hashString(s string) int64 {
	var h int64
	for _, c := range s {
		h = 31*h + int64(c)
	}
	return h
}

// NewPatientResponse creates a successful patient response.
func NewPatientResponse(patient *Patient, requestID string) *PatientResponse {
	return &PatientResponse{
		Success:   true,
		Patient:   patient,
		Timestamp: time.Now(),
		RequestID: requestID,
	}
}

// NewErrorResponse creates an error response for failed requests.
func NewErrorResponse(err error, requestID string) *PatientResponse {
	return &PatientResponse{
		Success:   false,
		Error:     err.Error(),
		Timestamp: time.Now(),
		RequestID: requestID,
	}
}
