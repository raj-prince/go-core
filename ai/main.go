package main

import (
	"bytes"
	"context"
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

func main() {
	ctx := context.Background()

	client := getClient(ctx)
	defer client.Close()
	model := client.GenerativeModel("gemini-2.5-pro") // Select the model.

	// Read all the sample config files and create a single string with all the content
	sampleConfigFolder := "/home/princer_google_com/dev/go-core/ai/samples" // Path to your sample configurations.
	folderContent, err := consolidateTextFiles(sampleConfigFolder)
	if err != nil {
		log.Fatalf("Error consolidating text files: %v", err)
	}

	// Read the tuning guide which is a PDF.
	// tuningGuidePath := "/home/princer_google_com/dev/go-core/ai/GCSFuseTuningGuideFinal.pdf" // Path to your tuning guide.
	// tuningGuideData := uploadFile(ctx, tuningGuidePath, client)
	tuningGuideData := genai.FileData{
		MIMEType: "application/pdf",
		URI:      "https://generativelanguage.googleapis.com/v1beta/files/cw1ho6k603w2", // Using the cached URI
	}

	// Read the workload details. We will determine the gcsfuse config based on these details.
	workloadFilePath := "/home/princer_google_com/workload_insight.yaml"
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
				   Use cache-dir as /tmp if cache-dir is needed. File cache should be enabled only when the workload is not too big and can fit in the disk.
				   Also, suggest lesser value of sequential-read-size-mb for example, 1MB, 2MB, 4MB. for random read workloads.`

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
	outputFile := "/home/princer_google_com/go-core/ai/generated_gcsfuse_config.yaml"
	if err := os.MkdirAll(filepath.Dir(outputFile), 0755); err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	f, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Error opening output file: %v", err)
	}
	defer f.Close()

	_, err = f.Write(responseContent.Bytes())
	if err != nil {
		log.Printf("Error saving generated config: %v\n", err)
		fmt.Println(responseContent.String()) // Print to console as fallback
	} else {
		fmt.Printf("Generated config saved to: %s\n", outputFile)
	}
}
