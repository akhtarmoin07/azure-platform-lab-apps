package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"

	_ "github.com/microsoft/go-mssqldb/azuread"
)

type databaseConfiguration struct {
	Driver           string
	ConnectionString string
}

func loadDatabaseConfiguration() (databaseConfiguration, error) {
	if connectionString := os.Getenv("DATABASE_URL"); connectionString != "" {
		return databaseConfiguration{
			Driver:           environmentOrDefault("DATABASE_DRIVER", "sqlserver"),
			ConnectionString: connectionString,
		}, nil
	}

	server := os.Getenv("SQL_SERVER")
	if server == "" {
		return databaseConfiguration{}, fmt.Errorf("SQL_SERVER is required when DATABASE_URL is not set")
	}
	database := os.Getenv("SQL_DATABASE")
	if database == "" {
		return databaseConfiguration{}, fmt.Errorf("SQL_DATABASE is required when DATABASE_URL is not set")
	}

	connectionURL := &url.URL{Scheme: "sqlserver", Host: server}
	query := connectionURL.Query()
	query.Set("database", database)
	query.Set("fedauth", environmentOrDefault("SQL_FEDAUTH", "ActiveDirectoryWorkloadIdentity"))
	query.Set("encrypt", "true")
	query.Set("TrustServerCertificate", "false")
	connectionURL.RawQuery = query.Encode()

	return databaseConfiguration{
		Driver:           "azuresql",
		ConnectionString: connectionURL.String(),
	}, nil
}

func migrateDatabase(ctx context.Context, database *sql.DB) error {
	const schema = `
		IF OBJECT_ID(N'dbo.medicines', N'U') IS NULL
		BEGIN
			CREATE TABLE dbo.medicines (
				id INT IDENTITY(1,1) NOT NULL CONSTRAINT PK_medicines PRIMARY KEY,
				name NVARCHAR(200) NOT NULL CONSTRAINT UQ_medicines_name UNIQUE,
				description NVARCHAR(500) NOT NULL,
				price DECIMAL(10,2) NOT NULL CONSTRAINT CK_medicines_price CHECK (price >= 0)
			);
		END;

		MERGE dbo.medicines AS target
		USING (VALUES
			(N'Vitamin D3', N'Daily vitamin D supplement', CAST(8.99 AS DECIMAL(10,2))),
			(N'Saline Nasal Spray', N'Gentle isotonic nasal spray', CAST(5.49 AS DECIMAL(10,2))),
			(N'Digital Thermometer', N'Fast-reading digital thermometer', CAST(12.90 AS DECIMAL(10,2)))
		) AS source (name, description, price)
		ON target.name = source.name
		WHEN NOT MATCHED BY TARGET THEN
			INSERT (name, description, price)
			VALUES (source.name, source.description, source.price);
	`

	if _, err := database.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply Azure SQL schema: %w", err)
	}
	return nil
}
