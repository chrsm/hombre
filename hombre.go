package hombre

import (
	"fmt"
	"sync"

	"github.com/layeh/gopher-luar"
	"github.com/nlopes/slack"
)

type Hombre struct {
	Conf *Config
	API  *slack.Client

	sync.Mutex   // embedded
	serviceChans map[string]service
	done         chan bool
}

type service struct {
	msgs chan *slack.MessageEvent
	done chan bool
}

// New creates a new Donut Bot instance and sets it up for work.
func New(c *Config) *Hombre {
	return &Hombre{
		Conf:         c,
		API:          slack.New(c.Token),
		serviceChans: make(map[string]service),
		done:         make(chan bool),
	}
}

// Listen starts the Donut Bot client and lua scripts.
func (h *Hombre) Listen() {
	// Initialize scripts that are "long running", one per goroutine (they need their own lua instance).
	// Each of these inits should return a channel on which they accept *slack.MessageEvent(s)

	rtm := h.API.NewRTM()
	go rtm.ManageConnection()

	// Initialize scripts that are "long running", one per goroutine (they need their own lua instance).
	// Each of these inits should return a channel on which they accept *slack.MessageEvent(s)
	for i := range h.Conf.Lua.Services {
		go func(svc LuaScript) {
			vm := h.makeLuaVM()

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
			if err := vm.DoFile(h.Conf.Lua.Path + "/" + svc.Name + ".lua"); err != nil {
				panic(err)
			}

			vm.Close()
			h.serviceChans[svc.Name].done <- true
		}(h.Conf.Lua.Services[i])
	}

proc:
	for {
		select {
		case <-h.done:
			// we're done here
			fmt.Println("Goodbye!")
			break proc
		case msg := <-rtm.IncomingEvents:
			switch ev := msg.Data.(type) {
			case *slack.DisconnectedEvent:
				fmt.Println("We got disconnected...reconnecting")
				rtm.Reconnect()
			case *slack.ConnectedEvent:
				fmt.Println("Connected!")
			case *slack.MessageEvent:
				fmt.Printf("[%s] %s: %s\n", ev.Channel, ev.User, ev.Text)

				// Services have the highest priority here
				for i := range h.Conf.Lua.Services {
					svc := h.Conf.Lua.Services[i]
					if svc.acceptsCommand(ev.Text) {
						h.serviceChans[svc.Name].msgs <- ev
					}
				}

				// Check if any scripts want this message
				for i := range h.Conf.Lua.Scripts {
					scr := h.Conf.Lua.Scripts[i]

					if scr.acceptsCommand(ev.Text) {
						go func(ev *slack.MessageEvent) {
							vm := h.makeLuaVM()

							// set the message
							vm.SetGlobal("msg", luar.New(vm, ev))
							vm.SetGlobal("rtm", luar.New(vm, rtm))

							if err := vm.DoFile(h.Conf.Lua.Path + "/" + scr.Name + ".lua"); err != nil {
								fmt.Println(ev.Text, err)
							}

							vm.Close()
						}(ev)
					}
				}
			case *slack.HelloEvent:
				// nil
			default:
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
