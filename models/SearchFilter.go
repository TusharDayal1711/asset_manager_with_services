package models

type AssetFilter struct {
	IsSearchText bool
	SearchText   string
	Status       []string
	OwnedBy      []string
	Type         []string
	Limit        int
	Offset       int
}

type EmployeeFilter struct {
	IsSearchText bool
	SearchText   string
	Type         []string
	Role         []string
	AssetStatus  []string
	Limit        int
	Offset       int
}

//type AssetSearchFilter struct {
//	Brand    string `schema:"brand"`
//	Model    string `schema:"model"`
//	SerialNo string `schema:"serial_no"`
//	Type     string `schema:"type"`
//	OwnedBy  string `schema:"owned_by"`
//}

// models/filter.go
