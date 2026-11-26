package controllers

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"lead-crm/backend/database"
	"lead-crm/backend/internal/lead/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	employeeCacheTTL = 30 * time.Second
	maxFilterClauses = 10
	maxInFilterItems = 50
)

var (
	empCacheMu      sync.RWMutex
	empNameCache    map[string]string
	empCacheExpires time.Time
)

var allowedLeadFilterFields = map[string]bool{
	"lead_number":                        true,
	"status":                             true,
	"assigned_to":                        true,
	"service_type":                       true,
	"created_at":                         true,
	"personal.full_name":                 true,
	"personal.phone":                     true,
	"personal.email":                     true,
	"personal.city":                      true,
	"personal.state":                     true,
	"financial.total_outstanding_amount": true,
	"financial.monthly_income":           true,
	"financial.harassment":               true,
}

func List(c *gin.Context) {
	canAssign := hasLeadPermission(c, "lead:assign")

	var employeeNames map[string]string
	if canAssign {
		employeeNames = fetchEmployeeNameMap(c.Request.Context())
	}

	projection := bson.M{
		"_id":                                1,
		"lead_number":                        1,
		"status":                             1,
		"service_type":                       1,
		"created_at":                         1,
		"personal.full_name":                 1,
		"personal.phone":                     1,
		"personal.email":                     1,
		"personal.city":                      1,
		"financial.total_outstanding_amount": 1,
		"financial.monthly_income":           1,
		"financial.harassment":               1,
	}
	if canAssign {
		projection["assigned_to"] = 1
	}

	columns := []table.TableColumn{
		{ID: "lead_number", Title: "Lead #", Sortable: true, Width: 120, Type: table.ColumnTypeText},
		{ID: "full_name", Title: "Borrower Name", Sortable: true, Width: 180, Type: table.ColumnTypeText},
		{ID: "phone", Title: "Phone", Sortable: false, Width: 140, Type: table.ColumnTypeText},
		{ID: "city", Title: "City", Sortable: false, Width: 130, Type: table.ColumnTypeText},
		{ID: "service_type", Title: "Service", Sortable: false, Width: 160, Type: table.ColumnTypeText},
		{ID: "total_outstanding_amount", Title: "Total Outstanding (₹)", Sortable: true, Width: 170, Type: table.ColumnTypeNumber},
		{ID: "monthly_income", Title: "Monthly Income (₹)", Sortable: true, Width: 160, Type: table.ColumnTypeNumber},
		{
			ID:       "harassment",
			Title:    "Harassment Calls",
			Sortable: false,
			Width:    150,
			Type:     table.ColumnTypeSelect,
			Options: []table.SelectOption{
				{Key: string(models.HarassmentStatusYes), Value: "Yes"},
				{Key: string(models.HarassmentStatusNo), Value: "No"},
			},
		},
		{
			ID:       "status",
			Title:    "Status",
			Sortable: true,
			Width:    140,
			Type:     table.ColumnTypeSelect,
			Options: []table.SelectOption{
				{Key: string(models.LeadStatusNew), Value: "New"},
				{Key: string(models.LeadStatusCallScheduled), Value: "Call Scheduled"},
				{Key: string(models.LeadStatusInterested), Value: "Interested"},
				{Key: string(models.LeadStatusNotInterested), Value: "Not Interested"},
				{Key: string(models.LeadStatusConverted), Value: "Converted"},
			},
		},
	}
	if canAssign {
		columns = append(columns, table.TableColumn{
			ID: "assigned_to", Title: "Assigned To", Sortable: false, Width: 160, Type: table.ColumnTypeText,
		})
	}
	columns = append(columns, table.TableColumn{
		ID: "created_at", Title: "Created At", Sortable: true, Width: 140, Type: table.ColumnTypeDate,
	})

	table.ExecuteTableList[models.Lead](c, table.TableListConfig[models.Lead]{
		CollectionName: "leads",
		EntityName:     "leads",
		ExportFileName: "leads_export",
		SoftDelete:     true,
		SearchFields: []string{
			"lead_number",
			"personal.full_name",
			"personal.phone",
			"personal.email",
			"personal.city",
		},
		AllowedSorts: map[string]string{
			"lead_number":              "lead_number",
			"full_name":                "personal.full_name",
			"total_outstanding_amount": "financial.total_outstanding_amount",
			"monthly_income":           "financial.monthly_income",
			"harassment":               "financial.harassment",
			"status":                   "status",
			"created_at":               "created_at",
		},
		DefaultSort: "created_at",
		Projection:  projection,
		Columns:     columns,
		TransformQuery: func(filter bson.M) bson.M {
			return sanitizeLeadFilter(filter, canAssign)
		},
		CursorExtractor: func(l models.Lead, sortField string) (string, any) {
			return l.ID.Hex(), extractLeadSortValue(l, sortField)
		},
		RowMapper: func(l models.Lead) map[string][]table.TableCell {
			return mapLeadRow(l, employeeNames, canAssign)
		},
	})
}

