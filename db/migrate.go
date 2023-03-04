package main

import (
	"bufio"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/lib/pq"

	"github.com/adlio/schema"
	"github.com/alecthomas/kong"
	"github.com/kennygrant/sanitize"
	_ "github.com/lib/pq"
)

//go:embed migrations
var migrationsEmbed embed.FS
var migrationsParsed = must(schema.FSMigrations(migrationsEmbed, "migrations/*.sql"))
var pgConnector = must(pq.NewConnector(os.Getenv(databaseConnEnv)))

const (
	databaseConnEnv = "DATABASE_CONN"
	schemaDir       = "db/schema"
	tableDir        = "table"
	pgDumpBin       = "pg_dump"
)

var cli struct {
	New struct {
		Name string `arg:"" optional:"" help:"A name or short description for the migration file."`
	} `cmd:"" help:"Create a new migration file."`
	Up     struct{} `cmd:"" help:"Run all the un-applied migrations."`
	Status struct{} `cmd:"" help:"Print the status of applied migrations."`
	Dump   struct {
		To string `arg:"" optional:"" help:"Directory to write schema files to."`
	} `cmd:"" help:"Dump the existing schema to disk."`
}

func main() {
	ktx := kong.Parse(&cli, kong.Name("go run db/migrate.go"), kong.Description("Schema migration tool."))
	switch ktx.Command() {
	case "new", "new <name>":
		ktx.FatalIfErrorf(NewCmd(cli.New.Name))
	case "up":
		ktx.FatalIfErrorf(UpCmd())
	case "status":
		ktx.FatalIfErrorf(StatusCmd())
	case "dump", "dump <to>":
		ktx.FatalIfErrorf(DumpCmd(cli.Dump.To))
	}
}

// UpCmd runs all the embedded migrations from the migrations folder
func UpCmd() error {
	sqldb := sql.OpenDB(pgConnector)
	defer sqldb.Close()
	migrator := schema.NewMigrator(schema.WithDialect(schema.Postgres))
	return migrator.Apply(sqldb, migrationsParsed)
}

// NewCmd will create a new versioned migration file with the provided name or prompt
func NewCmd(name string) error {
	if name == "" {
		name = promptDescription()
	}

	fname, err := createMigrationFile("db/migrations", name)
	if err != nil {
		return err
	}

	fmt.Println("Created new migration file", fname)
	return nil
}

// StatusCmd will print the current migration status
func StatusCmd() error {
	sqldb := sql.OpenDB(pgConnector)
	defer sqldb.Close()
	migrator := schema.NewMigrator(schema.WithDialect(schema.Postgres))
	results, err := migrator.GetAppliedMigrations(sqldb)
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', tabwriter.AlignRight)
	fmt.Fprintln(writer, "Version\tApplied At")
	for _, v := range sortedKeys(results) {
		h := results[v]
		fmt.Fprintf(writer, "%s\t%v\n", h.Migration.ID, h.AppliedAt)
	}
	writer.Flush()

	return nil
}

// DumpCmd will dump the ddl to the provided directory
func DumpCmd(to string) error {
	if to == "" {
		to = schemaDir
	}

	sqldb := sql.OpenDB(pgConnector)
	tables, err := getTables(sqldb)
	if err != nil {
		return err
	}

	if err := clearSchemaDir(to); err != nil {
		return err
	}

	for _, v := range tables {
		if err := writeDDL(v, filepath.Join(to, tableDir)); err != nil {
			return err
		}
	}

	// TODO: static data

	return nil
}

func sortedKeys[M ~map[K]V, K comparable, V any](m M) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func promptDescription() string {
	fmt.Print("Please enter a short description for the migration file. Or press Enter to skip.\n>")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	clean := sanitize.Name(scanner.Text())
	if len(clean) == 1 && clean[0] == '.' { // When Enter is only pressed to skip
		return ""
	}
	return clean
}

func createMigrationFile(dir string, name string) (string, error) {
	if !regexp.MustCompile(`[a-zA-Z0-9_\-]+`).MatchString(name) {
		return "", errors.New("invalid migration file name")
	}

	v, err := latestEmbedVersion()
	if err != nil {
		return "", err
	}
	filename := filepath.Join(dir, fmt.Sprintf("%08d_%s.sql", v, name))

	fp, err := os.Create(filename)
	if err != nil {
		return "", err
	}

	return filename, fp.Close()
}

func latestEmbedVersion() (uint, error) {
	const sequenceInterval = 10
	schema.SortMigrations(migrationsParsed)
	v := uint(1)

	if len(migrationsParsed) > 0 {
		last := migrationsParsed[len(migrationsParsed)-1]
		lastV := strings.Split(last.ID, "_")[0]
		id, err := strconv.Atoi(lastV)
		if err != nil {
			return 0, err
		}
		v = roundNext(uint(id), sequenceInterval)
	}
	return v, nil
}

func roundNext(n, next uint) uint {
	return uint(math.Round(float64(n)/float64(next)))*next + next
}

var (
	objectComment = regexp.MustCompile(`--\n-- Name:.+\n--\n`)
	footerComment = regexp.MustCompile(`--\n-- PostgreSQL database dump complete\n--\n`)
)

func writeDDL(tableName, directory string) error {
	ddl, err := pgDumpTable(tableName)
	if err != nil {
		return err
	}

	file := filepath.Join(directory, tableName+".sql")
	if err := mkdir(directory); err != nil {
		return err
	}
	return ioutil.WriteFile(file, []byte(ddl), 0664)
}

func mkdir(parent string) error {
	_, err := os.Stat(parent)
	if os.IsNotExist(err) {
		err = os.MkdirAll(parent, 0700)
	} else if err != nil {
		return err
	}
	return nil
}

func pgDumpTable(tableName string) (string, error) {
	u, err := url.Parse(os.Getenv(databaseConnEnv))
	if err != nil {
		return "", err
	}

	args := []string{
		"-h", u.Hostname(),
		"-p", u.Port(),
		"-U", u.User.Username(),
		u.Path[1:],
		"--no-owner",
		"--schema-only",
		"--schema", "public", //TODO parameterise schema
		"--table", tableName,
	}

	cmd := exec.Command(pgDumpBin, args...)
	if password, b := u.User.Password(); b {
		cmd.Env = append(os.Environ(), "PGPASSWORD="+password)
	}

	var out, errOut strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		println(errOut.String())
		return "", err
	}

	body := out.String()
	body = footerComment.ReplaceAllString(body, "")
	firstObject := objectComment.FindStringIndex(body)
	body = body[firstObject[0]:]

	return body, nil
}

func getTables(sqldb *sql.DB) ([]string, error) {
	tables := make([]string, 0)

	s := "SELECT table_name FROM information_schema.tables WHERE table_schema = $1"
	rows, err := sqldb.Query(s, "public")
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			return tables, err
		}
		tables = append(tables, tableName)
	}

	return tables, err
}

func clearSchemaDir(dir string) error {
	tables := filepath.Join(dir, tableDir)

	if err := os.RemoveAll(tables); err != nil {
		return err
	}

	return nil
}

func must[T any](t T, err error) T {
	if err != nil {
		panic(err.Error())
	}
	return t
}
