package main

import (
	"fmt"
	"html/template"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/template/html/v2"
)

type Todo struct {
	ID          int64
	Title       string
	CompletedAt *time.Time
	CreatedAt   time.Time
}

type TodoRepo struct {
	todoLock sync.Mutex
	todos    []Todo

	nextID atomic.Int64
}

func (r *TodoRepo) Add(todo Todo) {
	todo.ID = r.nextID.Add(1)

	r.todoLock.Lock()
	r.todos = append(r.todos, todo)
	r.todoLock.Unlock()
}

func (r *TodoRepo) Toggle(todoID int64) (*Todo, error) {
	todo := r.Get(todoID)
	if todo == nil {
		return nil, fmt.Errorf("Unable to find todo with ID: %d", todoID)
	}

	if todo.CompletedAt == nil {
		now := time.Now()
		todo.CompletedAt = &now
	} else {
		todo.CompletedAt = nil
	}

	return nil, nil
}

func (r *TodoRepo) Delete(todoID int64) error {
	r.todoLock.Lock()
	defer r.todoLock.Unlock()

	var todoIndex = -1
	for idx, todo := range r.todos {
		if todo.ID == todoID {
			todoIndex = idx
			break
		}
	}

	if todoIndex == -1 {
		return fmt.Errorf("Unable to find todo with ID: %d", todoID)
	}

	r.todos = append(r.todos[:todoIndex], r.todos[todoIndex+1:]...)
	return nil
}

func (r *TodoRepo) Get(todoID int64) *Todo {
	for idx := range r.todos {
		todo := &r.todos[idx]
		if todo.ID == todoID {
			return todo
		}
	}

	return nil
}

func (r *TodoRepo) ListByStatus(done bool) []Todo {
	var res []Todo

	for idx := range r.todos {
		if done && r.todos[idx].CompletedAt != nil || !done && r.todos[idx].CompletedAt == nil {
			res = append(res, r.todos[idx])
		}
	}

	// sort todos by CompletedAt if available on both items,
	// else fall back to comparing based on CreatedAt
	sort.Slice(res, func(i, j int) bool {
		todoA := res[i]
		todoB := res[j]

		if todoA.CompletedAt != nil && todoB.CompletedAt != nil {
			return todoB.CompletedAt.Before(*todoA.CompletedAt)
		}

		return todoB.CreatedAt.Before(todoA.CreatedAt)
	})

	return res
}

func NewTodo(title string, done bool) Todo {
	var completedAt *time.Time
	now := time.Now()

	if done {
		completedAt = &now
	}

	return Todo{
		Title:       title,
		CompletedAt: completedAt,
		CreatedAt:   now,
	}
}

var todoRepo TodoRepo

func main() {
	engine := html.New("./views", ".html")
	todoRepo = TodoRepo{}
	todoRepo.Add(NewTodo("first todo", false))
	todoRepo.Add(NewTodo("second todo", false))
	todoRepo.Add(NewTodo("third todo", false))
	todoRepo.Add(NewTodo("fourth todo", true))
	todoRepo.Add(NewTodo("fifth todo", true))

	engine.AddFunc("len", func(s []Todo) template.HTML {
		return template.HTML(fmt.Sprint(len(s)))
	})

	engine.AddFunc("hasItems", func(s []Todo) bool {
		return len(s) > 0
	})

	engine.Reload(true)
	app := fiber.New(fiber.Config{
		Views:     engine,
		Immutable: true,
	})

	app.Get("/", handleGetIndex)
	app.Get("/todos", handleGetTodoList)
	app.Put("/todos/:todo_id/toggle", handleToggleTodo)
	app.Delete("/todos/:todo_id", handleDeleteTodo)
	app.Post("/todos", handlePostTodo)

	log.Fatal(app.Listen(":3000"))
}

func handleGetIndex(c *fiber.Ctx) error {
	locals := fiber.Map{
		"todos_open": todoRepo.ListByStatus(false),
		"todos_done": todoRepo.ListByStatus(true),
	}

	return c.Render("index", locals, "layouts/main")
}

func handleToggleTodo(c *fiber.Ctx) error {
	todoId, err := c.ParamsInt("todo_id")
	if err != nil {
		log.Info("unable to parse todo_id", err)
		return c.SendStatus(404)
	}

	_, err = todoRepo.Toggle(int64(todoId))
	if err != nil {
		log.Info("unable to find todo by id", err)
		return c.SendStatus(404)
	}

	return handleGetTodoList(c)
}

func handleDeleteTodo(c *fiber.Ctx) error {
	todoId, err := c.ParamsInt("todo_id")
	if err != nil {
		log.Info("unable to parse todo_id", err)
		return c.SendStatus(404)
	}

	err = todoRepo.Delete(int64(todoId))
	if err != nil {
		log.Info("unable to find todo by id", err)
		return c.SendStatus(404)
	}

	return handleGetTodoList(c)
}

func handlePostTodo(c *fiber.Ctx) error {
	title := c.FormValue("todo", "unknown")
	todoRepo.Add(NewTodo(title, false))

	return handleGetTodoList(c)
}

func handleGetTodoList(c *fiber.Ctx) error {
	locals := fiber.Map{
		"todos_open": todoRepo.ListByStatus(false),
		"todos_done": todoRepo.ListByStatus(true),
	}

	return c.Render("partials/todo-list", locals)
}
