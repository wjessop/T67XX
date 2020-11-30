package t67xx

/*
fmt.Println(Bitmask(0x6).IsSet(0x2))
fmt.Println(Bitmask(f.FileHeader.Characteristics).ListDescriptions(charValues))
fmt.Println(Bitmask(f.FileHeader.Characteristics).ListValues(charValues))
*/

// Bitmask represents a bitmask value
type Bitmask uint16

// BitValue is a value and a description
type BitValue struct {
	value       Bitmask
	description string
}

// ListDescriptions returns a string array of all set bits
func (value Bitmask) ListDescriptions(values []BitValue) []string {
	list := make([]string, 0)
	currentValue := value
	for _, bv := range values {
		if currentValue&bv.value != 0 {
			currentValue ^= bv.value
			list = append(list, bv.description)
		}
	}
	return list
}

// ListValues returns a string array of all set bits
func (value Bitmask) ListValues(values []BitValue) []Bitmask {
	list := make([]Bitmask, 0)
	currentValue := value
	for _, bv := range values {
		if currentValue&bv.value != 0 {
			currentValue ^= bv.value
			list = append(list, bv.value)
		}
	}

	return list
}

// IsSet returns true if a bit is set
func (value Bitmask) IsSet(test Bitmask) bool {
	if value&test != 0 {
		return true
	}
	return false
}
