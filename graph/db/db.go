package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var singlePgInstance *pgxpool.Pool

var singleMongoInstance *mongo.Client

var pgOnce sync.Once

var mongoOnce sync.Once

func GetPostgresPool() *pgxpool.Pool {
	pgOnce.Do(func() {
		fmt.Println("create first and only postgresql connection pool")
		dbpool, err := pgxpool.New(context.Background(), os.Getenv("POSTGRES_DATABASE_URL"))
		if err != nil {
			log.Fatalf("%v", err)
		}
		err = dbpool.Ping(context.Background())
		if err != nil {
			log.Fatalf("%v", err)
		}
		singlePgInstance = dbpool
	})
	return singlePgInstance
}

func GetMongoClient() *mongo.Client {
	mongoOnce.Do(func() {
		fmt.Println("create first and only mongodb connection")
		client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(os.Getenv("MONGO_DATABASE_URL")))
		if err != nil {
			log.Fatalf("unable to connect to mongo database: %v\n", err)
		}
		singleMongoInstance = client
	})
	return singleMongoInstance
}
