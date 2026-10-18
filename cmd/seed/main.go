package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"lead-crm/backend/database"
	rolemodels "lead-crm/backend/internal/role/models"
	usermodels "lead-crm/backend/internal/user/models"
	"lead-crm/backend/utils"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	_ = godotenv.Load()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017/debt_crm"
	}

	_, err := database.ConnectMongoDB(mongoURI)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	ctx := context.Background()

	// 1. Ensure Roles with Granular Non-Wildcard Permissions
	roleCol := database.GetCollection("roles")
	allPerms := rolemodels.GetAllPermissionIDs()

	type RoleSeed struct {
		Name        string
		Description string
		Permissions []string
	}

	rolesToSeed := []RoleSeed{
		{
			Name:        "Super Admin",
			Description: "Full unrestricted administrative access across the platform",
			Permissions: allPerms, // All 34 granular permissions
		},
		{
			Name:        "Operations Admin",
			Description: "Operational administration over leads, cases, customers, and team members",
			Permissions: []string{
				"dashboard:read",
				"lead:read", "lead:create", "lead:update", "lead:assign", "lead:convert", "lead:delete",
				"customer:read", "customer:create", "customer:update", "customer:delete",
				"loan:read", "loan:create", "loan:update", "loan:delete",
				"case:read", "case:create", "case:update", "case:transition", "case:note",
				"assessment:read", "assessment:create", "assessment:update",
				"task:read", "task:create", "task:update", "task:complete",
				"complaint:read", "complaint:create", "complaint:update", "complaint:resolve",
				"member:read", "member:create", "member:update", "member:audit",
				"role:read",
				"meta:read", "meta:update",
			},
		},
		{
			Name:        "Sales Executive",
			Description: "Lead qualification, initial consultation, and customer onboarding",
			Permissions: []string{
				"dashboard:read",
				"lead:read", "lead:create", "lead:update", "lead:assign", "lead:convert",
				"customer:read", "customer:create",
				"task:read", "task:create", "task:update", "task:complete",
			},
		},
		{
			Name:        "Case Manager",
			Description: "Debt resolution case management, assessments, and client follow-ups",
			Permissions: []string{
				"dashboard:read",
				"customer:read", "customer:update",
				"loan:read", "loan:create", "loan:update",
				"case:read", "case:create", "case:update", "case:transition", "case:note",
				"assessment:read", "assessment:create", "assessment:update",
				"task:read", "task:create", "task:update", "task:complete",
				"complaint:read", "complaint:create", "complaint:update", "complaint:resolve",
			},
		},
		{
			Name:        "Support Agent",
			Description: "Customer service, grievance logging, and recovery harassment complaint resolution",
			Permissions: []string{
				"dashboard:read",
				"customer:read",
				"case:read",
				"complaint:read", "complaint:create", "complaint:update", "complaint:resolve",
				"task:read", "task:create", "task:update", "task:complete",
			},
		},
	}

	roleIDMap := make(map[string]primitive.ObjectID)

	for _, r := range rolesToSeed {
		var role rolemodels.Role
		err := roleCol.FindOne(ctx, bson.M{"name": r.Name, "is_deleted": bson.M{"$ne": true}}).Decode(&role)
		if err != nil {
			role = rolemodels.Role{
				ID:          primitive.NewObjectID(),
				Name:        r.Name,
				Description: r.Description,
				Permissions: r.Permissions,
				Status:      rolemodels.RoleStatusActive,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			_, err = roleCol.InsertOne(ctx, role)
			if err != nil {
				log.Fatalf("Failed to seed role %s: %v", r.Name, err)
			}
			fmt.Printf("✅ Created Role: %s (%d permissions)\n", r.Name, len(r.Permissions))
		} else {
			// Update permissions to keep in sync
			_, _ = roleCol.UpdateOne(ctx, bson.M{"_id": role.ID}, bson.M{"$set": bson.M{
				"permissions": r.Permissions,
				"updated_at":  time.Now(),
			}})
			fmt.Printf("ℹ️ Updated Role: %s (%d permissions)\n", r.Name, len(r.Permissions))
		}
		roleIDMap[r.Name] = role.ID
	}

	// 2. Ensure Default Seed Users
	userCol := database.GetCollection("users")

	type UserSeed struct {
		Name     string
		Email    string
		Password string
		RoleName string
		Phone    string
	}

	usersToSeed := []UserSeed{
		{
			Name:     "Super Admin",
			Email:    "admin@debtcrm.com",
			Password: "Admin@123456",
			RoleName: "Super Admin",
			Phone:    "+919876543210",
		},
		{
			Name:     "Rahul Sharma (Sales)",
			Email:    "sales@debtcrm.com",
			Password: "Sales@123456",
			RoleName: "Sales Executive",
			Phone:    "+919876543211",
		},
		{
			Name:     "Priya Patel (Case Mgr)",
			Email:    "case@debtcrm.com",
			Password: "Case@123456",
			RoleName: "Case Manager",
			Phone:    "+919876543212",
		},
		{
			Name:     "Amit Verma (Support)",
			Email:    "support@debtcrm.com",
			Password: "Support@123456",
			RoleName: "Support Agent",
			Phone:    "+919876543213",
		},
	}

	for _, u := range usersToSeed {
		roleID := roleIDMap[u.RoleName]
		var user usermodels.User
		err := userCol.FindOne(ctx, bson.M{"email": u.Email}).Decode(&user)
		if err != nil {
			hashed, _ := utils.Hash(u.Password)
			user = usermodels.User{
				ID:        primitive.NewObjectID(),
				Name:      u.Name,
				Email:     u.Email,
				Password:  hashed,
				Phone:     u.Phone,
				Status:    usermodels.UserStatusActive,
				RoleID:    &roleID,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			_, err = userCol.InsertOne(ctx, user)
			if err != nil {
				log.Fatalf("Failed to seed user %s: %v", u.Email, err)
			}
			fmt.Printf("✅ Created User: %s (%s) - Password: %s\n", u.Name, u.Email, u.Password)
		} else {
			// Update role
			_, _ = userCol.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{
				"role_id":    roleID,
				"status":     usermodels.UserStatusActive,
				"updated_at": time.Now(),
			}})
			fmt.Printf("ℹ️ User Exists: %s (%s) - Role: %s\n", u.Name, u.Email, u.RoleName)
		}
	}

	fmt.Println("\n🎉 Database Seeding Completed Successfully!")
}
