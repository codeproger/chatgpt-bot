package main

// bot.go

import (
	"fmt"
	"log"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

const (
	cmdStart      = "/start"
	cmdReset      = "/reset"
	cmdModel      = "/model"
	cmdTemp       = "/temperature"
	cmdPrompt     = "/prompt"
	cmdAge        = "/age"
	cmdPromptCL   = "/defaultprompt"
	cmdStream     = "/stream"
	cmdStop       = "/stop"
	cmdInfo       = "/info"
	cmdToJapanese = "/ja"
	cmdToEnglish  = "/en"
	cmdToRussian  = "/ru"
	cmdUsers      = "/users"
	cmdAddUser    = "/add"
	cmdDelUser    = "/del"
	cmdHelp       = "/help"
	msgStart      = "This bot will answer your messages with ChatGPT API"
	msgReset      = "This bots memory erased"
	masterPrompt  = "You are a helpful assistant. You always try to answer truthfully. If you don't know the answer, just say that you don't know, don't try to make up an answer. Don't explain yourself. Do not introduce yourself, just answer the user concisely."
)

var (
	menu   = &tele.ReplyMarkup{ResizeKeyboard: true}
	btn3   = tele.Btn{Text: "GPT3", Unique: "btnModel", Data: "gpt-3.5-turbo"}
	btn4   = tele.Btn{Text: "GPT4", Unique: "btnModel", Data: "gpt-4"}
	btn316 = tele.Btn{Text: "GPT3-16k", Unique: "btnModel", Data: "gpt-3.5-turbo-16k"}
	btnT0  = tele.Btn{Text: "0.0", Unique: "btntemp", Data: "0.0"}
	btnT2  = tele.Btn{Text: "0.2", Unique: "btntemp", Data: "0.2"}
	btnT4  = tele.Btn{Text: "0.4", Unique: "btntemp", Data: "0.4"}
	btnT6  = tele.Btn{Text: "0.6", Unique: "btntemp", Data: "0.6"}
	btnT8  = tele.Btn{Text: "0.8", Unique: "btntemp", Data: "0.8"}
	btnT10 = tele.Btn{Text: "1.0", Unique: "btntemp", Data: "1.0"}
)

// launch bot with given parameters
func (s Server) run() {
	pref := tele.Settings{
		Token:  s.conf.TelegramBotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := tele.NewBot(pref)
	if err != nil {
		log.Fatal(err)
		return
	}
	//b.Use(middleware.Logger())
	b.Use(s.whitelist())
	s.bot = b

	usage, err := s.getUsageMonth()
	if err != nil {
		log.Println(err)
	}
	log.Printf("Current usage: %0.2f", usage)

	b.Handle(cmdStart, func(c tele.Context) error {
		return c.Send(msgStart, "text", &tele.SendOptions{ReplyTo: c.Message()})
	})

	b.Handle(cmdModel, func(c tele.Context) error {
		menu.Inline(menu.Row(btn3, btn4, btn316))

		return c.Send("Select model", menu)
	})

	b.Handle(cmdTemp, func(c tele.Context) error {
		menu.Inline(menu.Row(btnT0, btnT2, btnT4, btnT6, btnT8, btnT10))
		chat := s.getChat(c.Chat().ID, c.Sender().Username)

		return c.Send(fmt.Sprintf("Set temperature from less random (0.0) to more random (1.0.\nCurrent: %0.2f (default: 0.8)", chat.Temperature), menu)
	})

	b.Handle(cmdPrompt, func(c tele.Context) error {
		query := c.Message().Payload
		if len(query) < 3 {
			return c.Send("Please provide a longer prompt", "text", &tele.SendOptions{
				ReplyTo: c.Message(),
			})
		}

		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		chat.MasterPrompt = query
		s.db.Save(&chat)

		return nil
	})

	b.Handle(cmdAge, func(c tele.Context) error {
		age, err := strconv.Atoi(c.Message().Payload)
		if err != nil {
			return c.Send("Please provide a number", "text", &tele.SendOptions{
				ReplyTo: c.Message(),
			})
		}
		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		chat.ConversationAge = int64(age)
		s.db.Save(&chat)

		return c.Send(fmt.Sprintf("Conversation age set to %d days", age), "text", &tele.SendOptions{ReplyTo: c.Message()})
	})

	b.Handle(cmdPromptCL, func(c tele.Context) error {
		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		chat.MasterPrompt = masterPrompt
		s.db.Save(&chat)

		return c.Send("Default prompt set", "text", &tele.SendOptions{ReplyTo: c.Message()})
	})

	b.Handle(cmdStream, func(c tele.Context) error {
		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		chat.Stream = !chat.Stream
		s.db.Save(&chat)
		status := "disabled"
		if chat.Stream {
			status = "enabled"
		}

		return c.Send("Stream is "+status, "text", &tele.SendOptions{ReplyTo: c.Message()})
	})

	b.Handle(cmdStop, func(c tele.Context) error {

		return nil
	})

	b.Handle(cmdInfo, func(c tele.Context) error {
		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		status := "disabled"
		if chat.Stream {
			status = "enabled"
		}

		//usage, err := s.getUsageMonth()
		//if err != nil {
		//	log.Println(err)
		//}
		//log.Printf("Current usage: %0.2f", usage)

		return c.Send(fmt.Sprintf("Model: %s\nTemperature: %0.2f\nPrompt: %s\nStreaming: %s\nConvesation Age (days): %d",
			chat.ModelName, chat.Temperature, chat.MasterPrompt, status, chat.ConversationAge,
		),
			"text",
			&tele.SendOptions{ReplyTo: c.Message()},
		)
	})

	b.Handle(cmdToJapanese, func(c tele.Context) error {
		go s.onTranslate(c, "To Japanese: ")

		return nil
	})

	b.Handle(cmdToEnglish, func(c tele.Context) error {
		go s.onTranslate(c, "To English: ")

		return nil
	})

	b.Handle(cmdToRussian, func(c tele.Context) error {
		go s.onTranslate(c, "To Russian: ")

		return nil
	})

	b.Handle(&btn3, func(c tele.Context) error {
		log.Printf("%s selected", c.Data())
		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		chat.ModelName = c.Data()
		s.db.Save(&chat)

		return c.Edit("Model set to " + c.Data())
	})

	// On inline button pressed (callback)
	b.Handle(&btn316, func(c tele.Context) error {
		log.Printf("%s selected", c.Data())
		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		chat.ModelName = c.Data()
		s.db.Save(&chat)

		return c.Edit("Model set to " + c.Data())
	})

	// On inline button pressed (callback)
	b.Handle(&btnT0, func(c tele.Context) error {
		log.Printf("Temp: %s\n", c.Data())
		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		chat.Temperature, _ = strconv.ParseFloat(c.Data(), 64)
		s.db.Save(&chat)

		return c.Edit("Temperature set to " + c.Data())
	})

	b.Handle(cmdReset, func(c tele.Context) error {
		chat := s.getChat(c.Chat().ID, c.Sender().Username)
		s.deleteHistory(chat.ID)

		return nil //c.Send(msgReset, "text", &tele.SendOptions{ReplyTo: c.Message()})
	})

	b.Handle(tele.OnText, func(c tele.Context) error {
		go s.onText(c)

		return nil
	})

	b.Handle(tele.OnQuery, func(c tele.Context) error {
		query := c.Query().Text
		go s.complete(c, query, false)

		return nil
	})

	b.Handle(tele.OnDocument, func(c tele.Context) error {
		go s.onDocument(c)

		return nil
	})

	b.Handle(tele.OnPhoto, func(c tele.Context) error {
		log.Printf("Got a photo, size %d, caption: %s\n", c.Message().Photo.FileSize, c.Message().Photo.Caption)

		return nil
	})

	b.Handle(tele.OnVoice, func(c tele.Context) error {
		go s.onVoice(c)

		return nil
	})

	b.Handle(cmdUsers, func(c tele.Context) error {
		if !in_array(c.Sender().Username, s.conf.AllowedTelegramUsers) {
			return nil
		}
		return s.onGetUsers(c)
	})

	b.Handle(cmdAddUser, func(c tele.Context) error {
		if !in_array(c.Sender().Username, s.conf.AllowedTelegramUsers) {
			return nil
		}
		name := c.Message().Payload
		if len(name) < 3 {
			return c.Send("Username is too short", "text", &tele.SendOptions{
				ReplyTo: c.Message(),
			})
		}
		s.addUser(name)

		return s.onGetUsers(c)
	})

	b.Handle(cmdDelUser, func(c tele.Context) error {
		if !in_array(c.Sender().Username, s.conf.AllowedTelegramUsers) {
			return nil
		}
		name := c.Message().Payload
		if len(name) < 3 {
			return c.Send("Username is too short", "text", &tele.SendOptions{
				ReplyTo: c.Message(),
			})
		}
		s.delUser(name)

		return s.onGetUsers(c)
	})

	b.Handle(cmdHelp, func(c tele.Context) error {
		text := "Commands:\n"
		text += cmdStart + " - start bot\n"
		text += cmdReset + " - reset bot memory\n"
		text += cmdModel + " - select model\n"
		text += cmdTemp + " - select temperature\n"
		text += cmdPrompt + " - set prompt\n"
		text += cmdAge + " - set conversation age\n"
		text += cmdPromptCL + " - set default prompt\n"
		text += cmdStream + " - enable/disable streaming\n"
		text += cmdStop + " - stop streaming\n"
		text += cmdInfo + " - show bot info\n"
		text += cmdToJapanese + " - translate to Japanese\n"
		text += cmdToEnglish + " - translate to English\n"
		text += cmdToRussian + " - translate to Russian\n"
		text += cmdUsers + " - show users\n"
		text += cmdAddUser + " - add user\n"
		text += cmdDelUser + " - delete user\n"
		text += cmdHelp + " - show this help\n"

		return c.Send(text, "text", &tele.SendOptions{ReplyTo: c.Message()})
	})

	b.Start()
}

func (s Server) onDocument(c tele.Context) {
	// body
	log.Printf("Got a file: %d", c.Message().Document.FileSize)
	// c.Message().Photo
}

func (s Server) onText(c tele.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(string(debug.Stack()), err)
		}
	}()

	message := c.Message().Payload
	if len(message) == 0 {
		message = c.Message().Text
	}

	s.complete(c, message, true)
}

