package dao

import (
	"context"

	"gorm.io/gorm"
)

// QueryCrowdTagsJob mirrors crowd_tags_job_mapper queryCrowdTagsJob.
func (d *DAO) QueryCrowdTagsJob(ctx context.Context, tx *gorm.DB, tagId, batchId string) (*CrowdTagsJob, error) {
	var j CrowdTagsJob
	err := tx.WithContext(ctx).
		Select("tag_type, tag_rule, stat_start_time, stat_end_time").
		Where("tag_id = ? AND batch_id = ?", tagId, batchId).
		Take(&j).Error
	if err != nil {
		return nil, recordNotFound(err)
	}
	return &j, nil
}

// AddCrowdTagsUserId mirrors crowd_tags_detail_mapper addCrowdTagsUserId.
func (d *DAO) AddCrowdTagsUserId(ctx context.Context, tx *gorm.DB, tagId, userId string) error {
	detail := &CrowdTagsDetail{TagId: tagId, UserId: userId}
	return checkAffected(tx.WithContext(ctx).Create(detail), 1)
}

// UpdateCrowdTagsStatistics mirrors crowd_tags_mapper updateCrowdTagsStatistics
// (statistics += count).
func (d *DAO) UpdateCrowdTagsStatistics(ctx context.Context, tx *gorm.DB, tagId string, count int) error {
	res := tx.WithContext(ctx).Model(&CrowdTags{}).
		Where("tag_id = ?", tagId).
		Updates(map[string]any{
			"statistics":  gorm.Expr("statistics + ?", count),
			"update_time": gorm.Expr("now()"),
		})
	return checkAffected(res, 1)
}
