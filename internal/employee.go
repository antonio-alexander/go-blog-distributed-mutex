package internal

// Employee models the information that describes an employee
type Employee struct {
	EmailAddress string `json:"email_address"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	Version      int    `json:"version"`
}
