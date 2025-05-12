package db

import (
	"context"
	"fmt"

	"github.com/Nemutagk/goroutes/helper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connection() (*mongo.Client, error) {
	dbHost := helper.GetEnv("DB_LOGS_HOST", "localhost")
	dbPort := helper.GetEnv("DB_LOGS_PORT", "27017")
	dbUser := helper.GetEnv("DB_LOGS_USER", "root")
	dbPass := helper.GetEnv("DB_LOGS_PASS", "password")
	dbName := helper.GetEnv("DB_LOGS_DATABASE", "test")
	dbAuthName := helper.GetEnv("DB_LOGS_AUTH_NAME", "admin")

	// Check if the environment variables are set
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPass == "" || dbName == "" || dbAuthName == "" {
		panic("missing required environment variables")
	}

	mongoUri := "mongodb://" + dbUser + ":" + dbPass + "@" + dbHost + ":" + dbPort + "/" + dbName + "?authSource=" + dbAuthName
	options := options.Client().ApplyURI(mongoUri)
	connection, err := mongo.Connect(context.TODO(), options)

	if err != nil {
		fmt.Println("Error connecting to MongoDB: ", err)
		return nil, err
	}

	return connection, nil
}
