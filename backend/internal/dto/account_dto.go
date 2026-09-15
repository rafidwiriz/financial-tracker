package dto

type CreateAccountRequest struct {
	Name		string	`json:"name" validate:"required,max=100"`
	Type		string	`json:"type" validate:"required,oneof=checking savings credit_card cash investment"`
	Currency	string	`json:"currency" validate:"required,len=3,uppercase"`
	Color		*string `json:"color" validate:"omitempty,hexcolor"`
}

type UpdateAccountRequest struct {
	Name		*string `json:"name" validate:"omitempty,max=100"`
	Type		*string `json:"type" validate:"omitempty,oneof=checking savings credit_card cash investment"`
	Color		*string `json:"color" validate:"omitempty,hexcolor"`
}
