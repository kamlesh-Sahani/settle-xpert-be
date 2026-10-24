package controllers

import (
	"lead-crm/backend/database"
	"lead-crm/backend/internal/employee/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetAll(c *gin.Context) {
	// Fetch all roles once to map role IDs to names
	rolesCollection := database.GetCollection("roles")
	roleMap := make(map[string]string)
	if rolesCollection != nil {
		roleCursor, _ := rolesCollection.Find(c.Request.Context(), bson.M{"is_deleted": bson.M{"$ne": true}}, options.Find().SetProjection(bson.M{"_id": 1, "name": 1}))
		if roleCursor != nil {
			var roles []bson.M
			_ = roleCursor.All(c.Request.Context(), &roles)
			for _, r := range roles {
				if id, ok := r["_id"].(primitive.ObjectID); ok {
					if name, ok := r["name"].(string); ok {
						roleMap[id.Hex()] = name
					}
				}
			}
		}
	}

	deptsCollection := database.GetCollection("departments")
	deptMap := make(map[string]string)
	if deptsCollection != nil {
		deptCursor, _ := deptsCollection.Find(c.Request.Context(), bson.M{"deleted_at": bson.M{"$exists": false}}, options.Find().SetProjection(bson.M{"_id": 1, "name": 1}))
		if deptCursor != nil {
			var depts []bson.M
			_ = deptCursor.All(c.Request.Context(), &depts)
			for _, d := range depts {
				if id, ok := d["_id"].(primitive.ObjectID); ok {
					if name, ok := d["name"].(string); ok {
						deptMap[id.Hex()] = name
					}
				}
			}
		}
	}

	table.ExecuteTableList[models.Employee](c, table.TableListConfig[models.Employee]{
		CollectionName: "employees",
		EntityName:     "employees",
		ExportFileName: "employees_export",
		BaseFilter: bson.M{
			"deleted_at": bson.M{"$exists": false},
		},
		SearchFields: []string{"name", "email", "phone", "emp_id"},
		AllowedSorts: map[string]string{
			"emp_id":     "emp_id",
			"name":       "name",
			"email":      "email",
			"status":     "status",
			"created_at": "created_at",
		},
		DefaultSort: "created_at",
		Projection: bson.M{
			"_id":           1,
			"emp_id":        1,
			"name":          1,
			"email":         1,
			"phone":         1,
			"role_id":       1,
			"department_id": 1,
			"status":        1,
			"created_at":    1,
		},
		Columns: []table.TableColumn{
			{ID: "emp_id", Title: "Employee ID", Sortable: true, Width: 150, Type: table.ColumnTypeText},
			{ID: "name", Title: "Name", Sortable: true, Width: 220, Type: table.ColumnTypeText},
			{ID: "email", Title: "Email Address", Sortable: true, Width: 220, Type: table.ColumnTypeText},
			{ID: "role", Title: "Role", Sortable: false, Width: 160, Type: table.ColumnTypeText},
			{ID: "department", Title: "Department", Sortable: false, Width: 160, Type: table.ColumnTypeText},
			{
				ID:       "status",
				Title:    "Status",
				Sortable: true,
				Width:    130,
				Type:     table.ColumnTypeSelect,
				Options: []table.SelectOption{
					{Key: models.EmployeeStatusActive, Value: "Active"},
					{Key: models.EmployeeStatusSuspended, Value: "Suspended"},
				},
			},
			{ID: "created_at", Title: "Joined At", Sortable: true, Width: 140, Type: table.ColumnTypeDate},
		},
		RowMapper: func(u models.Employee) map[string][]table.TableCell {
			row := make(map[string][]table.TableCell)

			empIDDisplay := u.EmpID
			if empIDDisplay == "" {
				empIDDisplay = "-"
			}
			row["emp_id"] = []table.TableCell{table.NewTextCell(empIDDisplay, "#4b5563", false)}
			row["name"] = []table.TableCell{table.NewLinkCell(u.Name, "/admin/employee/"+u.ID.Hex(), "#1890ff", true)}
			row["email"] = []table.TableCell{table.NewTextCell(u.Email, "#6b7280", false)}

			roleName := "N/A"
			if u.RoleID != nil {
				if name, exists := roleMap[u.RoleID.Hex()]; exists {
					roleName = name
				}
			}
			row["role"] = []table.TableCell{table.NewTextCell(roleName, "", false)}

			deptName := "N/A"
			if u.DepartmentID != nil {
				if name, exists := deptMap[u.DepartmentID.Hex()]; exists {
					deptName = name
				}
			}
			row["department"] = []table.TableCell{table.NewTextCell(deptName, "", false)}
			row["created_at"] = []table.TableCell{table.NewTextCell(u.CreatedAt.Format("02 Jan 2006"), "", false)}

			if u.Status == models.EmployeeStatusActive {
				row["status"] = []table.TableCell{table.Badge("Active", table.BadgeColors["SUCCESS"])}
			} else {
				row["status"] = []table.TableCell{table.Badge(u.Status, table.BadgeColors["REJECTED"])}
			}

			return row
		},
	})
}
