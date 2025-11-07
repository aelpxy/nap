package app

import (
	"fmt"

	"github.com/aelpxy/yap/pkg/models"
)

func LinkDatabase(app *models.Application, db *models.Database) error {
	if app.VPC != db.VPC {
		return fmt.Errorf("app and database must be in the same vpc (app: %s, db: %s)", app.VPC, db.VPC)
	}

	for _, linkedDB := range app.LinkedDatabases {
		if linkedDB == db.Name {
			return fmt.Errorf("database '%s' is already linked to app '%s'", db.Name, app.Name)
		}
	}

	if app.EnvVars == nil {
		app.EnvVars = make(map[string]string)
	}

	switch db.Type {
	case "postgres":
		InjectPostgresEnvVars(app, db)
	case "valkey":
		InjectValkeyEnvVars(app, db)
	default:
		return fmt.Errorf("unsupported database type: %s", db.Type)
	}

	app.LinkedDatabases = append(app.LinkedDatabases, db.Name)

	return nil
}

func UnlinkDatabase(app *models.Application, db *models.Database) error {
	found := false
	for i, linkedDB := range app.LinkedDatabases {
		if linkedDB == db.Name {
			app.LinkedDatabases = append(app.LinkedDatabases[:i], app.LinkedDatabases[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("database '%s' is not linked to app '%s'", db.Name, app.Name)
	}

	switch db.Type {
	case "postgres":
		RemovePostgresEnvVars(app)
	case "valkey":
		RemoveValkeyEnvVars(app)
	}

	return nil
}

func InjectPostgresEnvVars(app *models.Application, db *models.Database) {
	hostname := db.ContainerName

	app.EnvVars["DATABASE_URL"] = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		db.Username, db.Password, hostname, db.InternalPort, db.DatabaseName)
	app.EnvVars["POSTGRES_HOST"] = hostname
	app.EnvVars["POSTGRES_PORT"] = fmt.Sprintf("%d", db.InternalPort)
	app.EnvVars["POSTGRES_USER"] = db.Username
	app.EnvVars["POSTGRES_PASSWORD"] = db.Password
	app.EnvVars["POSTGRES_DATABASE"] = db.DatabaseName
}

func RemovePostgresEnvVars(app *models.Application) {
	delete(app.EnvVars, "DATABASE_URL")
	delete(app.EnvVars, "POSTGRES_HOST")
	delete(app.EnvVars, "POSTGRES_PORT")
	delete(app.EnvVars, "POSTGRES_USER")
	delete(app.EnvVars, "POSTGRES_PASSWORD")
	delete(app.EnvVars, "POSTGRES_DATABASE")
}

func InjectValkeyEnvVars(app *models.Application, db *models.Database) {
	hostname := db.ContainerName

	app.EnvVars["REDIS_URL"] = fmt.Sprintf("redis://:%s@%s:%d",
		db.Password, hostname, db.InternalPort)
	app.EnvVars["VALKEY_URL"] = fmt.Sprintf("redis://:%s@%s:%d",
		db.Password, hostname, db.InternalPort)
	app.EnvVars["VALKEY_HOST"] = hostname
	app.EnvVars["VALKEY_PORT"] = fmt.Sprintf("%d", db.InternalPort)
	app.EnvVars["VALKEY_PASSWORD"] = db.Password
}

func RemoveValkeyEnvVars(app *models.Application) {
	delete(app.EnvVars, "REDIS_URL")
	delete(app.EnvVars, "VALKEY_URL")
	delete(app.EnvVars, "VALKEY_HOST")
	delete(app.EnvVars, "VALKEY_PORT")
	delete(app.EnvVars, "VALKEY_PASSWORD")
}
