package source

// A config object that is able to receive data in the form of string maps.
type Mapper interface {
	// Reindex the data stucture.
	UpdateIndices()
	// Set data from a map[string]interface{}.
	FromStringMap(data map[string]interface{})
	// Merge a map[string]interface{} into existing data.
	ToStringMap() map[string]interface{}
}
