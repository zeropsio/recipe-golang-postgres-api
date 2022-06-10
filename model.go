package main

import (
	"context"

	_ "embed"

	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

//go:embed schema.sql
var migration string

type Todo struct {
	Id        int    `json:"id" db:"id"`
	Completed bool   `json:"completed" db:"completed"`
	Text      string `json:"text" db:"text"`
}

type UpdateTodo struct {
	Completed *bool  `json:"completed" db:"completed"`
	Text      string `json:"text" db:"text"`
}

type TodoRepository struct {
	conn *pgxpool.Pool
}

func (t TodoRepository) FindOne(ctx context.Context, id int) (Todo, bool, error) {
	var todo Todo
	err := pgxscan.Get(ctx, t.conn, &todo, `SELECT id, completed, text FROM todos WHERE id=$1`, id)
	if pgxscan.NotFound(err) {
		return todo, false, nil
	}
	if err != nil {
		return todo, false, err
	}
	return todo, true, nil
}

func (t TodoRepository) FindAll(ctx context.Context) ([]Todo, error) {
	var todos []Todo
	err := pgxscan.Select(ctx, t.conn, &todos, `SELECT id, completed, text FROM todos ORDER BY id DESC`)
	return todos, err
}

func (t TodoRepository) Create(ctx context.Context, todo Todo) (Todo, error) {
	var id int
	err := t.conn.QueryRow(ctx, "INSERT INTO todos(completed, text) VALUES ($1, $2) RETURNING id", todo.Completed, todo.Text).Scan(&id)
	if err != nil {
		return Todo{}, err
	}
	todo.Id = id
	return todo, nil
}

func (t TodoRepository) Edit(ctx context.Context, id int, updateTodo UpdateTodo) (Todo, error) {
	oldTodo, found, err := t.FindOne(ctx, id)
	if err != nil {
		return oldTodo, err
	}
	if !found {
		return oldTodo, pgx.ErrNoRows
	}
	if updateTodo.Completed != nil {
		oldTodo.Completed = *updateTodo.Completed
	}
	if updateTodo.Text != "" {
		oldTodo.Text = updateTodo.Text
	}
	_, err = t.conn.Exec(ctx, "UPDATE todos SET completed=$1, text=$2 WHERE id=$3", oldTodo.Completed, oldTodo.Text, id)
	return oldTodo, err
}

func (t TodoRepository) Delete(ctx context.Context, id int) error {
	_, err := t.conn.Exec(ctx, "DELETE FROM todos WHERE id=$1", id)
	return err
}

func (t TodoRepository) PrepareDatabase(ctx context.Context, dropTable bool, seeds []string) error {
	if dropTable {
		_, err := t.conn.Exec(ctx, "DROP TABLE IF EXISTS todos")
		if err != nil {
			return err
		}
	}
	_, err := t.conn.Exec(ctx, migration)

	if dropTable {
		for _, seed := range seeds {
			_, err := t.Create(ctx, Todo{Text: seed})
			if err != nil {
				return err
			}
		}
	}

	return err
}
