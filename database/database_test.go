package database

import "testing"

const demo_query = "SELECT * FROM test WHERE id = ? AND test = ? LIMIT 1"

func TestGetParamQuery(t *testing.T) {
	q := GetParamQuery("postgres", demo_query)
	if q != "SELECT * FROM test WHERE id = $1 AND test = $2 LIMIT 1" {
		t.Fatalf("Incorrect postgres query")
	}

	q = GetParamQuery("sqlite3", demo_query)
	if q != "SELECT * FROM test WHERE id = ? AND test = ? LIMIT 1" {
		t.Fatalf("Incorrect sqlite3 query")
	}
}
