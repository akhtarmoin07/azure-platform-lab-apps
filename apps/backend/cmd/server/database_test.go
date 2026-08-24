package main

import (
	"strings"
	"testing"
)

func TestLoadDatabaseConfigurationForAzureWorkloadIdentity(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SQL_SERVER", "example.database.windows.net")
	t.Setenv("SQL_DATABASE", "pharmacy-dev")
	t.Setenv("SQL_FEDAUTH", "ActiveDirectoryWorkloadIdentity")

	configuration, err := loadDatabaseConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Driver != "azuresql" {
		t.Fatalf("expected azuresql driver, got %q", configuration.Driver)
	}
	for _, expected := range []string{
		"database=pharmacy-dev",
		"fedauth=ActiveDirectoryWorkloadIdentity",
		"encrypt=true",
		"TrustServerCertificate=false",
	} {
		if !strings.Contains(configuration.ConnectionString, expected) {
			t.Fatalf("expected connection string to contain %q, got %q", expected, configuration.ConnectionString)
		}
	}
}

func TestLoadDatabaseConfigurationForLocalSQLServer(t *testing.T) {
	t.Setenv("DATABASE_URL", "sqlserver://sa:password@database:1433?database=pharmacy")
	t.Setenv("DATABASE_DRIVER", "sqlserver")

	configuration, err := loadDatabaseConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.Driver != "sqlserver" {
		t.Fatalf("expected sqlserver driver, got %q", configuration.Driver)
	}
}
