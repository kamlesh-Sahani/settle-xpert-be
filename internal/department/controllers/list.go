package controllers

import (
	"lead-crm/backend/internal/department/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/bson"
)

func List(c *gin.Context) {
	table.ExecuteTableList[models.Department](c, table.TableListConfig[models.Department]{
		CollectionName: "departments",
		EntityName:     "departments",
		ExportFileName: "departments_export",
		BaseFilter: bson.M{
			"deleted_at": bson.M{"$exists": false},
		},
		SearchFields: []string{"name", "code", "description"},
		AllowedSorts: map[string]string{
			"name":       "name",
			"code":       "code",
			"status":     "status",
			"created_at": "created_at",
		},
		DefaultSort: "created_at",
		Projection: bson.M{
			"_id":         1,
			"name":        1,
			"code":        1,
			"description": 1,
			"status":      1,
			"created_at":  1,
		},
		Columns: []table.TableColumn{
			{ID: "name", Title: "Department Name", Sortable: true, Width: 220, Type: table.ColumnTypeText},
			{ID: "code", Title: "Code", Sortable: true, Width: 120, Type: table.ColumnTypeText},
			{ID: "description", Title: "Description", Sortable: false, Width: 300, Type: table.ColumnTypeText},
			{
				ID:       "status",
				Title:    "Status",
				Sortable: true,
				Width:    130,
				Type:     table.ColumnTypeSelect,
				Options: []table.SelectOption{
					{Key: string(models.DepartmentStatusActive), Value: "Active"},
					{Key: string(models.DepartmentStatusInactive), Value: "Inactive"},
				},
			},
			{ID: "created_at", Title: "Created At", Sortable: true, Width: 150, Type: table.ColumnTypeDate},
		},
		RowMapper: func(d models.Department) map[string][]table.TableCell {
			row := make(map[string][]table.TableCell)

			row["name"] = []table.TableCell{table.NewLinkCell(d.Name, "/admin/departments/"+d.ID.Hex(), "#1890ff", true)}
			row["code"] = []table.TableCell{table.NewTextCell(d.Code, "#111827", true)}

			desc := d.Description
			if desc == "" {
				desc = "-"
			}
			row["description"] = []table.TableCell{table.NewTextCell(desc, "#6b7280", false)}

			if d.Status == models.DepartmentStatusActive {
				row["status"] = []table.TableCell{table.Badge("Active", table.BadgeColors["SUCCESS"])}
			} else {
				row["status"] = []table.TableCell{table.Badge("Inactive", table.BadgeColors["REJECTED"])}
			}

			row["created_at"] = []table.TableCell{table.NewTextCell(d.CreatedAt.Format("02 Jan 2006"), "", false)}

			return row
		},
	})
}
