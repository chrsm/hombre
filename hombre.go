package hombre

import (
	"log"
	"regexp"
	"sync"

	"github.com/slack-go/slack"
	"layeh.com/gopher-luar"
)

type Hombre struct {
	API *slack.Client

	running      bool
	sync.Mutex   // embedded
	serviceChans map[string]service
	done         chan bool

	path     string
	scripts  []Script
	services []Script
}

type service struct {
	msgs chan *slack.MessageEvent
	done chan bool
}

// New creates a new Donut Bot instance and sets it up for work.
func New(token string, opts ...Option) *Hombre {
	h := &Hombre{
		API:          slack.New(token),
		serviceChans: make(map[string]service),
		done:         make(chan bool),
	}

	for i := range opts {
		opts[i](h)
	}

	return h
}

type Option func(*Hombre)

func OptionLuaPath(path string) Option {
	return func(h *Hombre) {
		h.path = path
	}
}

type Script struct {
	Path     string
	Name     string
	Commands []string
}

func (h *Hombre) AddScript(s Script) {
	if h.running {
		panic("can't add a script to a running instance atm")
	}

	h.scripts = append(h.scripts, s)
}

func (h *Hombre) AddService(s Script) {
	if h.running {
		panic("can't add a service to a running instance atm")
	}

	h.services = append(h.services)
}

// Listen starts the Donut Bot client and lua scripts.
func (h *Hombre) Listen() {
	// Initialize scripts that are "long running", one per goroutine (they need their own lua instance).
	// Each of these inits should return a channel on which they accept *slack.MessageEvent(s)

	rtm := h.API.NewRTM()
	go rtm.ManageConnection()
	log.Printf("hombre: rtm running")

	// Initialize scripts that are "long running", one per goroutine (they need their own lua instance).
	// Each of these inits should return a channel on which they accept *slack.MessageEvent(s)
	for i := range h.services {
		go func(svc Script) {
			log.Printf("hombre: starting svc(%s)", svc.Name)
			vm := h.makeLuaVM()

			cpath := vm.GetGlobal("package.path")
			vm.SetGlobal("package.path", luar.New(vm, cpath.String()+";/?.lua;/?/init.lua"))

			// register the message in the global scope of the script;
			// this is how this script will receive messages
			ch := make(chan *slack.MessageEvent)
			vm.SetGlobal("msgch", luar.New(vm, ch))

			// register slack's RTM instance
			vm.SetGlobal("rtm", luar.New(vm, rtm))

			// register the message channel
			h.Lock()
			h.serviceChans[svc.Name] = service{
				msgs: ch,
				done: make(chan bool),
			}
			h.Unlock()

			// run the script
			log.Printf("hombre: running service script")
			if err := vm.DoFile(h.path + "/" + svc.Name + ".lua"); err != nil {
				panic(err)
			}

			log.Printf("hombre: shutting down lua vm for svc(%s)", svc.Name)
			vm.Close()

			h.serviceChans[svc.Name].done <- true
		}(h.services[i])
	}

proc:
	for {
		select {
		case <-h.done:
			// we're done here
			log.Println("Goodbye!")
			break proc
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.DisconnectedEvent:
				log.Println("We got disconnected...should reconnect automagically")
			case *slack.ConnectedEvent:
				log.Println("Connected!")
			case *slack.MessageEvent:
				log.Printf("[%s] %s: %s\n", ev.Channel, ev.User, ev.Text)

				// Services have the highest priority here
				for i := range h.services {
					svc := h.services[i]
					if svc.acceptsCommand(ev.Text) {
						h.serviceChans[svc.Name].msgs <- ev
					}
				}

				// Check if any scripts want this message
				for i := range h.scripts {
					scr := h.scripts[i]

					if scr.acceptsCommand(ev.Text) {
						go func(ev *slack.MessageEvent) {
							log.Printf("executing script(%s)", scr.Name)
							vm := h.makeLuaVM()

							// set the message
							vm.SetGlobal("msg", luar.New(vm, ev))
							vm.SetGlobal("rtm", luar.New(vm, rtm))

							if err := vm.DoFile(h.path + "/" + scr.Name + ".lua"); err != nil {
								log.Println(ev.Text, err)
							}

							vm.Close()
						}(ev)
					}
				}
			default:
				log.Printf("event came in: %#v", msg)
				log.Printf("event data: %#v", msg.Data)

				if err, ok := msg.Data.(*slack.ConnectionErrorEvent); ok {
					log.Printf("err occurred? %#v", err.ErrorObj)
					panic("")
				}
				// nil
			}
		}
	}
}

func (h *Hombre) Close() {
	// kill the listener

	// stop running goroutines
	for i := range h.serviceChans {
		// The long-running script should exit on its own,
		close(h.serviceChans[i].msgs)
		// but we'll wait until the VM is closed
		<-h.serviceChans[i].done
	}

	h.done <- true
}

func (s Script) acceptsCommand(cmd string) bool {
	for i := range s.Commands {
		if s.Commands[i] == "*" { // wildcard
			return true
		} else if ok, err := regexp.MatchString("^!"+regexp.QuoteMeta(s.Commands[i]), cmd); ok && err == nil {
			return true
		}
	}

	return false
}
