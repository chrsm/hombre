package hombre

import (
	"net/http"

	"github.com/cjoudrey/gluahttp"
	"github.com/layeh/gopher-luar"
	"github.com/yuin/gluare"
	"github.com/yuin/gopher-lua"
)

func (h *Hombre) makeLuaVM() *lua.LState {
	// init the lua vm (caller's responsibility to close it)
	l := lua.NewState(lua.Options{IncludeGoStackTrace: true})

	l.PreloadModule("re", gluare.Loader)                                   // regex
	l.PreloadModule("http", gluahttp.NewHttpModule(&http.Client{}).Loader) // http

	l.SetGlobal("hombre", luar.New(l, h))
	l.SetGlobal("conf", luar.New(l, h.Conf))
	l.SetGlobal("slack", luar.New(l, h.API))
	l.SetGlobal("luaWorkingDir", luar.New(l, h.Conf.Lua.Path))

	return l
}
