package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	"reflect"

	sqlcctx "github.com/ryuichi24/try-sqlc/internal/db/gen"
	_ "modernc.org/sqlite"
)


func run() error {
	ctx := context.Background()

	db, err := sql.Open("sqlite", "file:test.db")
	if err != nil {
		return err
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
		Age: sql.NullInt64{Int64: 80, Valid: true},
		
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
