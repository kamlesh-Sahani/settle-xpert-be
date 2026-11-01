package controllers

import (
	"fmt"

	"lead-crm/backend/internal/customer/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/bson"
)

func List(c *gin.Context) {
	baseFilter := bson.M{
		"deleted_at": bson.M{"$exists": false},
	}

	table.ExecuteTableList[models.Customer](c, table.TableListConfig[models.Customer]{
		CollectionName: "customers",
		EntityName:     "customers",
		ExportFileName: "customers_export",
		BaseFilter:     baseFilter,
		SearchFields:   []string{"customer_number", "full_name", "phone", "email", "city"},
		AllowedSorts: map[string]string{
			"customer_number":   "customer_number",
			"full_name":         "full_name",
			"total_debt_amount": "total_debt_amount",
			"status":            "status",
			"created_at":        "created_at",
		},
		DefaultSort: "created_at",
		Projection: bson.M{
			"_id":               1,
			"customer_number":   1,
			"lead_id":           1,
			"lead_number":       1,
			"full_name":         1,
			"email":             1,
			"phone":             1,
			"city":              1,
			"total_debt_amount": 1,
			"status":            1,
			"assigned_to":       1,
			"assigned_to_name":  1,
			"created_at":        1,
		},
		Columns: []table.TableColumn{
			{ID: "customer_number", Title: "Customer #", Sortable: true, Width: 130, Type: table.ColumnTypeText},
			{ID: "full_name", Title: "Customer Name", Sortable: true, Width: 180, Type: table.ColumnTypeText},
			{ID: "phone", Title: "Phone", Sortable: false, Width: 140, Type: table.ColumnTypeText},
			{ID: "city", Title: "City", Sortable: false, Width: 120, Type: table.ColumnTypeText},
			{ID: "total_debt_amount", Title: "Total Debt (₹)", Sortable: true, Width: 140, Type: table.ColumnTypeText},
			{ID: "status", Title: "Status", Sortable: true, Width: 120, Type: table.ColumnTypeText},
			{ID: "assigned_to_name", Title: "Assigned Agent", Sortable: false, Width: 160, Type: table.ColumnTypeText},
			{ID: "created_at", Title: "Converted Date", Sortable: true, Width: 140, Type: table.ColumnTypeDate},
		},
		RowMapper: func(cust models.Customer) map[string][]table.TableCell {
			row := make(map[string][]table.TableCell)

			row["customer_number"] = []table.TableCell{table.NewLinkCell(cust.CustomerNumber, "/customers/"+cust.ID.Hex(), "#1890ff", true)}
			row["full_name"] = []table.TableCell{table.NewTextCell(cust.FullName, "#111827", true)}
			row["phone"] = []table.TableCell{table.NewTextCell(cust.Phone, "#4b5563", false)}

			city := cust.City
			if city == "" {
				city = "—"
			}
			row["city"] = []table.TableCell{table.NewTextCell(city, "#6b7280", false)}

			debtText := "—"
			if cust.TotalDebtAmount > 0 {
				debtText = fmt.Sprintf("₹%.0f", cust.TotalDebtAmount)
			}
			row["total_debt_amount"] = []table.TableCell{table.NewTextCell(debtText, "#e62e4d", true)}

			badgeColor := table.BadgeColors["SUCCESS"]
			if cust.Status != models.CustomerStatusActive {
				badgeColor = table.BadgeColors["INFO"]
			}
			row["status"] = []table.TableCell{table.Badge(string(cust.Status), badgeColor)}

			assignee := cust.AssignedToName
			if assignee == "" {
				assignee = "Unassigned"
			}
			row["assigned_to_name"] = []table.TableCell{table.NewTextCell(assignee, "#374151", false)}
			row["created_at"] = []table.TableCell{table.NewTextCell(cust.CreatedAt.Format("02 Jan 2006"), "", false)}

			return row
		},
	})
}
