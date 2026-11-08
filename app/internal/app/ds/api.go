package ds

type PublishTelescopeRequest struct {
	Description string   `json:"description" binding:"required,max=255"`
	Latitude    *float64 `json:"latitude" binding:"required,gte=-90,lte=90"`
	Longitude   *float64 `json:"longitude" binding:"required,gte=-180,lte=180"`
}

type LikeTelescopeRequest struct {
	Like *int `json:"like" binding:"required,oneof=0 1"`
}

type RegisterUserRequest struct {
	Login    string `json:"login" binding:"required,max=64"`
	Password string `json:"password" binding:"required,min=6,max=100"`
}
type TelescopeSerializer struct {
	Telescope
	IsCreator  bool   `json:"is_creator"`
	LikesCount int64  `json:"likes_count"`
	ImageName  string `json:"image_name"`
	VideoName  string `json:"video_name"`
}

type FullTelescopeSerializer struct {
	TelescopeSerializer
	IsLiked int `json:"is_liked"`
}
