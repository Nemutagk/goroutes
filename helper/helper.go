package helper

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/joho/godotenv"
)

var loadEnvOnce sync.Once

func LoadEnv() {
	loadEnvOnce.Do(func() {
		fmt.Println("Loading environment variables...")
		if err := godotenv.Overload(); err != nil {
			fmt.Println("Error loading .env file: ", err)
		}
	})
}

func GetEnv(key string, defaultValue string) string {
	LoadEnv()

	value := os.Getenv(key)

	if value != "" {
		return value
	}

	return defaultValue
}

func GenerateUuid() string {
	return uuid.Must(uuid.NewV7()).String()
}

func PrettyPrint(data any) {
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Println("Error formatting JSON:", err)
		return
	}

	fmt.Println(string(prettyJSON))
}