func (s Server) onVoice(c tele.Context) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(string(debug.Stack()), err)
		}
	}()

	log.Printf("Got a voice, size %d, caption: %s\n", c.Message().Voice.FileSize, c.Message().Voice.Caption)

	s.handleVoice(c)
}

func (s Server) onTranslate(c tele.Context, prefix string) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(string(debug.Stack()), err)
		}
	}()

	query := c.Message().Payload
	if len(query) < 1 {
		_ = c.Send("Please provide a longer prompt", "text", &tele.SendOptions{
			ReplyTo: c.Message(),
		})

		return
	}

	response, err := s.answer(prefix+query, c)
	if err != nil {
		log.Println(err)
		_ = c.Send(err.Error(), "text", &tele.SendOptions{ReplyTo: c.Message()})

		return
	}

	_ = c.Send(response, "text", &tele.SendOptions{
		ReplyTo:   c.Message(),
		ParseMode: tele.ModeMarkdown,
	})
}

func (s Server) onGetUsers(c tele.Context) error {
	users := s.getUsers()
	text := "Users:\n"
	for _, user := range users {
		threads := user.Threads
		var historyLen int64
		var updatedAt time.Time
		var totalTokens int
		if len(threads) > 0 {
			s.db.Model(&ChatMessage{}).Where("chat_id = ?", threads[0].ID).Count(&historyLen)
			updatedAt = threads[0].UpdatedAt
			totalTokens = threads[0].TotalTokens
		}

		text += fmt.Sprintf("*%s*, history: *%d*, last used: *%s*, usage: *%d*\n", user.Username, historyLen, updatedAt.Format("2006/01/02 15:04"), totalTokens)
	}

	return c.Send(text, "text", &tele.SendOptions{ReplyTo: c.Message(), ParseMode: tele.ModeMarkdown})
}

