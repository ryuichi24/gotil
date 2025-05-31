-- +goose Up

-- +goose StatementBegin
ALTER TABLE authors ADD COLUMN age INTEGER;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE authors DROP COLUMN age;
-- +goose StatementEnd
