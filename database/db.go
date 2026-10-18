package database

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

var (
	Client *mongo.Client
	DBName = "debt_crm"
)

func init() {
	table.SetDefaultCollectionResolver(GetCollection)
}

func ConnectMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if cs, err := connstring.ParseAndValidate(uri); err == nil && cs.Database != "" {
		DBName = cs.Database
	} else if envDB := os.Getenv("DB_NAME"); envDB != "" {
		DBName = envDB
	}

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to MongoDB successfully (database: %s)", DBName)
	Client = client
	return client, nil
}

func GetCollection(collectionName string) *mongo.Collection {
	if Client == nil {
		return nil
	}
	return Client.Database(DBName).Collection(strings.TrimSpace(collectionName))
}
