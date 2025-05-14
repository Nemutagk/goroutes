package helper

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/gofrs/uuid"
	"github.com/joho/godotenv"
)

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
