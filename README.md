hombre
===========

This is a Slack bot on drugs. If lua were drugs, I mean.

README is sorta bare right now, as this project is being massaged out of a
project at work.

Lua Services and Scripts
===========

Lua is a pretty simple language. The VM implementation is based on Lua 5.1,
even though there's like a 5.3 now.

There are a few globals available to *all* scripts:

	* hombre
	* conf
	* luaWorkingDir

And a few modules:

	* re (regex)
	* http
	* json

###About "Services"

The "service" scripts are initialized when hombre starts up. These receive a
go channel that accepts messages from Slack (`msgch`).

At this point, it's time to stop any further execution, hence the standard loop:

    local exit = false
    while not exit do
        local msg, ok = msgch:receive()
        
        if not ok then
            exit = true
        else
            -- do stuff with msg
        end
    end

If an error is encountered during service execution, the lua VM will cause a panic
in the hombre "host", requiring a restart.

###About "Scripts"

One-off scripts are run on-demand when a !command matches what is configured in `config.json`.
These scripts receive a single global `msg` variable.

Once the script is done, the VM is closed.

A quick way to get the "arguments" to a command is like so:

    local cmd = "!hey "
    local query = string.sub(msg.Text, #cmd) -- msg = global


