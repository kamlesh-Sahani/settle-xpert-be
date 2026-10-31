package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"lead-crm/backend/database"
	auditmodels "lead-crm/backend/internal/audit/models"

	"github.com/gin-gonic/gin"
	"github.com/kamlesh-Sahani/go-commons/response"
	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ModalData defines parameters for opening a modal from a LINK cell.
type ModalData struct {
	ApiURL  string         `json:"apiUrl"`
	ApiType string         `json:"apiType"`
	Payload map[string]any `json:"payload,omitempty"`
	Params  map[string]any `json:"params,omitempty"`
}

// AuditCell represents a formatted cell returned to DynamicTable.
type AuditCell struct {
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Color     string     `json:"color,omitempty"`
	IsBold    bool       `json:"isBold,omitempty"`
	ModalData *ModalData `json:"modalData,omitempty"`
}

type HistoryRequest struct {
	ID          string                        `json:"id"`
	EntityID    string                        `json:"entityId"`
	EntityType  string                        `json:"entityType"`
	Page        int                           `json:"page"`
	Limit       int                           `json:"limit"`
	Search      string                        `json:"search"`
	TableSearch string                        `json:"tableSearch"`
	SortBy      string                        `json:"sortBy"`
	SortField   string                        `json:"sortField"`
	Order       string                        `json:"order"`
	SortOrder   string                        `json:"sortOrder"`
	Filters     map[string]table.ColumnFilter `json:"filters"`
}

func (r *HistoryRequest) GetSearch() string {
	if r.Search != "" {
		return r.Search
	}
	return r.TableSearch
}

func (r *HistoryRequest) GetSortField() string {
	if r.SortBy != "" {
		return r.SortBy
	}
	return r.SortField
}

func (r *HistoryRequest) GetSortOrder() string {
	if r.Order != "" {
		return r.Order
	}
	return r.SortOrder
}

func (r *HistoryRequest) GetEntityID() string {
	if r.EntityID != "" {
		return r.EntityID
	}
	return r.ID
}

