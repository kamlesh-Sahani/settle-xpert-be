package controllers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"

	"lead-crm/backend/database"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var digitRegex = regexp.MustCompile(`\d+`)

// NextTeamIden derives the next sequential team code (TEAM-00001, TEAM-00002, ...).
// It inspects existing records and total count so codes are monotonically increasing and never reused.
func NextTeamIden(ctx context.Context) (string, error) {
	collection := database.GetCollection("users")

	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return "", err
	}

	maxSeq := int(total)

	// Scan existing team_ids to find the highest sequence number used so far
	cursor, err := collection.Find(
		ctx,
		bson.M{"team_id": bson.M{"$exists": true, "$ne": ""}},
		options.Find().SetProjection(bson.M{"team_id": 1}),
	)
	if err == nil {
		defer cursor.Close(ctx)
		var list []struct {
			TeamID string `bson:"team_id"`
		}
		if err := cursor.All(ctx, &list); err == nil {
			for _, item := range list {
				numStr := digitRegex.FindString(item.TeamID)
				if numStr != "" {
					if n, convErr := strconv.Atoi(numStr); convErr == nil {
						if n > maxSeq {
							maxSeq = n
						}
					}
				}
			}
		}
	}

	seq := maxSeq + 1
	for {
		iden := fmt.Sprintf("TEAM-%05d", seq)
		// Collision probe against both exact match and case-insensitive/prefix variants
		count, err := collection.CountDocuments(ctx, bson.M{
			"$or": []bson.M{
				{"team_id": iden},
				{"team_id": primitive.Regex{Pattern: fmt.Sprintf("(?i)^team-0*%d$", seq)}},
			},
		})
		if err != nil {
			return "", err
		}
		if count == 0 {
			return iden, nil
		}
		seq++
	}
}

// NextIden handles GET /api/v1/team/next-iden
func NextIden(c *gin.Context) {
	iden, err := NextTeamIden(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Unable to generate team code. Please try again.")
		return
	}

	response.Success(c, "Next team code fetched", map[string]string{
		"iden":   iden,
		"teamId": iden,
	})
}
