package main

import (
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/template/html/v2"
)

type Todo struct {
	ID     int
	Title  string
	DoneAt *time.Time
}

func NewTodo(title string, done bool) Todo {
	var doneAt *time.Time
	id := todoId
	todoId++

	if done {
		now := time.Now()
		doneAt = &now
	}

	return Todo{ID: id, Title: title, DoneAt: doneAt}
}

func todoById(id int) (*Todo, error) {
	for idx := range todos {
		todo := &todos[idx]
		if todo.ID == id {
			return todo, nil
		}
	}

	return nil, fmt.Errorf("Unable to find todo with id: %d", id)
}

func todosByDone(done bool) []*Todo {
	var res []*Todo

	for idx := range todos {
		if done && todos[idx].DoneAt != nil || !done && todos[idx].DoneAt == nil {
			res = append(res, &todos[idx])
		}
	}

	sort.Slice(res, func(i, j int) bool {
		todoA := res[i]
		todoB := res[j]

		if todoA.DoneAt == nil || todoB.DoneAt == nil {
			return false
		}
		return todoB.DoneAt.Before(*todoA.DoneAt)
	})

	return res
}

var (
	todoId = 1
	todos  = []Todo{
		NewTodo("first todo", false),
		NewTodo("second todo", false),
		NewTodo("third todo", false),
		NewTodo("fourth todo", true),
		NewTodo("fifth todo", true),
	}
)

func main() {
	engine := html.New("./views", ".html")

	engine.AddFunc("len", func(s []*Todo) template.HTML {
		return template.HTML(fmt.Sprint(len(s)))
	})

	engine.AddFunc("hasItems", func(s []*Todo) bool {
		return len(s) > 0
	})

	engine.Reload(true)
	app := fiber.New(fiber.Config{
		Views: engine,

		// Let's reel it in with the performance and have a sane default here
		Immutable: true,
	})

	app.Get("/", func(c *fiber.Ctx) error {
		locals := fiber.Map{
			"todos_open": todosByDone(false),
			"todos_done": todosByDone(true),
		}

		return c.Render("index", locals, "layouts/main")
	})

	app.Get("/todos", func(c *fiber.Ctx) error {
		locals := fiber.Map{
			"todos_open": todosByDone(false),
			"todos_done": todosByDone(true),
		}

		return c.Render("index", locals)
	})

	app.Put("/todos/:todo_id/toggle", func(c *fiber.Ctx) error {
		todoId, err := c.ParamsInt("todo_id")
		if err != nil {
			log.Info("unable to parse todo_id", err)
			return c.SendStatus(404)
		}

		todo, err := todoById(todoId)
		if err != nil {
			log.Info("unable to find todo by id", err)
			return c.SendStatus(404)
		}

		if todo.DoneAt == nil {
			now := time.Now()
			todo.DoneAt = &now
		} else {
			todo.DoneAt = nil
		}

		locals := fiber.Map{
			"todos_open": todosByDone(false),
			"todos_done": todosByDone(true),
		}

		return c.Render("index", locals)
	})

	app.Post("/todos", func(c *fiber.Ctx) error {
		newTodo := c.FormValue("todo", "unknown")
		todos = append(todos, NewTodo(newTodo, false))

		locals := fiber.Map{
			"todos_open": todosByDone(false),
			"todos_done": todosByDone(true),
		}

		return c.Render("index", locals)
	})

	app.Listen(":3000")
}
