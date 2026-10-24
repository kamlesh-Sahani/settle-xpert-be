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

// NextEmployeeIden derives the next sequential employee code (EMP-00001, EMP-00002, ...).
func NextEmployeeIden(ctx context.Context) (string, error) {
	collection := database.GetCollection("employees")

	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return "", err
	}

	maxSeq := int(total)

	cursor, err := collection.Find(
		ctx,
		bson.M{"emp_id": bson.M{"$exists": true, "$ne": ""}},
		options.Find().SetProjection(bson.M{"emp_id": 1}),
	)
	if err == nil {
		defer cursor.Close(ctx)
		var list []struct {
			EmpID string `bson:"emp_id"`
		}
		if err := cursor.All(ctx, &list); err == nil {
			for _, item := range list {
				numStr := digitRegex.FindString(item.EmpID)
				if numStr != "" {
					if n, convErr := strconv.Atoi(numStr); convErr == nil && n > maxSeq {
						maxSeq = n
					}
				}
			}
		}
	}

	seq := maxSeq + 1
	for {
		iden := fmt.Sprintf("EMP-%05d", seq)
		count, err := collection.CountDocuments(ctx, bson.M{
			"$or": []bson.M{
				{"emp_id": iden},
				{"emp_id": primitive.Regex{Pattern: fmt.Sprintf("(?i)^emp-0*%d$", seq)}},
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

// NextIden handles GET /api/v1/emp/next-iden
func NextIden(c *gin.Context) {
	iden, err := NextEmployeeIden(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Unable to generate employee code. Please try again.")
		return
	}

	response.Success(c, "Next employee code fetched", map[string]string{
		"empId": iden,
	})
}