func mapLeadRow(l models.Lead, employeeNames map[string]string, canAssign bool) map[string][]table.TableCell {
	city := l.Personal.City
	if city == "" {
		city = "—"
	}

	service := l.ServiceType
	if service == "" {
		service = "—"
	}

	outstandingText := "—"
	if l.Financial != nil && l.Financial.TotalOutstandingAmount > 0 {
		outstandingText = fmt.Sprintf("₹%.0f", l.Financial.TotalOutstandingAmount)
	}

	incomeText := "—"
	if l.Financial != nil && l.Financial.MonthlyIncome > 0 {
		incomeText = fmt.Sprintf("₹%.0f", l.Financial.MonthlyIncome)
	}

	harassmentCell := table.NewTextCell("—", "#6b7280", false)
	if l.Financial != nil && l.Financial.Harassment != "" {
		switch l.Financial.Harassment {
		case models.HarassmentStatusYes:
			harassmentCell = table.Badge("Yes", table.BadgeColors["REJECTED"])
		case models.HarassmentStatusNo:
			harassmentCell = table.Badge("No", table.BadgeColors["SUCCESS"])
		}
	}

	row := map[string][]table.TableCell{
		"lead_number":              {table.NewLinkCell(l.LeadNumber, "/employee/leads/detail/"+l.ID.Hex(), "#1890ff", true)},
		"full_name":                {table.NewTextCell(l.Personal.FullName, "#111827", true)},
		"phone":                    {table.NewTextCell(l.Personal.Phone, "#4b5563", false)},
		"city":                     {table.NewTextCell(city, "#6b7280", false)},
		"service_type":             {table.NewTextCell(service, "#4b5563", false)},
		"total_outstanding_amount": {table.NewTextCell(outstandingText, "#e62e4d", true)},
		"monthly_income":           {table.NewTextCell(incomeText, "#111827", false)},
		"harassment":               {harassmentCell},
		"status":                   {leadStatusBadge(l.Status)},
		"created_at":               {table.NewTextCell(l.CreatedAt.Format("02 Jan 2006"), "", false)},
	}

	if canAssign {
		assignee := "Unassigned"
		if l.AssignedTo != nil && !l.AssignedTo.IsZero() {
			if name, ok := employeeNames[l.AssignedTo.Hex()]; ok && name != "" {
				assignee = name
			}
		}
		row["assigned_to"] = []table.TableCell{table.NewTextCell(assignee, "#374151", false)}
	}

	return row
}

func leadStatusBadge(status models.LeadStatus) table.TableCell {
	switch status {
	case models.LeadStatusNew:
		return table.Badge("New", table.BadgeColors["INFO"])
	case models.LeadStatusCallScheduled:
		return table.Badge("Call Scheduled", table.BadgeColors["PENDING"])
	case models.LeadStatusInterested:
		return table.Badge("Interested", table.BadgeColors["SPECIAL"])
	case models.LeadStatusNotInterested:
		return table.Badge("Not Interested", table.BadgeColors["REJECTED"])
	case models.LeadStatusConverted:
		return table.Badge("Converted", table.BadgeColors["SUCCESS"])
	default:
		return table.Badge(string(status), table.BadgeColors["NEUTRAL"])
	}
}

