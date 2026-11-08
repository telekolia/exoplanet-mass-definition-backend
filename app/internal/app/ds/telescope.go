package ds

import "time"

const (
	TelescopeDraft     = "draft"
	TelescopePublished = "published"
	TelescopeDeleted   = "deleted"
)

type Telescope struct {
	ID           uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	Status       string     `gorm:"type:varchar(20);not null;check:chk_telescope_status,status IN ('draft','published','deleted')" json:"status"`
	Title        string     `gorm:"type:varchar(127); not null" json:"title"`
	Description  string     `gorm:"type:varchar(255)" json:"description"`
	Latitude     float64    `gorm:"type:float8; not null" json:"latitude"`
	Longitude    float64    `gorm:"type:float8; not null" json:"longitude"`
	ImageURL     string     `gorm:"type:varchar(255)" json:"image_url"`
	VideoURL     string     `gorm:"type:varchar(255)" json:"video_url"`
	CreatorID    uint       `gorm:"not null;uniqueIndex:idx_telescope_creator_draft,where:status = 'draft'" json:"creator_id"`
	CreationTime time.Time  `gorm:"type:timestamp;not null;autoCreateTime" json:"creation_time"`
	FormingTime  *time.Time `gorm:"type:timestamp" json:"forming_time"`

	Creator User `gorm:"foreignKey:CreatorID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"-"`
}
