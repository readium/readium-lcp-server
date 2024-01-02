package dbutils

import (
	"bytes"
	"fmt"
	"strings"
)

func getPostgresQuery(query string) string {
	var buffer bytes.Buffer
	idx := 1
	for _, char := range query {
		if char == '?' {
			buffer.WriteString(fmt.Sprintf("$%d", idx))
			idx += 1
		} else {
			buffer.WriteRune(char)
		}
	}
	return buffer.String()
}

// GetParamQuery replaces parameter placeholders '?' in the SQL query to
// placeholders supported by the selected database driver.
func GetParamQuery(database, query string) string {
	if strings.HasPrefix(database, "postgres") {
		return getPostgresQuery(query)
	} else {
		return query
	}
}
