package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	genai "github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func uploadFile(ctx context.Context, fileName string, client *genai.Client) genai.FileData {
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	file, err := client.UploadFile(ctx, "", f, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("URI for file %s with mimeType %s is %s\n", fileName, file.MIMEType, file.URI)

	// --- POLLING LOGIC ---
	// The file is not ready to be used until its state is ACTIVE.
	// We must poll the API until the processing is complete.
	for {
		// Get the latest status of the file.
		f, err := client.GetFile(ctx, file.Name)
		if err != nil {
			log.Fatalf("Failed to get file status for %s: %v", file.Name, err)
		}

		// If the file is active, we can stop polling and use it.
		if f.State == genai.FileStateActive {
			fmt.Printf("File '%s' is now active. URI: %s\n", f.DisplayName, f.URI)
			return genai.FileData{
				MIMEType: f.MIMEType,
				URI:      f.URI,
			}
		}

		// If the file processing failed, we can't continue.
		if f.State == genai.FileStateFailed {
			log.Fatalf("File processing failed for %s. State: %s", f.DisplayName, f.State)
		}

		fmt.Printf("File '%s' is still processing, waiting 5 seconds...\n", f.DisplayName)
		time.Sleep(5 * time.Second) // Wait before checking again.
	}
}

// This function correctly handles text files by reading them directly.
func consolidateTextFiles(folderPath string) (string, error) {
	var builder strings.Builder
	err := filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			builder.WriteString(fmt.Sprintf("\n--- START OF FILE: %s ---\n", path))
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				log.Printf("Warning: Could not read file %s: %v", path, readErr)
				builder.WriteString(fmt.Sprintf("Error reading file: %v", readErr))
			} else {
				builder.Write(content)
			}
			builder.WriteString(fmt.Sprintf("\n--- END OF FILE: %s ---\n", path))
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	return builder.String(), nil
}

func uploadFolder(ctx context.Context, folderPath string, client *genai.Client) []genai.FileData {
	filesData := make([]genai.FileData, 0)
	filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return nil
		}
		if !info.IsDir() {
			filesData = append(filesData, uploadFile(ctx, path, client))
		}
		return nil
	})

	return filesData
}

func getClient(ctx context.Context) *genai.Client {
	// Access your API key from the environment variable.
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	// Initialize the client.
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatal(err)
	}
	return client
}

type CacheEntry struct {
	FilesData   []genai.FileData
	LastUpdated time.Time
}

func loadCache(cacheFile string) (map[string]CacheEntry, error) {
	cache := make(map[string]CacheEntry)
	file, err := os.Open(cacheFile)
	if err != nil {
		if os.IsNotExist(err) {
			return cache, nil // Return empty cache if file doesn't exist
		}
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&cache)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func saveCache(cacheFile string, cache map[string]CacheEntry) error {
	file, err := os.Create(cacheFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(cache)
}

func getFilesData(ctx context.Context, client *genai.Client, cacheFile string, tuningGuidePath string, sampleConfigFolder string) []genai.FileData {
	cache, err := loadCache(cacheFile)
	if err != nil {
		log.Printf("Error loading cache: %v\n", err)
	}

	cacheKey := "all_files" // You can create more sophisticated keys if needed
	cachedData, found := cache[cacheKey]

	if found && time.Since(cachedData.LastUpdated) < 24*time.Hour {
		log.Println("Using cached file URIs")
		return cachedData.FilesData
	}

	log.Println("Uploading files and updating cache...")

	// filesData := make([]genai.FileData, 0)
	filesData := append(uploadFolder(ctx, sampleConfigFolder, client), uploadFile(ctx, tuningGuidePath, client))
	// filesData := uploadFolder(ctx, sampleConfigFolder, client)
	// filesData = append(filesData, uploadFile(ctx, tuningGuidePath, client))

	cache[cacheKey] = CacheEntry{
		FilesData:   filesData,
		LastUpdated: time.Now(),
	}
	if err := saveCache(cacheFile, cache); err != nil {
		log.Printf("Error saving cache: %v\n", err)
	}

	return filesData
}

func main() {
	ctx := context.Background()

	client := getClient(ctx)

	model := client.GenerativeModel("gemini-2.5-pro") // Select the model.

	// tuningGuidePath := "/home/abhishekmgupta_google_com/go-core/ai/GCSFuseTuningGuideFinal.pdf" // Path to your tuning guide.
	sampleConfigFolder := "/home/abhishekmgupta_google_com/go-core/ai/samples" // Path to your sample configurations.
	// cacheFile := "/home/abhishekmgupta_google_com/go-core/ai/file_cache.json"              // Path to the cache file.

	folderContent, err := consolidateTextFiles(sampleConfigFolder)
	if err != nil {
		log.Fatalf("Error consolidating text files: %v", err)
	}
	// tuningGuideData := uploadFile(ctx, tuningGuidePath, client)
	tuningGuideData := genai.FileData{
		MIMEType: "application/pdf",
		URI:      "https://generativelanguage.googleapis.com/v1beta/files/eb2jxbyh0dn0",
	}

	// filesData := getFilesData(ctx, client, cacheFile, tuningGuidePath, sampleConfigFolder)
	workloadFilePath := "/home/abhishekmgupta_google_com/go-core/ai/workload_details.txt"
	workloadData, err := os.ReadFile(workloadFilePath)
	if err != nil {
		log.Fatal(err)
	}

	workloadDetails := "Start of workload data\n" + string(workloadData) + "\nEnd of workload data\n"

	// Prepare the prompt.
	genericInstructions := `Use the tuning guide to understand what values to configure. 
				   I have also added some sample gcsfuse configs for gpu and tpu for checkpointing, serving and training workload.
				   Give equal importance to all sources and combine the details from all these sources.
				   Checkpointing is primarily write workload. Serving is mostly sequential read workload. Training is mostly random read workload.
				   Use cache-dir as /tmp if cache-dir is needed. File cache should be enabled only when the workload is not too big and can fit in the disk`
	promptQuery := `Generate a config for GCSFuse for the provided workload.
	               Just generate a YAML file which can be saved directly to a file. `

	prompt := []genai.Part{
		genai.Text(genericInstructions),
		genai.Text(promptQuery),
		genai.Text(workloadDetails),
		genai.Text("--- START OF SAMPLE CONFIGURATIONS ---"),
		genai.Text(folderContent),
		genai.Text("--- END OF SAMPLE CONFIGURATIONS ---"),
		tuningGuideData,
	}

	// Generate content.
	resp, err := model.GenerateContent(ctx, prompt...)
	if err != nil {
		log.Fatal(err)
	}

	// Print the response.
	var responseContent bytes.Buffer
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				responseContent.WriteString(fmt.Sprintf("%v", part))
			}
		}
	}

	// Save the generated config to a file.
	outputFile := "/home/abhishekmgupta_google_com/go-core/ai/generated_config.yaml"
	err = os.WriteFile(outputFile, responseContent.Bytes(), 0644)
	if err != nil {
		log.Printf("Error saving generated config: %v\n", err)
		fmt.Println(responseContent.String()) // Print to console as fallback
	} else {
		fmt.Printf("Generated config saved to: %s\n", outputFile)
	}
}
