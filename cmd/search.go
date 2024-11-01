package cmd

import (
	"fmt"
	"github.com/aakashshankar/vexdb/embed/openai"
	"github.com/aakashshankar/vexdb/search"
	"github.com/aakashshankar/vexdb/storage"
	"github.com/spf13/cobra"
	"path/filepath"
	"sort"
	"strconv"
)

var searchCmd = &cobra.Command{
	Use:   "search [embeddings file] [query] [topk]",
	Short: "Perform a semantic search based in the query",
	Long:  `This command performs a semantic search based on the query.`,
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		embeddingsDir := args[0]
		query := args[1]
		k := 1
		if len(args) == 3 {
			var err error
			k, err = strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid value for k = %v: %w", args[2], err)
			}
		}

		store := storage.NewStore(1000, embeddingsDir)
		embedder, err := openai.NewOpenAIEmbedder()
		if err != nil {
			return fmt.Errorf("error initializing embedder: %w", err)
		}

		queryEmbedding, err := embedder.Embed(query)
		if err != nil {
			return fmt.Errorf("error generating embeddings: %w", err)
		}

		results, err := searchVectorStore(store, queryEmbedding, embeddingsDir, k)
		if err != nil {
			return fmt.Errorf("error searching vector database: %w", err)
		}
		for _, result := range results {
			fmt.Printf("%s -> %f\n", result.Key, result.Score)
		}

		return nil
	},
}

type SearchResult struct {
	Key   string
	Score float64
}

func searchVectorStore(store *storage.Store, queryEmbeddings []float64, destDir string, k int) ([]SearchResult, error) {
	results := make([]SearchResult, 0, k)

	files, err := filepath.Glob(filepath.Join(destDir, "*.sst"))
	if err != nil {
		return nil, fmt.Errorf("could not list SSTables in %s: %v", destDir, err)
	}

	for _, file := range files {
		sstable, err := storage.OpenSSTable(file)
		if err != nil {
			return nil, fmt.Errorf("could not open stable %s: %v", file, err)
		}

		for _, key := range sstable.Index {
			vector, exists, err := sstable.Get(key)
			if err != nil {
				return nil, fmt.Errorf("could not fetch key %s from sstable: %v", key, err)
			}

			if exists {
				score := search.Cosine(queryEmbeddings, vector)
				results = append(results, SearchResult{
					Key:   key,
					Score: score,
				})
			}
		}
	}

	for key, value := range store.Memtable.Data {
		vector, err := storage.DeserializeVector(value)
		if err != nil {
			return nil, fmt.Errorf("error deserializing vector for key %s: %w", key, err)
		}
		score := search.Cosine(queryEmbeddings, vector)
		results = append(results, SearchResult{
			Key:   key,
			Score: score,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > k {
		results = results[:k]
	}

	return results, nil
}
