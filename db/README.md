# Database Schema Management

## Migrate tool

The migrate tool provides an opinionated and thus simple way to manage the postgres schema.

- **Forward only migrations**. Down migrations are often untested and not recommended in production.
- **Sequence number versioning**. Timestamp versioning is quicker for development, but conflicts on version number 
helps ensure intended ordering after merging an old PR.
- **Sequence number interval of 10**. Allows up to 9 hotfix migrations if production and development databases have diverged.
- **Migrations must be synced with schema definition**. The DDL (and static DML) representation of the database must match
the applied migrations in source control. Having both representations in sync helps in understanding the difference
between commits.
- **SQL only migrations**. No extra work needed to support native functionality. Migrations are embedded and the binary
is copied to the docker image.


## Usage

<!--usage-shell-->
```
Usage: go run db/migrate.go <command>

Schema migration tool.

Flags:
  -h, --help    Show context-sensitive help.

Commands:
  new [<name>]
    Create a new migration file.

  up
    Run all the un-applied migrations.

  status
    Print the status of applied migrations.

  dump [<to>]
    Dump the existing schema to disk.

Run "go run db/migrate.go <command> --help" for more information on a command.
```
