package encoding

import "fmt"

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const base = 62

// Encode returns the Base62 form of id. Encode(0) is "0".
func Encode(id uint64) string {
	if id == 0 {
		return "0"
	}
	var bytes []byte
	for id > 0 {
		bytes = append(bytes, alphabet[id%base])
		id /= base
	}
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	return string(bytes)
}

// Decode reverses Encode. It rejects a blank code, a character outside the alphabet, or a value above uint64.
func Decode(shortCode string) (uint64, error) {
	if shortCode == "" {
		return 0, fmt.Errorf("shortCode must not be blank")
	}
	var total uint64
	for _, ch := range shortCode {
		index := indexInAlphabet(ch)
		if index < 0 {
			return 0, fmt.Errorf("invalid Base62 character: %c", ch)
		}
		next, ok := mulAdd(total, base, uint64(index))
		if !ok {
			return 0, fmt.Errorf("Base62 value overflows u64")
		}
		total = next
	}
	return total, nil
}

func indexInAlphabet(ch rune) int {
	for i, candidate := range alphabet {
		if candidate == ch {
			return i
		}
	}
	return -1
}

func mulAdd(total uint64, multiplier int, add uint64) (uint64, bool) {
	if total > ^uint64(0)/uint64(multiplier) {
		return 0, false
	}
	product := total * uint64(multiplier)
	if product > ^uint64(0)-add {
		return 0, false
	}
	return product + add, true
}
