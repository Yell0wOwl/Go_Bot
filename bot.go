package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var (
	BotToken   = "5370976840:AAGGlkRGxNgxuHilxAKceC5OuEChxaeMSJ8"
	WebhookURL = "https://brightyellowapp.herokuapp.com"
)

type Task struct {
	Text        string
	Performer   string
	PerformerID int64
	Owner       string
	OwnerID     int64
	ID          int
}

var tasks []Task

var myTasks []Task

var newTask Task

var nextId int = 1

var thisId int

var needNewString bool

func startTaskBot(ctx context.Context) error {

	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Fatalf("NewBotAPI failed: %s", err)
	}

	bot.Debug = true

	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		log.Fatalf("NewWebhook failed: %s", err)
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Fatalf("SetWebhook failed: %s", err)
	}

	updates := bot.ListenForWebhook("/")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		log.Fatalln("http err:", http.ListenAndServe(":"+port, nil))
	}()

	for update := range updates {

		log.Printf("upd: %#v\n", update)

		if update.Message == nil {
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		command := update.Message.Text
		lenth := len(command)
		text := ""

		if command == "/tasks" {
			needNewString = false
			if tasks == nil {
				text = "Нет задач"
			} else {
				for _, tk := range tasks {
					if needNewString {
						text += "\n\n"
					}
					text += strconv.Itoa(tk.ID) + ". " + tk.Text + " by @" + tk.Owner + "\n"
					if tk.Performer != "" {
						text += "assignee: "
						if tk.Performer == update.Message.From.UserName {
							text += "я\n" + "/unassign_" + strconv.Itoa(tk.ID) + " /resolve_" + strconv.Itoa(tk.ID)
						} else {
							text += "@" + tk.Performer
						}
					} else {
						text += "/assign_" + strconv.Itoa(tk.ID)
					}
					needNewString = true
				}
			}
		}

		if lenth > 3 && command[:4] == "/new" {
			newTask = Task{}
			newTask.Text = command[5:]
			newTask.OwnerID = update.Message.Chat.ID
			newTask.Owner = update.Message.From.UserName
			newTask.ID = nextId
			newTask.Performer = ""
			nextId += 1
			tasks = append(tasks, newTask)
			text += `Задача "` + newTask.Text + `" создана, id=` + strconv.Itoa(newTask.ID)
		}

		if lenth > 8 && command[:8] == "/assign_" {
			thisId, err = strconv.Atoi(command[8:])
			if err != nil {
				text = "Error: incorrect id"
			} else {
				for i, tk := range tasks {
					if tk.ID == thisId {
						if tk.Performer != "" {
							bot.Send(tgbotapi.NewMessage(
								tk.PerformerID,
								`Задача "`+tk.Text+`" назначена на @`+update.Message.From.UserName,
							))
						} else {
							if tk.Owner != update.Message.From.UserName {
								bot.Send(tgbotapi.NewMessage(
									tk.OwnerID,
									`Задача "`+tk.Text+`" назначена на @`+update.Message.From.UserName,
								))
							}
						}
						text += `Задача "` + tk.Text + `" назначена на вас`
						tasks[i].PerformerID = update.Message.Chat.ID
						tasks[i].Performer = update.Message.From.UserName
					}
				}
			}
		}

		if lenth > 10 && command[:10] == "/unassign_" {
			thisId, err = strconv.Atoi(command[10:])
			if err != nil {
				text = "Error: incorrect id"
			} else {
				for i, tk := range tasks {
					if tk.ID == thisId {
						if tk.Performer == update.Message.From.UserName {
							text = `Принято`
							tasks[i].Performer = ""
							bot.Send(tgbotapi.NewMessage(
								tk.OwnerID,
								`Задача "`+tk.Text+`" осталась без исполнителя`,
							))
						} else {
							text = `Задача не на вас`
						}
					}
				}
			}
		}

		if lenth > 9 && command[:9] == "/resolve_" {
			thisId, err = strconv.Atoi(command[9:])
			if err != nil {
				text = "Error: incorrect id"
			} else {
				for i, tk := range tasks {
					if tk.ID == thisId {
						if tk.Performer == update.Message.From.UserName {
							text = `Задача "` + tk.Text + `" выполнена`
							bot.Send(tgbotapi.NewMessage(
								tk.OwnerID,
								`Задача "`+tk.Text+`" выполнена @`+tk.Performer,
							))
							if len(tasks) > 1 {
								tasks = append(tasks[:i], tasks[i+1:]...)
							} else {
								tasks = nil
							}
						} else {
							text = `Задача не на вас`
						}
					}
				}
			}
		}

		if command == "/my" {
			for _, tk := range tasks {
				if tk.Performer == update.Message.From.UserName {
					myTasks = append(myTasks, tk)
				}
			}
			needNewString = false
			if myTasks == nil {
				text += "Нет задач"
			} else {
				for _, tk := range myTasks {
					if needNewString {
						text += "\n\n"
					}
					text += strconv.Itoa(tk.ID) + ". " + tk.Text + " by @" + tk.Owner + "\n"
					if tk.Performer == tk.Owner {
						text += "/unassign_" + strconv.Itoa(tk.ID) + " /resolve_" + strconv.Itoa(tk.ID)
					} else {
						text += "/assign_" + strconv.Itoa(tk.ID)
					}
					needNewString = true
				}
			}
			myTasks = nil
		}

		if command == "/owner" {
			for _, tk := range tasks {
				if tk.Owner == update.Message.From.UserName {
					myTasks = append(myTasks, tk)
				}
			}
			needNewString = false
			if myTasks == nil {
				text = "Нет задач"
			} else {
				for _, tk := range myTasks {
					if needNewString {
						text += "\n\n"
					}
					text += strconv.Itoa(tk.ID) + ". " + tk.Text + " by @" + tk.Owner + "\n"
					if tk.Performer == tk.Owner {
						text += "/unassign_" + strconv.Itoa(tk.ID) + " /resolve_" + strconv.Itoa(tk.ID)
					} else {
						text += "/assign_" + strconv.Itoa(tk.ID)
					}
					needNewString = true
				}
			}
			myTasks = nil
		}
	}
	return nil
}

func main() {
	err := startTaskBot(context.Background())
	if err != nil {
		panic(err)
	}
}
