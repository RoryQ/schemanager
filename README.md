# Schemanager
### PostgreSQL Schema Management Tools

This repo demonstrates some opinionated PostgreSQL schema management functionality and can be used as a project template 
for a golang application that uses postgres as the data store.

### Design considerations

- **Forward only migrations**. Down migrations are often untested and not recommended in production.
- **Sequence number versioning**. Timestamp versioning avoids merge conflicts between team members, but conflicts on 
  version number helps ensure intended ordering after merging an old PR.
- **Sequence number interval of 10**. Allows up to 9 hotfix migrations if production and development databases have diverged.
- **Migrations must be synced with schema definition**. The DDL (and static DML) representation of the database must match
  the applied migrations in source control. Having both representations in sync helps in understanding the difference
  between commits. (requires `pg_dump`)
- **SQL only migrations**. No extra work needed to support native functionality. Migrations are embedded and the binary
  is copied to the docker image.

### Recommended workflow

#### 1. `make migrate-new` to create a new migration file

Run the `make migrate-new` target to generate a versioned migration file for you to edit. The full filename is the
version number so the descriptive name should not be changed or else the migration will be re-run.

#### 2. Test changes locally

Test the schema changes, preferably against a non-persisted instance of the database until the schema is right. You can
recreate and rerun the migrations freely this way.

#### 3. `make schema` to save schema

When ready to open a PR, run `make schema` to export the final schema to disk and commit that along with the migrations.
The `Schema Sync Check / migrations-equal-schema` workflow runs against the PR to verify this has been done. If squash
merges only are used, then the migrations and schema show in the same commit and the schema definition files can be
`git blame`ed to identify when and what migration introduced a particular change.
