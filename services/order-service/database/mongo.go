package database

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

var (
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
)

// ConnectMongoDB initializes connection to MongoDB for IoT plant telemetry & clickstream logs.
func ConnectMongoDB() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		host := os.Getenv("MONGO_HOST")
		if host == "" {
			host = "mongo"
		}
		port := os.Getenv("MONGO_PORT")
		if port == "" {
			port = "27017"
		}
		uri = "mongodb://" + host + ":" + port
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri).SetTimeout(3 * time.Second)
	client, err := mongo.Connect(clientOpts)
	if err != nil {
		log.Printf("[MongoDB] Warning: Failed to connect to MongoDB at %s: %v", uri, err)
		return
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		log.Printf("[MongoDB] Warning: Ping to MongoDB failed: %v (service may be initializing)", err)
		// Still keep client if connection will recover
	}

	dbName := os.Getenv("MONGO_DB_NAME")
	if dbName == "" {
		dbName = "kather_baksho_iot"
	}

	MongoClient = client
	MongoDB = client.Database(dbName)
	log.Printf("[MongoDB] Initialized connection to database '%s' at %s", dbName, uri)
}

// MongoPing checks whether MongoDB is currently reachable.
func MongoPing() error {
	if MongoClient == nil {
		return errors.New("mongodb client is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return MongoClient.Ping(ctx, readpref.Primary())
}
