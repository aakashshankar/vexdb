package cmd

import (
	"fmt"
	"github.com/aakashshankar/vexdb/embed"
	"github.com/aakashshankar/vexdb/embed/openai"
	"github.com/aakashshankar/vexdb/storage"
	"github.com/aakashshankar/vexdb/tokenize/bert"
	"github.com/ledongthuc/pdf"
	"github.com/spf13/cobra"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	openaiMaxTokens    = 8191
	sixtyFourMegabytes = 64 * 1024 * 1024
)

var embedCmd = &cobra.Command{
	Use:   "embed [source path] [destination directory]",
	Short: "Generate embeddings for a file",
	Long:  `This command generates embeddings for the content of the specified file using the OpenAI embedder.`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceDir := args[0]
		destPath := args[1]

		files, err := os.ReadDir(destPath)

		if len(files) > 0 {
			return fmt.Errorf("destination directory not empty: %v", destPath)
		}

		_ = os.MkdirAll(destPath, 0777)

		embedder, err := openai.NewOpenAIEmbedder()
		if err != nil {
			return fmt.Errorf("error initializing embedder: %w", err)
		}
		// FATAL: Will not flush to disk if data is less
		store := storage.NewStore(sixtyFourMegabytes, destPath)

		err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			base := filepath.Base(path)
			if !info.IsDir() && base[0] != '.' {
				ext := strings.ToLower(filepath.Ext(path))

				var content string

				switch ext {
				case ".txt", ".md", ".go", ".py", ".java", ".js", ".c", ".cpp", ".rs":
					data, err := os.ReadFile(path)
					if err != nil {
						return fmt.Errorf("could not read file %s: %v", path, err)
					}
					content = string(data)
				case ".pdf":
					content, err = extractTextFromPDF(path)
					if err != nil {
						return fmt.Errorf("could not extract text from pdf file %s: %v", path, err)
					}
				}

				err = embedContentAndStore(embedder, content, path, store)

				if err != nil {
					return err
				}

			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("error processing files in %s: %w", sourceDir, err)
		}

		err = store.Flush()
		if err != nil {
			return fmt.Errorf("failed to Flush final Memtable data to disk: %v", err)
		}

		fmt.Printf("Embeddings in %s stored at %s\n", sourceDir, destPath)

		return nil
	},
}

func embedContentAndStore(embedder embed.Embedder, content string, path string, store *storage.Store) error {
	tokenizer := bert.NewBertTokenize()

	chunks, err := tokenizer.SplitTokens(content, openaiMaxTokens)
	if err != nil {
		return err
	}

	for i, chunk := range chunks {
		decoded, err := tokenizer.Decode(chunk)
		if err != nil {
			return err
		}

		embeddings, err := embedder.Embed(decoded)
		if err != nil {
			return err
		}

		key := path + ":" + strconv.Itoa(i)
		err = store.Put(key, embeddings)
		if err != nil {
			return err
		}
	}

	return nil
}

func extractTextFromPDF(path string) (string, error) {
	fmt.Printf("reading from pdf: %s\n", path)
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)

	var builder strings.Builder
	b, err := r.GetPlainText()
	if err != nil {
		return "", err
	}
	_, err = io.Copy(&builder, b)
	if err != nil {
		return "", err
	}
	return builder.String(), nil
}
