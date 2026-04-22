package dto

type UpdateProfileRequest struct {
	FullName    *string `json:"full_name"    binding:"omitempty,min=2,max=100" example:"Nguyen Van A"`
	PhoneNumber *string `json:"phone_number" binding:"omitempty,min=7,max=15"  example:"+84912345678"`
}
