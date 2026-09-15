package database

import "testing"

func TestMySQLConnectionConfigSetsClientAndSessionTimezone(t *testing.T) {
	configuration, err := mysqlConnectionConfig("user:pass@tcp(localhost:3306)/old", "application")
	if err != nil {
		t.Fatal(err)
	}
	if !configuration.ParseTime || configuration.Loc.String() != "Asia/Shanghai" || configuration.Params["time_zone"] != "'+08:00'" || configuration.DBName != "application" {
		t.Fatalf("mysql config = %+v", configuration)
	}
}
