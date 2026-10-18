package controllers

import (
	"strconv"

	"lead-crm/backend/internal/role/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/bson"
)

func List(c *gin.Context) {
	table.ExecuteTableList[models.Role](c, table.TableListConfig[models.Role]{
		CollectionName: "roles",
		EntityName:     "roles",
		ExportFileName: "roles_export",
		BaseFilter: bson.M{
			"is_deleted": bson.M{"$ne": true},
		},
		SearchFields: []string{"name", "description"},
		AllowedSorts: map[string]string{
			"name":       "name",
			"created_at": "created_at",
		},
		DefaultSort: "created_at",
		Projection: bson.M{
			"_id":         1,
			"name":        1,
			"description": 1,
			"permissions": 1,
			"status":      1,
			"is_deleted":  1,
			"created_at":  1,
		},
		Columns: []table.TableColumn{
			{ID: "name", Title: "Name", Sortable: true, Width: 250, Type: table.ColumnTypeText},
			{ID: "description", Title: "Description", Sortable: false, Width: 350, Type: table.ColumnTypeText},
			{ID: "permissions", Title: "Permissions", Sortable: false, Width: 150, Type: table.ColumnTypeNumber},
			{ID: "created_at", Title: "Created At", Sortable: true, Width: 150, Type: table.ColumnTypeDate},
		},
		RowMapper: func(r models.Role) map[string][]table.TableCell {
			row := make(map[string][]table.TableCell)

			row["name"] = []table.TableCell{table.NewLinkCell(r.Name, "/admin/roles/"+r.ID.Hex(), "#1890ff", true)}
			desc := r.Description
			if desc == "" {
				desc = "N/A"
			}
			row["description"] = []table.TableCell{table.NewTextCell(desc, "#6b7280", false)}
			row["permissions"] = []table.TableCell{table.NewTextCell(strconv.Itoa(len(r.Permissions)), "", false)}
			row["created_at"] = []table.TableCell{table.NewTextCell(r.CreatedAt.Format("02 Jan 2006"), "", false)}

			return row
		},
	})
}
