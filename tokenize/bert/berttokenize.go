package bert

import (
	"fmt"
	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/decoder"
	"github.com/sugarme/tokenizer/pretrained"
)

type BertTokenize struct {
	tk *tokenizer.Tokenizer
}

func NewBertTokenize() *BertTokenize {
	tk := pretrained.BertBaseUncased()
	tk.WithDecoder(decoder.NewWordPieceDecoder("##", true))

	return &BertTokenize{
		tk: tk,
	}
}

func (bt *BertTokenize) encode(input string) ([]int, error) {
	encoding, err := bt.tk.EncodeSingle(input, true)

	if err != nil {
		return nil, err
	}

	return encoding.Ids, nil
}

func (bt *BertTokenize) SplitTokens(text string, size int) ([][]int, error) {
	encoded, err := bt.encode(text)

	if err != nil {
		return nil, fmt.Errorf("failed to encode input text: %w", err)
	}

	var chunks [][]int

	for i := 0; i < len(encoded); i += size {
		end := i + size
		if end > len(encoded) {
			end = len(encoded)
		}
		chunks = append(chunks, encoded[i:end])
	}

	return chunks, nil
}

func (bt *BertTokenize) Decode(chunk []int) (string, error) {
	return bt.tk.Decode(chunk, true), nil
}
