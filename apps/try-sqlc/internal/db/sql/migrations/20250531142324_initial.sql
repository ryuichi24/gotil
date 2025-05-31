-- +goose Up
CREATE TABLE authors (
  id   INTEGER PRIMARY KEY,
  name text    NOT NULL,
  bio  text
);
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose Down
DROP TABLE authors;
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