func (s Server) complete(c tele.Context, message string, reply bool) {
	chat := s.getChat(c.Chat().ID, c.Sender().Username)
	if strings.HasPrefix(strings.ToLower(message), "reset") {
		s.deleteHistory(chat.ID)
		return
	}

	text := "..."
	sentMessage := c.Message()
	if !reply {
		text = fmt.Sprintf("_Transcript:_\n%s\n\n_Answer:_ \n\n", message)
		sentMessage, _ = c.Bot().Send(c.Recipient(), text, "text", &tele.SendOptions{
			ReplyTo:   c.Message(),
			ParseMode: tele.ModeMarkdown,
		})
		c.Set("reply", *sentMessage)
	}

	response, err := s.answer(message, c)
	if err != nil {
		return
	}
	log.Printf("User: %s. Response length: %d\n", c.Sender().Username, len(response))

	if len(response) == 0 {
		return
	}

	if len(response) > 4096 {
		file := tele.FromReader(strings.NewReader(response))
		_ = c.Send(&tele.Document{File: file, FileName: "answer.txt", MIME: "text/plain"})
		return
	}
	if !reply {
		text = text[:len(text)-3] + response
		if _, err := c.Bot().Edit(sentMessage, text, "text", &tele.SendOptions{
			ReplyTo:   c.Message(),
			ParseMode: tele.ModeMarkdown,
		}); err != nil {
			c.Bot().Edit(sentMessage, text)
		}
		return
	}

	_ = c.Send(response, "text", &tele.SendOptions{
		ReplyTo:   c.Message(),
		ParseMode: tele.ModeMarkdown,
	})
}

