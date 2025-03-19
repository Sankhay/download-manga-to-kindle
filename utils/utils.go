package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func ExtractLastNumbers(url string) string {
	re := regexp.MustCompile(`\d+`) // Captura todos os números
	matches := re.FindAllString(url, -1)

	if len(matches) > 0 {
		return matches[len(matches)-1] // Retorna o último número encontrado
	}
	return ""
}

func ImageNameToResized(filename string) string {
	// Regular expression to find "_<number>.jpg" at the end
	re := regexp.MustCompile(`_(\d+)(\.\w+)$`)

	// Replace with "_resized_<number>.jpg"
	newFilename := re.ReplaceAllString(filename, "_resized_${1}${2}")

	return newFilename
}

func DeleteAllImages() error {
	imageDir := "images"

	// Check if the directory exists
	if _, err := os.Stat(imageDir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", imageDir)
	}

	// Walk through the directory and delete each file
	err := filepath.Walk(imageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, only delete files
		if !info.IsDir() {
			err := os.Remove(path)
			if err != nil {
				return fmt.Errorf("failed to delete file %s: %w", path, err)
			}
			fmt.Println("Deleted:", path)
		}
		return nil
	})

	if err != nil {
		return err
	}

	fmt.Println("All images deleted successfully.")
	return nil
}

func ChangeExtensionToJpg(filename string) string {
	// Check if the filename ends with ".png"
	if strings.HasSuffix(filename, ".png") {
		// Replace the ".png" with ".jpg"
		return filename[:len(filename)-4] + ".jpg"
	} else if strings.HasSuffix(filename, ".webp") {
		return filename[:len(filename)-5] + ".jpg"
	}
	// Return the original filename if it's not a .png
	return filename
}
