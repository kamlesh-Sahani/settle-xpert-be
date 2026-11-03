package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	RoleStatusActive   = "ACTIVE"
	RoleStatusInactive = "IN_ACTIVE"
)

type WorkingHours struct {
	Enabled   bool   `bson:"enabled" json:"enabled"`
	StartTime string `bson:"start_time,omitempty" json:"startTime,omitempty"`
	EndTime   string `bson:"end_time,omitempty" json:"endTime,omitempty"`
	Days      []int  `bson:"days,omitempty" json:"days,omitempty"`
}

func (wh *WorkingHours) IsAllowedNow() bool {
	if wh == nil || !wh.Enabled {
		return true
	}

	now := time.Now()

	if len(wh.Days) > 0 {
		dayAllowed := false
		currentDay := int(now.Weekday())
		for _, d := range wh.Days {
			if d == currentDay {
				dayAllowed = true
				break
			}
		}
		if !dayAllowed {
			return false
		}
	}

	if wh.StartTime == "" || wh.EndTime == "" {
		return true
	}

	currentTime := now.Format("15:04")
	if wh.StartTime <= wh.EndTime {
		return currentTime >= wh.StartTime && currentTime <= wh.EndTime
	}
	return currentTime >= wh.StartTime || currentTime <= wh.EndTime
}

type Role struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Description  string             `bson:"description" json:"description"`
	Status       string             `bson:"status" json:"status"`
	Permissions  []string           `bson:"permissions" json:"permissions"`
	WorkingHours *WorkingHours      `bson:"working_hours,omitempty" json:"workingHours,omitempty"`
	IsDeleted    bool               `bson:"is_deleted" json:"isDeleted"`
	CreatedAt    time.Time          `bson:"created_at" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updatedAt"`
}
