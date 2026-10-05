package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	cmdUnknown = "неизвестная команда"
	itemKeys   = "ключи"
)

type Place struct {
	Name  string
	Items []string
}

type Action struct {
	Item   string
	Target string
	Run    func(p *Player) string
}

type Room struct {
	Name        string
	EnterText   string
	Description string
	Todo        func(p *Player) string
	Objects     []*Place
	Routes      []*Room
	CanGo       map[string]func(p *Player) (bool, string)
	Actions     []Action
}

type Player struct {
	Location    *Room
	HasBackpack bool
	Inventory   []string
	DoorOpen    bool
}

func roomNames(rooms []*Room) string {
	names := make([]string, 0, len(rooms))
	for _, room := range rooms {
		names = append(names, room.Name)
	}

	return strings.Join(names, ", ")
}

func (p *Player) LookAround() string {
	var parts []string
	if p.Location.Description != "" {
		parts = append(parts, p.Location.Description)
	}

	for _, place := range p.Location.Objects {
		if len(place.Items) > 0 {
			parts = append(parts, place.Name+": "+strings.Join(place.Items, ", "))
		}
	}

	if p.Location.Todo != nil {
		if todo := p.Location.Todo(p); todo != "" {
			parts = append(parts, todo)
		}
	}

	var ans string
	if len(parts) > 0 {
		ans = strings.Join(parts, ", ") + "."
	} else {
		ans = "пустая комната."
	}

	ans += " можно пройти - " + roomNames(p.Location.Routes)

	return ans
}

func (p *Player) GoTo(location string) string {
	if p.Location.Routes != nil {
		for _, room := range p.Location.Routes {
			if room.Name == location {
				if p.Location.CanGo != nil {
					if check, ok := p.Location.CanGo[location]; ok {
						if ok, msg := check(p); !ok {
							return msg
						}
					}
				}

				if room.Name == "домой" {
					p.Location.Actions[0].Run(p)
				} else {
					p.Location = room
				}

				return p.Location.EnterText + " можно пройти - " + roomNames(p.Location.Routes)
			}
		}
	}

	return "нет пути в " + location
}

func removeItemFromRoom(room *Room, name string) bool {
	for _, place := range room.Objects {
		for i, item := range place.Items {
			if item == name {
				place.Items = append(place.Items[:i], place.Items[i+1:]...)
				return true
			}
		}
	}

	return false
}

func contains(items []string, item string) bool {
	for _, i := range items {
		if i == item {
			return true
		}
	}

	return false
}

func (p *Player) PutOn(thing string) string {
	if thing == "рюкзак" && !p.HasBackpack {
		if removeItemFromRoom(p.Location, thing) {
			p.HasBackpack = true
			return "вы надели: " + thing
		}
	}

	return "нет такого"
}

func (p *Player) Get(thing string) string {
	if !p.HasBackpack {
		return "некуда класть"
	}

	if removeItemFromRoom(p.Location, thing) {
		p.Inventory = append(p.Inventory, thing)
		return "предмет добавлен в инвентарь: " + thing
	}

	return "нет такого"
}

func (p *Player) Apply(what, to string) string {
	if !contains(p.Inventory, what) {
		return "нет предмета в инвентаре - " + what
	}

	for _, action := range p.Location.Actions {
		if action.Item == what && action.Target == to {
			return action.Run(p)
		}
	}

	if what == itemKeys && to == "дверь" {
		p.DoorOpen = true
		return "дверь открыта"
	}

	return "не к чему применить"
}

var player Player

func main() {
	initGame()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Println(handleCommand(scanner.Text()))
	}
}

func initGame() {
	kitchen := &Room{
		Name:        "кухня",
		EnterText:   "кухня, ничего интересного.",
		Description: "ты находишься на кухне",
		Objects: []*Place{
			{"на столе", []string{"чай"}},
		},
	}

	kitchen.Todo = func(p *Player) string {
		if p.HasBackpack {
			return "надо идти в универ"
		}
		return "надо собрать рюкзак и идти в универ"
	}

	corridor := &Room{
		Name:        "коридор",
		EnterText:   "ничего интересного.",
		Description: "ничего интересного",
	}

	corridor.CanGo = map[string]func(p *Player) (bool, string){
		"улица": func(p *Player) (bool, string) {
			if !p.DoorOpen {
				return false, "дверь закрыта"
			}

			return true, ""
		},
	}

	corridor.Actions = []Action{
		{
			itemKeys,
			"дверь",
			func(p *Player) string {
				p.DoorOpen = true
				return "дверь открыта"
			},
		},
	}

	room := &Room{
		Name:      "комната",
		EnterText: "ты в своей комнате.",
		Objects: []*Place{
			{"на столе", []string{itemKeys, "конспекты"}},
			{"на стуле", []string{"рюкзак"}},
		},
	}

	street := &Room{
		Name:        "улица",
		EnterText:   "на улице весна.",
		Description: "на улице весна",
	}

	street.Actions = []Action{
		{
			Run: func(p *Player) string {
				p.Location = corridor
				return ""
			},
		},
	}

	home := &Room{
		Name:        "домой",
		EnterText:   "дома.",
		Description: "дома",
	}

	player.Location = kitchen
	kitchen.Routes = []*Room{corridor}
	corridor.Routes = []*Room{kitchen, room, street}
	room.Routes = []*Room{corridor}
	street.Routes = []*Room{home}
	home.Routes = []*Room{street}

	player.HasBackpack = false
	player.Inventory = []string{}
	player.DoorOpen = false
}

func handleCommand(command string) string {
	cmd := strings.Fields(command)

	if len(cmd) == 0 {
		return cmdUnknown
	}

	switch cmd[0] {
	case "осмотреться":
		if len(cmd) < 1 {
			return cmdUnknown
		}

		return player.LookAround()

	case "идти":
		if len(cmd) < 2 {
			return cmdUnknown
		}

		return player.GoTo(cmd[1])

	case "надеть":
		if len(cmd) < 2 {
			return cmdUnknown
		}

		return player.PutOn(cmd[1])

	case "взять":
		if len(cmd) < 2 {
			return cmdUnknown
		}

		return player.Get(cmd[1])

	case "применить":
		if len(cmd) < 3 {
			return cmdUnknown
		}

		return player.Apply(cmd[1], cmd[2])

	default:
		return cmdUnknown
	}
}
