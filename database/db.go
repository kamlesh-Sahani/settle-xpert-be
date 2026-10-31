package database

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/kamlesh-Sahani/go-commons/table"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/mongo/driver/connstring"
)

var (
	mu     sync.RWMutex
	Client *mongo.Client
	DBName = "debt_crm"
)

func init() {
	table.SetDefaultCollectionResolver(GetCollection)
}

func ConnectMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resolvedDB := "debt_crm"
	if cs, err := connstring.ParseAndValidate(uri); err == nil && cs.Database != "" {
		resolvedDB = cs.Database
	} else if envDB := os.Getenv("DB_NAME"); envDB != "" {
		resolvedDB = envDB
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

	mu.Lock()
	Client = client
	DBName = resolvedDB
	mu.Unlock()

	log.Printf("Connected to MongoDB successfully (database: %s)", resolvedDB)
	return client, nil
}

func GetCollection(collectionName string) *mongo.Collection {
	mu.RLock()
	c := Client
	d := DBName
	mu.RUnlock()

	if c == nil {
		return nil
	}
	return c.Database(d).Collection(strings.TrimSpace(collectionName))
}