// getChat returns chat from db or creates a new one
func (s Server) getChat(chatID int64, username string) Chat {
	var chat Chat

	s.db.FirstOrCreate(&chat, Chat{ChatID: chatID})
	if len(chat.MasterPrompt) == 0 {
		chat.MasterPrompt = masterPrompt
		chat.ModelName = "gpt-3.5-turbo"
		chat.Temperature = 0.8
		chat.ConversationAge = 1
		s.db.Save(&chat)
	}

	if len(username) > 0 && chat.UserID == 0 {
		user := s.getUser(username)
		chat.UserID = user.ID
		s.db.Save(&chat)
	}

	if chat.ConversationAge == 0 {
		chat.ConversationAge = 1
		s.db.Save(&chat)
	}

	s.db.Find(&chat.History, "chat_id = ?", chat.ID)
	log.Printf("History %d, chatid %d\n", len(chat.History), chat.ID)

	return chat
}

// getUsers returns all users from db
func (s Server) getUsers() []User {
	var users []User
	s.db.Model(&User{}).Preload("Threads").Find(&users)

	return users
}

// getUser returns user from db
func (s Server) getUser(username string) User {
	var user User
	s.db.First(&user, User{Username: username})

	return user
}

func (s Server) addUser(username string) {
	s.db.Create(&User{Username: username})
}

func (s Server) delUser(userNane string) {
	s.db.Where("username = ?", userNane).Delete(&User{})
}

func (s Server) deleteHistory(chatID uint) {
	s.db.Where("chat_id = ?", chatID).Delete(&ChatMessage{})
}

// generate a user-agent value
func userAgent(userID int64) string {
	return fmt.Sprintf("telegram-chatgpt-bot:%d", userID)
}

// Restrict returns a middleware that handles a list of provided
// usernames with the logic defined by In and Out functions.
// If the username is found in the Usernames field, In function will be called,
// otherwise Out function will be called.
func Restrict(v RestrictConfig) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		if v.In == nil {
			v.In = next
		}
		if v.Out == nil {
			v.Out = next
		}
		return func(c tele.Context) error {
			for _, username := range v.Usernames {
				if username == c.Sender().Username {
					return v.In(c)
				}
			}
			return v.Out(c)
		}
	}
}

// Whitelist returns a middleware that skips the update for users
// NOT specified in the usernames field.
func (s Server) whitelist() tele.MiddlewareFunc {
	admins := s.conf.AllowedTelegramUsers
	var usernames []string
	s.db.Model(&User{}).Pluck("username", &usernames)
	for _, username := range admins {
		if !in_array(username, usernames) {
			usernames = append(usernames, username)
		}
	}

	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return Restrict(RestrictConfig{
			Usernames: usernames,
			In:        next,
			Out: func(c tele.Context) error {
				return c.Send(fmt.Sprintf("not allowed: %s", c.Sender().Username), "text", &tele.SendOptions{ReplyTo: c.Message()})
			},
		})(next)
	}
}
