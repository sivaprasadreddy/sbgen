package springboot

type DbMigrationsConfig struct {
	pg projectGenerator
}

func NewDbMigrationsConfig(pg projectGenerator) *DbMigrationsConfig {
	return &DbMigrationsConfig{pg: pg}
}

func (d DbMigrationsConfig) generate(pc ProjectConfig) error {
	return d.createDbMigrationFiles(pc)
}

func (d DbMigrationsConfig) createDbMigrationFiles(pc ProjectConfig) error {
	templateMap := map[string]string{}

	if pc.DbMigrationTool == Flyway {
		switch pc.DbType {
		case PostgreSQL:
			templateMap["db/migration/flyway/V001__create_users_table_postgresql.sql"] = "db/migration/V001__create_users_table.sql"
		case MySQL:
			templateMap["db/migration/flyway/V001__create_users_table_mysql.sql"] = "db/migration/V001__create_users_table.sql"
		case MariaDB:
			templateMap["db/migration/flyway/V001__create_users_table_mariadb.sql"] = "db/migration/V001__create_users_table.sql"
		case H2:
			templateMap["db/migration/flyway/V001__create_users_table_h2.sql"] = "db/migration/V001__create_users_table.sql"
		}
	}

	if pc.DbMigrationTool == Liquibase {
		templateMap["db/migration/liquibase/liquibase-changelog.xml"] = "db/migration/liquibase-changelog.xml"

		switch pc.DbType {
		case PostgreSQL:
			templateMap["db/migration/liquibase/changelog/001-create_users_table-postgresql.xml"] = "db/migration/changelog/001-create_users_table.xml"
		case MySQL:
			templateMap["db/migration/liquibase/changelog/001-create_users_table-mysql.xml"] = "db/migration/changelog/001-create_users_table.xml"
		case MariaDB:
			templateMap["db/migration/liquibase/changelog/001-create_users_table-mariadb.xml"] = "db/migration/changelog/001-create_users_table.xml"
		case H2:
			templateMap["db/migration/liquibase/changelog/001-create_users_table-h2.xml"] = "db/migration/changelog/001-create_users_table.xml"

		}
	}

	for tmpl, filePath := range templateMap {
		err := d.pg.executeTemplate(pc, srcMainResourcesPath+tmpl, srcMainResourcesPath+filePath)
		if err != nil {
			return err
		}
	}
	return nil
}
