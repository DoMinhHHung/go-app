package dto

import "github.com/DoMinhHHung/go-app/user-service/internal/usecase"

type Response struct {
	Success bool   `json:"success" example:"true"`
	Message string `json:"message" example:"ok"`
	Data    any    `json:"data,omitempty"`
}

type ErrorResponse struct {
	Success bool   `json:"success" example:"false"`
	Message string `json:"message" example:"error description"`
	Code    string `json:"code"    example:"ERROR_CODE"`
}

type ListUsersData struct {
	Users  []*usecase.ProfileOutput `json:"users"`
	Limit  int                      `json:"limit"  example:"20"`
	Offset int                      `json:"offset" example:"0"`
}

func OK(message string, data any) Response {
	return Response{Success: true, Message: message, Data: data}
}

func Fail(message, code string) ErrorResponse {
	return ErrorResponse{Success: false, Message: message, Code: code}
}
