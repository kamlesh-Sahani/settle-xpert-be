package controllers

import (
	"fmt"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/bson"
)

func List(c *gin.Context) {
	baseFilter := bson.M{
		"deleted_at": bson.M{"$exists": false},
	}

	table.ExecuteTableList[models.Lead](c, table.TableListConfig[models.Lead]{
		CollectionName: "leads",
		EntityName:     "leads",
		ExportFileName: "leads_export",
		BaseFilter:     baseFilter,
		SearchFields: []string{"full_name", "phone", "email", "lead_number", "city"},
		AllowedSorts: map[string]string{
			"lead_number":       "lead_number",
			"full_name":         "full_name",
			"total_debt_amount": "total_debt_amount",
			"status":            "status",
			"created_at":        "created_at",
		},
		DefaultSort: "created_at",
		Projection: bson.M{
			"_id":               1,
			"lead_number":       1,
			"full_name":         1,
			"email":             1,
			"phone":             1,
			"city":              1,
			"source":            1,
			"total_debt_amount": 1,
			"monthly_income":    1,
			"current_emi_amount": 1,
			"status":            1,
			"assigned_to":       1,
			"assigned_to_name":  1,
			"created_at":        1,
		},
		Columns: []table.TableColumn{
			{ID: "lead_number", Title: "Lead #", Sortable: true, Width: 120, Type: table.ColumnTypeText},
			{ID: "full_name", Title: "Borrower Name", Sortable: true, Width: 180, Type: table.ColumnTypeText},
			{ID: "phone", Title: "Phone", Sortable: false, Width: 140, Type: table.ColumnTypeText},
			{ID: "total_debt_amount", Title: "Total Debt (₹)", Sortable: true, Width: 140, Type: table.ColumnTypeText},
			{ID: "source", Title: "Source", Sortable: false, Width: 130, Type: table.ColumnTypeText},
			{
				ID:       "status",
				Title:    "Status",
				Sortable: true,
				Width:    130,
				Type:     table.ColumnTypeSelect,
				Options: []table.SelectOption{
					{Key: string(models.LeadStatusNew), Value: "New"},
					{Key: string(models.LeadStatusInProgress), Value: "In Progress"},
					{Key: string(models.LeadStatusConverted), Value: "Converted"},
					{Key: string(models.LeadStatusLost), Value: "Lost"},
				},
			},
			{ID: "assigned_to_name", Title: "Assigned To", Sortable: false, Width: 160, Type: table.ColumnTypeText},
			{ID: "created_at", Title: "Created At", Sortable: true, Width: 140, Type: table.ColumnTypeDate},
		},
		RowMapper: func(l models.Lead) map[string][]table.TableCell {
			row := make(map[string][]table.TableCell)

			row["lead_number"] = []table.TableCell{table.NewLinkCell(l.LeadNumber, "/employee/leads/detail/"+l.ID.Hex(), "#1890ff", true)}
			row["full_name"] = []table.TableCell{table.NewTextCell(l.FullName, "#111827", true)}
			row["phone"] = []table.TableCell{table.NewTextCell(l.Phone, "#4b5563", false)}

			debtText := "—"
			if l.TotalDebtAmount > 0 {
				debtText = fmt.Sprintf("₹%.0f", l.TotalDebtAmount)
			}
			row["total_debt_amount"] = []table.TableCell{table.NewTextCell(debtText, "#e62e4d", true)}

			row["source"] = []table.TableCell{table.NewTextCell(string(l.Source), "#6b7280", false)}

			switch l.Status {
			case models.LeadStatusNew:
				row["status"] = []table.TableCell{table.Badge("New", table.BadgeColors["INFO"])}
			case models.LeadStatusInProgress:
				row["status"] = []table.TableCell{table.Badge("In Progress", table.BadgeColors["WARNING"])}
			case models.LeadStatusConverted:
				row["status"] = []table.TableCell{table.Badge("Converted", table.BadgeColors["SUCCESS"])}
			case models.LeadStatusLost:
				row["status"] = []table.TableCell{table.Badge("Lost", table.BadgeColors["REJECTED"])}
			default:
				row["status"] = []table.TableCell{table.NewTextCell(string(l.Status), "", false)}
			}

			assignee := l.AssignedToName
			if assignee == "" {
				assignee = "Unassigned"
			}
			row["assigned_to_name"] = []table.TableCell{table.NewTextCell(assignee, "#374151", false)}
			row["created_at"] = []table.TableCell{table.NewTextCell(l.CreatedAt.Format("02 Jan 2006"), "", false)}

			return row
		},
	})
}
