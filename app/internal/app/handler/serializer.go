package handler

import "app/internal/app/ds"

func (h *Handler) serializeTelescope(telescope ds.Telescope, userID uint, count int64) ds.TelescopeSerializer {
	item := ds.TelescopeSerializer{
		Telescope: telescope, LikesCount: count,
		ImageName: telescope.ImageURL, VideoName: telescope.VideoURL,
		IsCreator: userID != 0 && telescope.CreatorID == userID,
	}
	item.ImageURL = h.Repository.MediaURL(telescope.ImageURL)
	item.VideoURL = h.Repository.MediaURL(telescope.VideoURL)
	return item
}
