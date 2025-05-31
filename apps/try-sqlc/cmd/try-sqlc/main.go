package main

import (
	"context"
	"database/sql"
	// must add the blank identifier "_" because we don't directly use any exported names from the package
	// but just want the side effects
	_ "embed"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	sqlcctx "github.com/ryuichi24/try-sqlc/internal/db/gen"
	gotilsql "github.com/ryuichi24/try-sqlc/internal/db/sql"
	"log"
	"reflect"
)

func run() error {
	ctx := context.Background()

	db, err := sql.Open("sqlite3", "file:test.db")
	if err != nil {
		return err
	}

	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatal(err)
	}

	// Set the embedded migrations filesystem for goose
	goose.SetBaseFS(gotilsql.MigrationsFS)

	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatal(err)
	}

	queries := sqlcctx.New(db)

	// list all authors
	authors, err := queries.ListAuthors(ctx)
	if err != nil {
		return err
	}
	log.Println(authors)

	// create an author
	insertedAuthor, err := queries.CreateAuthor(ctx, sqlcctx.CreateAuthorParams{
		Name: "Brian Kernighan",
		Bio:  sql.NullString{String: "Co-author of The C Programming Language and The Go Programming Language", Valid: true},
		Age:  sql.NullInt64{Int64: 80, Valid: true},
	})
	if err != nil {
		return err
	}
	log.Println(insertedAuthor)

	// get the author we just inserted
	fetchedAuthor, err := queries.GetAuthor(ctx, insertedAuthor.ID)
	if err != nil {
		return err
	}

	// prints true
	log.Println(reflect.DeepEqual(insertedAuthor, fetchedAuthor))
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}

}
