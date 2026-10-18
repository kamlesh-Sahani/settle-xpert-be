package database

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func ConnectMongo(uri, dbName string) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}

	db := client.Database(dbName)
	d := &Database{
		Client: client,
		DB:     db,
	}

	if err := d.ensureIndexes(); err != nil {
		log.Printf("[WARN] Error ensuring indexes: %v", err)
	}

	return d, nil
}

func (d *Database) ensureIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Users unique email
	_, err := d.DB.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		log.Printf("Index error users: %v", err)
	}

	// Leads phone & email indexes
	_, err = d.DB.Collection("leads").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "phone", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "assigned_to", Value: 1}}},
		{Keys: bson.D{{Key: "lead_number", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	if err != nil {
		log.Printf("Index error leads: %v", err)
	}

	// Customers customer_number and phone
	_, err = d.DB.Collection("customers").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "customer_number", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "phone", Value: 1}}},
		{Keys: bson.D{{Key: "email", Value: 1}}},
		{Keys: bson.D{{Key: "user_id", Value: 1}}},
	})
	if err != nil {
		log.Printf("Index error customers: %v", err)
	}

	// Cases case_number, customer_id, status
	_, err = d.DB.Collection("cases").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "case_number", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "customer_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "case_manager_id", Value: 1}}},
	})
	if err != nil {
		log.Printf("Index error cases: %v", err)
	}

	// Loans customer_id
	_, err = d.DB.Collection("loans").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "customer_id", Value: 1}},
	})
	if err != nil {
		log.Printf("Index error loans: %v", err)
	}

	// Tasks assigned_to, related_id, due_date
	_, err = d.DB.Collection("tasks").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "assigned_to", Value: 1}}},
		{Keys: bson.D{{Key: "related_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
		{Keys: bson.D{{Key: "due_date", Value: 1}}},
	})
	if err != nil {
		log.Printf("Index error tasks: %v", err)
	}

	// Audit logs entity_type, entity_id
	_, err = d.DB.Collection("audit_logs").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "entity_type", Value: 1}, {Key: "entity_id", Value: 1}}},
		{Keys: bson.D{{Key: "timestamp", Value: -1}}},
	})
	if err != nil {
		log.Printf("Index error audit_logs: %v", err)
	}

	return nil
}
