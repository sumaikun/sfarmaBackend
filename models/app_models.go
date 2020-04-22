package models

// CreateProduct is the request body of create product in prestashop input
type CreateProduct struct {
	Product   string `json:"product"`
	Reference string `json:"reference"`
	Price     string `json:"price"`
}