func extractLeadSortValue(l models.Lead, sortField string) any {
	switch sortField {
	case "lead_number":
		return l.LeadNumber
	case "personal.full_name":
		return l.Personal.FullName
	case "financial.total_outstanding_amount":
		if l.Financial != nil {
			return l.Financial.TotalOutstandingAmount
		}
		return 0.0
	case "financial.monthly_income":
		if l.Financial != nil {
			return l.Financial.MonthlyIncome
		}
		return 0.0
	case "status":
		return string(l.Status)
	case "created_at":
		return l.CreatedAt.Format(time.RFC3339Nano)
	default:
		return nil
	}
}

func fetchEmployeeNameMap(ctx context.Context) map[string]string {
	now := time.Now()
	empCacheMu.RLock()
	if empNameCache != nil && now.Before(empCacheExpires) {
		cached := empNameCache
		empCacheMu.RUnlock()
		return cached
	}
	empCacheMu.RUnlock()

	names := make(map[string]string)
	empCol := database.GetCollection("employees")
	if empCol == nil {
		return names
	}

	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	cursor, err := empCol.Find(queryCtx, bson.M{}, options.Find().SetProjection(bson.M{"_id": 1, "name": 1}))
	if err != nil || cursor == nil {
		return names
	}
	defer cursor.Close(queryCtx)

	var employees []struct {
		ID   primitive.ObjectID `bson:"_id"`
		Name string             `bson:"name"`
	}
	if err := cursor.All(queryCtx, &employees); err != nil {
		return names
	}

	for _, emp := range employees {
		if !emp.ID.IsZero() {
			names[emp.ID.Hex()] = emp.Name
		}
	}

	empCacheMu.Lock()
	empNameCache = names
	empCacheExpires = time.Now().Add(employeeCacheTTL)
	empCacheMu.Unlock()

	return names
}

func sanitizeLeadFilter(filter bson.M, canAssign bool) bson.M {
	cleaned := cleanFilterClause(filter, canAssign)
	cleaned["deleted_at"] = bson.M{"$exists": false}
	return cleaned
}

func cleanFilterClause(clause bson.M, canAssign bool) bson.M {
	out := bson.M{}
	for k, v := range clause {
		if k == "$and" || k == "$or" {
			if subClauses, ok := v.([]bson.M); ok {
				kept := make([]bson.M, 0, min(len(subClauses), maxFilterClauses))
				for _, sub := range subClauses {
					if len(kept) >= maxFilterClauses {
						break
					}
					if cleanedSub := cleanFilterClause(sub, canAssign); len(cleanedSub) > 0 {
						kept = append(kept, cleanedSub)
					}
				}
				if len(kept) > 0 {
					out[k] = kept
				}
			}
			continue
		}

		if !allowedLeadFilterFields[k] {
			continue
		}
		if k == "assigned_to" && !canAssign {
			continue
		}
		if len(out) >= maxFilterClauses {
			break
		}
		v = capFilterSlice(v)
		if k == "assigned_to" {
			out[k] = coerceObjectIDFilter(v)
			continue
		}
		out[k] = v
	}
	return out
}

func capFilterSlice(val any) any {
	m, ok := val.(bson.M)
	if !ok {
		return val
	}
	out := bson.M{}
	for op, opVal := range m {
		if slice, isSlice := opVal.([]any); isSlice && len(slice) > maxInFilterItems {
			out[op] = slice[:maxInFilterItems]
			continue
		}
		out[op] = opVal
	}
	return out
}

func coerceObjectIDFilter(val any) any {
	switch v := val.(type) {
	case string:
		if oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(v)); err == nil {
			return oid
		}
		return v
	case bson.M:
		out := bson.M{}
		for op, opVal := range v {
			switch s := opVal.(type) {
			case string:
				if oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(s)); err == nil {
					out[op] = oid
				} else {
					out[op] = s
				}
			case []any:
				converted := make([]any, 0, len(s))
				for _, item := range s {
					if str, ok := item.(string); ok {
						if oid, err := primitive.ObjectIDFromHex(strings.TrimSpace(str)); err == nil {
							converted = append(converted, oid)
							continue
						}
					}
					converted = append(converted, item)
				}
				out[op] = converted
			default:
				out[op] = opVal
			}
		}
		return out
	default:
		return val
	}
}
