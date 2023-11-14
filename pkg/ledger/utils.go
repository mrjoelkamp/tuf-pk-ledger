package ledger

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
)

func stripTrailingSlash(url string) string {
	return strings.TrimSuffix(url, "/")
}

func jsonStructToString(v any) (string, error) {
	stringData, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(stringData), nil
}

func hashString(input string) string {
	s256 := sha256.New()
	s256.Write([]byte(input))
	hashSum := s256.Sum(nil)
	hash := hex.EncodeToString(hashSum)
	return hash
}

func readJSONFile(path string, v any) error {
	// Read JSON file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Unmarshal JSON data into the struct
	err = json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	return nil
}

func writeJSONFile(path string, v any) error {
	// Marshal the struct into JSON format
	data, err := json.MarshalIndent(&v, "", "  ")
	if err != nil {
		return err
	}

	// Write JSON data to a file
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}

func fileExists(filePath string) (bool, error) {
	// Check if the file or path exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// File or path does not exist
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		// File or path already exists
		return true, nil
	}
}

func createFile(filePath string, data string) error {
	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write data to the file
	_, err = file.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}
