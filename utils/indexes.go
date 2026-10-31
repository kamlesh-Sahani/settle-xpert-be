package utils

import (
	"context"
	"log"
	"time"

	"lead-crm/backend/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func EnsureCollectionIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	empCol := database.GetCollection("employees")
	if empCol != nil {
		cursor, err := empCol.Indexes().List(ctx)
		if err == nil && cursor != nil {
			defer cursor.Close(ctx)
			var rawIndexes []bson.M
			if err = cursor.All(ctx, &rawIndexes); err == nil {
				for _, idx := range rawIndexes {
					if idx["name"] == "idx_employees_email" {
						_, _ = empCol.Indexes().DropOne(ctx, "idx_employees_email")
					}
				}
			}
		}
	}

	indexMap := map[string][]mongo.IndexModel{
		"employees": {
			{
				Keys: bson.D{{Key: "email", Value: 1}},
				Options: options.Index().
					SetName("idx_employees_email").
					SetUnique(true).
					SetPartialFilterExpression(bson.M{
						"deleted_at": bson.M{"$exists": false},
					}),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("idx_employees_status"),
			},
			{
				Keys:    bson.D{{Key: "emp_id", Value: 1}},
				Options: options.Index().SetName("idx_employees_emp_id").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "department_id", Value: 1}, {Key: "deleted_at", Value: 1}},
				Options: options.Index().SetName("idx_employees_department_id"),
			},
			{
				Keys:    bson.D{{Key: "role_id", Value: 1}, {Key: "deleted_at", Value: 1}},
				Options: options.Index().SetName("idx_employees_role_id"),
			},
		},
		"sessions": {
			{
				Keys:    bson.D{{Key: "employee_id", Value: 1}},
				Options: options.Index().SetName("idx_sessions_employee_id"),
			},
			{
				Keys:    bson.D{{Key: "refresh_token", Value: 1}},
				Options: options.Index().SetName("idx_sessions_refresh_token"),
			},
			{
				Keys:    bson.D{{Key: "expires_at", Value: 1}},
				Options: options.Index().SetName("idx_sessions_expires_at").SetExpireAfterSeconds(0),
			},
		},
		"roles": {
			{
				Keys:    bson.D{{Key: "name", Value: 1}},
				Options: options.Index().SetName("idx_roles_name"),
			},
		},
		"departments": {
			{
				Keys: bson.D{{Key: "code", Value: 1}},
				Options: options.Index().
					SetName("idx_departments_code").
					SetUnique(true).
					SetPartialFilterExpression(bson.M{
						"deleted_at": bson.M{"$exists": false},
					}),
			},
			{
				Keys:    bson.D{{Key: "name", Value: 1}},
				Options: options.Index().SetName("idx_departments_name"),
			},
		},
		"leads": {
			{
				Keys:    bson.D{{Key: "lead_number", Value: 1}},
				Options: options.Index().SetName("idx_leads_lead_number").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "phone", Value: 1}},
				Options: options.Index().SetName("idx_leads_phone"),
			},
			{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetName("idx_leads_email"),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("idx_leads_status"),
			},
			{
				Keys:    bson.D{{Key: "assigned_to", Value: 1}},
				Options: options.Index().SetName("idx_leads_assigned_to"),
			},
			{
				Keys:    bson.D{{Key: "created_at", Value: -1}},
				Options: options.Index().SetName("idx_leads_created_at"),
			},
		},
		"meta_webhook_logs": {
			{
				Keys:    bson.D{{Key: "created_at", Value: -1}},
				Options: options.Index().SetName("idx_meta_webhook_logs_created_at"),
			},
			{
				Keys:    bson.D{{Key: "leadgen_id", Value: 1}},
				Options: options.Index().SetName("idx_meta_webhook_logs_leadgen_id"),
			},
		},
		"customers": {
			{
				Keys:    bson.D{{Key: "phone", Value: 1}},
				Options: options.Index().SetName("idx_customers_phone"),
			},
			{
				Keys:    bson.D{{Key: "email", Value: 1}},
				Options: options.Index().SetName("idx_customers_email"),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("idx_customers_status"),
			},
		},
		"cases": {
			{
				Keys:    bson.D{{Key: "customer_id", Value: 1}},
				Options: options.Index().SetName("idx_cases_customer_id"),
			},
			{
				Keys:    bson.D{{Key: "case_number", Value: 1}},
				Options: options.Index().SetName("idx_cases_case_number").SetUnique(true),
			},
			{
				Keys:    bson.D{{Key: "status", Value: 1}},
				Options: options.Index().SetName("idx_cases_status"),
			},
			{
				Keys:    bson.D{{Key: "case_manager_id", Value: 1}},
				Options: options.Index().SetName("idx_cases_manager_id"),
			},
		},
		"tasks": {
			{
				Keys:    bson.D{{Key: "assigned_to", Value: 1}, {Key: "status", Value: 1}},
				Options: options.Index().SetName("idx_tasks_assigned_status"),
			},
			{
				Keys:    bson.D{{Key: "due_date", Value: 1}},
				Options: options.Index().SetName("idx_tasks_due_date"),
			},
		},
		"audit_logs": {
			{
				Keys:    bson.D{{Key: "entity_id", Value: 1}, {Key: "action_on", Value: -1}},
				Options: options.Index().SetName("idx_audit_logs_entity_action_on"),
			},
			{
				Keys:    bson.D{{Key: "action_on", Value: -1}},
				Options: options.Index().SetName("idx_audit_logs_action_on"),
			},
		},
	}

	for collName, models := range indexMap {
		col := database.GetCollection(collName)
		if col == nil {
			continue
		}
		_, err := col.Indexes().CreateMany(ctx, models)
		if err != nil {
			log.Printf("[INDEX] Warning creating indexes for %s: %v", collName, err)
		}
	}
}
