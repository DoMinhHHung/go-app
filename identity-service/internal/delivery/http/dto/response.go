package dto

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

func OK(message string, data any) Response {
	return Response{Success: true, Message: message, Data: data}
}

func Fail(message, code string) ErrorResponse {
	return ErrorResponse{Success: false, Message: message, Code: code}
}
