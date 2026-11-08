package repository

import (
	"app/internal/app/ds"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *Repository) GetTelescopes(ctx context.Context, minLatitude, maxLatitude float64) ([]ds.Telescope, error) {
	telescopes := []ds.Telescope{}
	err := r.db.WithContext(ctx).
		Where("status = ? AND latitude >= ? AND latitude <= ?", ds.TelescopePublished, minLatitude, maxLatitude).
		Order("id asc").Find(&telescopes).Error
	return telescopes, err
}

func (r *Repository) GetTelescopeDraft(ctx context.Context, userID uint) (ds.Telescope, error) {
	var telescope ds.Telescope
	err := r.db.WithContext(ctx).Where("creator_id = ? AND status = ?", userID, ds.TelescopeDraft).First(&telescope).Error
	return telescope, err
}
func (r *Repository) GetTelescopeFeedItem(ctx context.Context, currentID uint, next bool) (ds.Telescope, error) {
	var telescope ds.Telescope
	query := r.db.WithContext(ctx).Where("status = ?", ds.TelescopePublished)
	if currentID == 0 {
		return telescopeFirst(query, &telescope)
	}
	if !next {
		err := query.Where("id = ?", currentID).First(&telescope).Error
		return telescope, err
	}
	var current ds.Telescope
	if err := r.db.WithContext(ctx).Where("id = ? AND status = ?", currentID, ds.TelescopePublished).First(&current).Error; err != nil {
		return telescope, err
	}
	err := query.Where("id > ?", currentID).Order("id asc").First(&telescope).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return telescopeFirst(r.db.WithContext(ctx).Where("status = ?", ds.TelescopePublished), &telescope)
	}
	return telescope, err
}

func telescopeFirst(query *gorm.DB, telescope *ds.Telescope) (ds.Telescope, error) {
	err := query.Order("id asc").First(telescope).Error
	return *telescope, err
}

func (r *Repository) CreateDraftTelescope(ctx context.Context, userID uint, title string, image, video *multipart.FileHeader) (ds.Telescope, error) {
	var telescope ds.Telescope
	var uploaded []string
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user ds.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return fmt.Errorf("текущий пользователь отсутствует, выполните миграцию: %w", err)
		}
		var count int64
		if err := tx.Model(&ds.Telescope{}).Where("creator_id = ? AND status = ?", userID, ds.TelescopeDraft).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("у пользователя уже есть черновик: %w", ErrConflict)
		}
		telescope = ds.Telescope{Title: title, Status: ds.TelescopeDraft, CreatorID: userID, CreationTime: time.Now()}
		if err := tx.Create(&telescope).Error; err != nil {
			return err
		}
		for _, media := range []struct {
			kind string
			file *multipart.FileHeader
			name *string
		}{{"image", image, &telescope.ImageURL}, {"video", video, &telescope.VideoURL}} {
			if media.file == nil {
				continue
			}
			name, err := r.uploadTelescopeMedia(ctx, telescope.ID, media.kind, media.file)
			if err != nil {
				return err
			}
			uploaded = append(uploaded, name)
			*media.name = name
		}
		return tx.Model(&ds.Telescope{}).Where("id = ?", telescope.ID).
			Updates(map[string]interface{}{"image_url": telescope.ImageURL, "video_url": telescope.VideoURL}).Error
	})
	if err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		for _, name := range uploaded {
			if cleanupErr := r.removeTelescopeMedia(cleanupCtx, name); cleanupErr != nil {
				err = errors.Join(err, cleanupErr)
			}
		}
	}
	return telescope, err
}

func ownedTelescope(tx *gorm.DB, id, userID uint) (ds.Telescope, error) {
	var telescope ds.Telescope
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status <> ?", id, ds.TelescopeDeleted).First(&telescope).Error
	if err != nil {
		return telescope, err
	}
	if telescope.CreatorID != userID {
		return telescope, ErrForbidden
	}
	return telescope, nil
}

func (r *Repository) PublishTelescope(ctx context.Context, id, userID uint, description string, latitude, longitude float64) (ds.Telescope, error) {
	var telescope ds.Telescope
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		telescope, err = ownedTelescope(tx, id, userID)
		if err != nil {
			return err
		}
		if telescope.Status != ds.TelescopeDraft {
			return fmt.Errorf("публиковать можно только черновик: %w", ErrConflict)
		}
		now := time.Now()
		updates := map[string]interface{}{
			"description": description, "latitude": latitude, "longitude": longitude,
			"status": ds.TelescopePublished, "forming_time": now,
		}
		if err := tx.Model(&ds.Telescope{}).Where("id = ? AND status = ?", id, ds.TelescopeDraft).Updates(updates).Error; err != nil {
			return err
		}
		telescope.Description, telescope.Latitude, telescope.Longitude = description, latitude, longitude
		telescope.Status, telescope.FormingTime = ds.TelescopePublished, &now
		return nil
	})
	return telescope, err
}

func (r *Repository) DeleteTelescope(ctx context.Context, id, userID uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		telescope, err := ownedTelescope(tx, id, userID)
		if err != nil {
			return err
		}
		if telescope.Status != ds.TelescopeDraft && telescope.Status != ds.TelescopePublished {
			return ErrConflict
		}
		return tx.Model(&ds.Telescope{}).Where("id = ?", id).
			Update("status", ds.TelescopeDeleted).Error
	})
}
