package dto

type HelloWorldRequest struct {
	Name string `json:"name" validate:"required,min=3"`
}

type HelloWorldResponse struct {
	Message string `json:"message"`
}
