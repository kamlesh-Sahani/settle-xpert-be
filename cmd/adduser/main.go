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

	// 1. Get Super Admin Role
	roleCol := database.GetCollection("roles")
	var role rolemodels.Role
	err = roleCol.FindOne(ctx, bson.M{"name": "Super Admin"}).Decode(&role)
	if err != nil {
		log.Fatalf("Super Admin role not found: %v", err)
	}

	// 3. User details to create/upsert
	name := "Kamlesh"
	email := "kamlesh@debtcrm.com"
	password := "Admin@123456"
	phone := "+919876543299"

	hashed, err := utils.Hash(password)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	userCol := database.GetCollection("users")
	var existing usermodels.User
	err = userCol.FindOne(ctx, bson.M{"email": email}).Decode(&existing)
	if err == nil {
		// Update password & role
		_, err = userCol.UpdateOne(ctx, bson.M{"_id": existing.ID}, bson.M{"$set": bson.M{
			"password":   hashed,
			"role_id":    role.ID,
			"status":     usermodels.UserStatusActive,
			"updated_at": time.Now(),
		}})
		if err != nil {
			log.Fatalf("Failed to update user: %v", err)
		}
		fmt.Printf("✅ Updated user %s (%s) with Super Admin role and password %s\n", name, email, password)
		return
	}

	newUser := usermodels.User{
		ID:        primitive.NewObjectID(),
		Name:      name,
		Email:     email,
		Password:  hashed,
		Phone:     phone,
		Status:    usermodels.UserStatusActive,
		RoleID:    &role.ID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = userCol.InsertOne(ctx, newUser)
	if err != nil {
		log.Fatalf("Failed to insert user: %v", err)
	}

	fmt.Printf("✅ Created user %s (%s) with Super Admin role and password %s\n", name, email, password)
}
