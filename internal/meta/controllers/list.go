package controllers

import (
	"lead-crm/backend/internal/meta/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/bson"
)

// List returns webhook logs using standard table.ExecuteTableList for DynamicTable
func List(c *gin.Context) {
	table.ExecuteTableList[models.MetaWebhookLog](c, table.TableListConfig[models.MetaWebhookLog]{
		CollectionName: "meta_webhook_logs",
		EntityName:     "meta_webhook_logs",
		ExportFileName: "meta_webhook_logs_export",
		BaseFilter:     bson.M{},
		SearchFields:   []string{"full_name", "phone", "email", "lead_number", "form_id", "leadgen_id"},
		AllowedSorts: map[string]string{
			"created_at":  "created_at",
			"status":      "status",
			"lead_number": "lead_number",
		},
		DefaultSort: "created_at",
		Projection: bson.M{
			"_id":           1,
			"leadgen_id":    1,
			"page_id":       1,
			"form_id":       1,
			"lead_number":   1,
			"full_name":     1,
			"phone":         1,
			"email":         1,
			"status":        1,
			"error_message": 1,
			"created_at":    1,
		},
		Columns: []table.TableColumn{
			{ID: "created_at", Title: "Timestamp", Sortable: true, Width: 180, Type: table.ColumnTypeDate},
			{ID: "status", Title: "Status", Sortable: true, Width: 120, Type: table.ColumnTypeText},
			{ID: "lead_number", Title: "Lead Number", Sortable: true, Width: 150, Type: table.ColumnTypeText},
			{ID: "full_name", Title: "Full Name", Sortable: true, Width: 180, Type: table.ColumnTypeText},
			{ID: "phone", Title: "Phone", Sortable: false, Width: 140, Type: table.ColumnTypeText},
			{ID: "email", Title: "Email", Sortable: false, Width: 200, Type: table.ColumnTypeText},
			{ID: "form_id", Title: "Form ID", Sortable: false, Width: 160, Type: table.ColumnTypeText},
			{ID: "details", Title: "Details", Sortable: false, Width: 250, Type: table.ColumnTypeText},
		},
		RowMapper: func(log models.MetaWebhookLog) map[string][]table.TableCell {
			row := make(map[string][]table.TableCell)

			row["created_at"] = []table.TableCell{table.NewTextCell(log.CreatedAt.Format("02 Jan 2006, 15:04:05"), "#6b7280", false)}

			badgeColor := table.BadgeColors["REJECTED"]
			if log.Status == "SUCCESS" {
				badgeColor = table.BadgeColors["SUCCESS"]
			} else if log.Status == "DUPLICATE" {
				badgeColor = table.BadgeColors["WARNING"]
			}
			row["status"] = []table.TableCell{table.Badge(log.Status, badgeColor)}

			leadNum := log.LeadNumber
			if leadNum == "" {
				leadNum = log.LeadgenID
			}
			row["lead_number"] = []table.TableCell{table.NewTextCell(leadNum, "#1890ff", true)}

			name := log.FullName
			if name == "" {
				name = "N/A"
			}
			row["full_name"] = []table.TableCell{table.NewTextCell(name, "#111827", true)}
			row["phone"] = []table.TableCell{table.NewTextCell(log.Phone, "#4b5563", false)}
			row["email"] = []table.TableCell{table.NewTextCell(log.Email, "#6b7280", false)}

			formID := log.FormID
			if formID == "" {
				formID = "-"
			}
			row["form_id"] = []table.TableCell{table.NewTextCell(formID, "#6b7280", false)}

			details := log.ErrorMessage
			if log.Status == "SUCCESS" {
				details = "Lead ingested into CRM"
			}
			row["details"] = []table.TableCell{table.NewTextCell(details, "", false)}

			return row
		},
	})
}