// GetAuditHistory executes a table listing for audit history matching optional entity_id.
// Columns: Action (LINK with modalData), No of changes (TEXT), Action On (TEXT), Action By (TEXT)
func GetAuditHistory(c *gin.Context) {
	var req HistoryRequest
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &req)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	targetID := req.GetEntityID()
	if targetID == "" {
		targetID = c.Query("entityId")
	}
	if targetID == "" {
		targetID = c.Query("id")
	}

	page := req.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	col := database.GetCollection("audit_logs")
	if col == nil {
		response.InternalServerError(c, "Database collection not found")
		return
	}

	baseFilter := bson.M{}
	if targetID != "" {
		baseFilter["entity_id"] = targetID
	}

	search := strings.TrimSpace(req.GetSearch())
	if search != "" {
		pattern := primitive.Regex{Pattern: regexp.QuoteMeta(search), Options: "i"}
		baseFilter["$or"] = []bson.M{
			{"action": pattern},
			{"action_by": pattern},
		}
	}

	// Apply column filters (e.g. Action filter)
	if req.Filters != nil {
		if actionFilter, ok := req.Filters["action"]; ok && len(actionFilter.Value) > 0 {
			var actions []string
			for _, v := range actionFilter.Value {
				if strings.EqualFold(v, "Created") {
					actions = append(actions, auditmodels.ActionCreate)
				} else if strings.EqualFold(v, "Updated") {
					actions = append(actions, auditmodels.ActionUpdate)
				} else {
					actions = append(actions, strings.ToUpper(v))
				}
			}
			if len(actions) == 1 {
				baseFilter["action"] = actions[0]
			} else if len(actions) > 1 {
				baseFilter["action"] = bson.M{"$in": actions}
			}
		}
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	totalRows, err := col.CountDocuments(ctx, baseFilter)
	if err != nil {
		totalRows = 0
	}

	sortField := "action_on"
	sortDir := -1
	sortFieldParam := req.GetSortField()
	if sortFieldParam != "" {
		switch sortFieldParam {
		case "action", "action_by", "no_of_changes", "action_on":
			sortField = sortFieldParam
		}
	}
	sortOrderParam := req.GetSortOrder()
	if sortOrderParam == "asc" || sortOrderParam == "1" {
		sortDir = 1
	}

	findOptions := options.Find().
		SetSort(bson.D{{Key: sortField, Value: sortDir}}).
		SetSkip(int64((page - 1) * limit)).
		SetLimit(int64(limit))

	cursor, err := col.Find(ctx, baseFilter, findOptions)
	if err != nil {
		response.InternalServerError(c, "Failed to retrieve audit history")
		return
	}
	defer cursor.Close(ctx)

	var items []auditmodels.AuditLog
	if err := cursor.All(ctx, &items); err != nil {
		response.InternalServerError(c, "Failed to parse audit history")
		return
	}

	// Columns definition sent to DynamicTable
	columns := []table.TableColumn{
		{
			ID:       "action",
			Title:    "Action",
			Sortable: true,
			Width:    130,
			Type:     table.ColumnTypeSelect,
			Options: []table.SelectOption{
				{Key: "Updated", Value: "Updated"},
				{Key: "Created", Value: "Created"},
			},
		},
		{
			ID:       "no_of_changes",
			Title:    "No of changes",
			Sortable: true,
			Width:    140,
		},
		{
			ID:       "action_on",
			Title:    "Action On",
			Sortable: true,
			Width:    180,
			Type:     table.ColumnTypeDate,
		},
		{
			ID:       "action_by",
			Title:    "Action By",
			Sortable: true,
			Width:    180,
			Type:     table.ColumnTypeText,
		},
	}

	rows := make([]map[string][]AuditCell, 0, len(items))
	for _, a := range items {
		actionDisplay := "Updated"
		if a.Action == auditmodels.ActionCreate {
			actionDisplay = "Created"
		}
		actionBy := a.ActionBy
		if actionBy == "" {
			actionBy = "System"
		}

		noOfChanges := a.NoOfChanges
		if noOfChanges == 0 {
			_, count := ComputeDiff(a.OldData, a.NewData, a.Action)
			if count > 0 {
				noOfChanges = count
			} else if a.Action == auditmodels.ActionCreate {
				noOfChanges = 1
			}
		}

		row := make(map[string][]AuditCell)

		// 1. Action: LINK type cell with modalData
		detailsURL := strings.TrimSuffix(c.Request.URL.Path, "/") + "/details"
		row["action"] = []AuditCell{
			{
				Type:   "LINK",
				Title:  actionDisplay,
				Color:  "#e11d48",
				IsBold: true,
				ModalData: &ModalData{
					ApiURL:  detailsURL,
					ApiType: "POST",
					Payload: map[string]any{
						"id": a.ID.Hex(),
					},
				},
			},
		}

		// 2. No of changes: simple TEXT response
		row["no_of_changes"] = []AuditCell{
			{
				Type:  "TEXT",
				Title: fmt.Sprintf("%d", noOfChanges),
				Color: "#111827",
			},
		}

		// 3. Action On: simple TEXT response
		row["action_on"] = []AuditCell{
			{
				Type:  "TEXT",
				Title: a.ActionOn.Format("2006-01-02 15:04"),
				Color: "#374151",
			},
		}

		// 4. Action By: simple TEXT response
		row["action_by"] = []AuditCell{
			{
				Type:  "TEXT",
				Title: actionBy,
				Color: "#374151",
			},
		}

		rows = append(rows, row)
	}

	totalPages := int(math.Ceil(float64(totalRows) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	responseData := gin.H{
		"columns":     columns,
		"rows":        rows,
		"features":    table.BuildTableFeatures(true, true, true, true, totalRows > 0),
		"isPaginated": true,
		"pageNumber":  page,
		"totalPages":  totalPages,
		"pageSize":    limit,
		"totalRows":   int(totalRows),
	}

	message := "Audit history fetched successfully"
	if totalRows == 0 {
		message = "No audit history found"
	}

	response.Success(c, message, responseData)
}

// GetAuditHistoryDetails returns the field-level changes (Field, Older Value, Changed Value)
// for a single audit log entry.
func GetAuditHistoryDetails(c *gin.Context) {
	var req HistoryRequest
	if c.Request.Body != nil {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err == nil && len(bodyBytes) > 0 {
			_ = json.Unmarshal(bodyBytes, &req)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	rawID := req.ID
	if rawID == "" {
		rawID = req.EntityID
	}
	if rawID == "" {
		rawID = c.Query("id")
	}
	if rawID == "" {
		rawID = c.Param("id")
	}

	if rawID == "" {
		response.BadRequest(c, "Audit log ID is required")
		return
	}

	objID, err := primitive.ObjectIDFromHex(rawID)
	if err != nil {
		response.BadRequest(c, "Invalid audit log ID format")
		return
	}

	col := database.GetCollection("audit_logs")
	if col == nil {
		response.InternalServerError(c, "Database collection not found")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	findFilter := bson.M{"_id": objID}

	var doc auditmodels.AuditLog
	err = col.FindOne(ctx, findFilter).Decode(&doc)
	if err != nil {
		response.NotFound(c, "Audit log record not found")
		return
	}

	actionDisplay := "Updated"
	if doc.Action == auditmodels.ActionCreate {
		actionDisplay = "Created"
	}

	actionBy := doc.ActionBy
	if actionBy == "" {
		actionBy = "System"
	}

	diffs, _ := ComputeDiff(doc.OldData, doc.NewData, doc.Action)

	// Columns definition for the audit details table
	columns := []table.TableColumn{
		{
			ID:       "field",
			Title:    "Field",
			Sortable: false,
			Width:    200,
			Type:     table.ColumnTypeText,
		},
		{
			ID:       "older_value",
			Title:    "Older Value",
			Sortable: false,
			Width:    250,
		},
		{
			ID:       "changed_value",
			Title:    "Changed Value",
			Sortable: false,
			Width:    250,
		},
	}

	// Filter by search query if provided
	search := strings.TrimSpace(req.GetSearch())
	if search != "" {
		searchTerm := strings.ToLower(search)
		var filtered []FieldDiff
		for _, d := range diffs {
			if strings.Contains(strings.ToLower(d.Field), searchTerm) ||
				strings.Contains(strings.ToLower(d.OlderValue), searchTerm) ||
				strings.Contains(strings.ToLower(d.ChangedValue), searchTerm) {
				filtered = append(filtered, d)
			}
		}
		diffs = filtered
	}

	// Filter by Field column if provided in Filters
	if req.Filters != nil {
		if fieldFilter, ok := req.Filters["field"]; ok && len(fieldFilter.Value) > 0 {
			op := fieldFilter.Operator
			var filtered []FieldDiff
			for _, d := range diffs {
				targetField := strings.ToLower(d.Field)
				targetFieldSnake := strings.ReplaceAll(targetField, " ", "_")
				match := false
				for _, val := range fieldFilter.Value {
					query := strings.ToLower(strings.TrimSpace(val))
					if query == "" {
						continue
					}
					switch op {
					case table.FilterOpStartsWith, "starts_with", "start_with":
						if strings.HasPrefix(targetField, query) || strings.HasPrefix(targetFieldSnake, query) {
							match = true
						}
					case table.FilterOpEqual, "equal", "eq":
						if targetField == query || targetFieldSnake == query {
							match = true
						}
					case table.FilterOpNotEqual, "not_equal", "ne":
						if targetField != query && targetFieldSnake != query {
							match = true
						}
					default: // contains
						if strings.Contains(targetField, query) || strings.Contains(targetFieldSnake, query) {
							match = true
						}
					}
					if match {
						break
					}
				}
				if match {
					filtered = append(filtered, d)
				}
			}
			diffs = filtered
		}
	}

	totalDiffs := len(diffs)
	page := req.Page
	if page <= 0 {
		page = 1
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	start := (page - 1) * limit
	if start > totalDiffs {
		start = totalDiffs
	}
	end := start + limit
	if end > totalDiffs {
		end = totalDiffs
	}
	pagedDiffs := diffs[start:end]

	rows := make([]map[string][]AuditCell, 0, len(pagedDiffs))
	for _, d := range pagedDiffs {
		row := make(map[string][]AuditCell)
		row["field"] = []AuditCell{
			{
				Type:   "TEXT",
				Title:  d.Field,
				IsBold: false,
				Color:  "#111827",
			},
		}
		row["older_value"] = []AuditCell{
			{
				Type:   "TEXT",
				Title:  d.OlderValue,
				IsBold: false,
				Color:  "#6b7280",
			},
		}
		row["changed_value"] = []AuditCell{
			{
				Type:   "TEXT",
				Title:  d.ChangedValue,
				IsBold: false,
				Color:  "#111827",
			},
		}
		rows = append(rows, row)
	}

	totalPages := int(math.Ceil(float64(totalDiffs) / float64(limit)))
	if totalPages == 0 {
		totalPages = 1
	}

	responseData := gin.H{
		"columns":     columns,
		"rows":        rows,
		"features":    table.BuildTableFeatures(true, false, true, true, totalDiffs > 0),
		"isPaginated": true,
		"pageNumber":  page,
		"totalPages":  totalPages,
		"pageSize":    limit,
		"totalRows":   totalDiffs,
		"id":          doc.ID.Hex(),
		"action":      actionDisplay,
		"action_on":   doc.ActionOn.Format("2006-01-02 15:04"),
		"action_by":   actionBy,
		"changes":     diffs,
	}

	response.Success(c, "Audit log details fetched successfully", responseData)
}
