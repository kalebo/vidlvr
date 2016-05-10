package main

import (
	"fmt"
	"github.com/thoj/go-ircevent"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"time"
)

var room = "#Physics"

var yt, _ = regexp.Compile(`^(https?\:\/\/)?(www\.)?(youtube\.com|youtu\.?be)\/.+$`)
var im, _ = regexp.Compile(`^[^\s](?:([^:/?#]+):)?(?://([^/?#]*))?([^?#]*\.(?:jpg|gif|png))$`)
var wv, _ = regexp.Compile(`^[^\s](?:([^:/?#]+):)?(?://([^/?#]*))?([^?#]*\.(?:gifv|webm))$`)
var vl, _ = regexp.Compile(`!vol [0-9]+`)
var tx, _ = regexp.Compile(`!write .+`)
var fn, _ = regexp.Compile(`[^/]*$`)

func powerscreen() {
	exec.Command("xset", "dpms", "force", "on").Run()
}

func closeafter(name string, duration string) {
	durationobj, _ := time.ParseDuration(duration)
	time.Sleep(durationobj)
	exec.Command("wmctrl", "-c", name).Run()
}

func cycledisplay() {
	name := ":ACTIVE:" // active refers to the last open window which cycles to the left
	exec.Command("wmctrl", "-r", name, "-b", "remove,fullscreen").Run()
	exec.Command("wmctrl", "-r", name, "-b", "remove,maximized_vert,maximized_horz").Run()
	exec.Command("wmctrl", "-r", name, "-e", "0,0,0,-1,-1").Run()
	exec.Command("wmctrl", "-r", name, "-b", "add,fullscreen").Run()
}

func dltemp(url string) string {
	flocation := "/tmp/" + fn.FindString(url)
	out, _ := os.Create(flocation)
	defer out.Close()
	rdr, _ := http.Get(url)
	defer rdr.Body.Close()
	io.Copy(out, rdr.Body)
	return flocation
}

func main() {
	con := irc.IRC("fanta", "fanta")
	err := con.Connect("jupiter.byu.edu:6667")
	if err != nil {
		fmt.Println("Failed to connect!")
		return
	}

	con.AddCallback("001", func(e *irc.Event) {
		con.Join(room)
	})

	// Youtube Player
	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		if yt.MatchString(e.Message()) {
			con.Privmsg(room, "Playing youtube video -- use `!stop` to abort playback.")
			powerscreen()
			cycledisplay()
			exec.Command("mpv", "-fs", e.Message()).Start()
		}
	})

	// Webm or Gifv Player
	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		if wv.MatchString(e.Message()) {
			con.Privmsg(room, "Playing gifv or webm -- use `!stop` to abort playback.")
			powerscreen()
			cycledisplay()
			exec.Command("mpv", "-fs", "--loop=inf", e.Message()).Start()
			go closeafter(flocation, "1h")
		}
	})

	// Kill mpv
	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		if e.Message() == "!stop" {
			cmd := exec.Command("killall", "mpv")
			cmdOut, _ := cmd.Output()
			fmt.Println(string(cmdOut))
		}
	})

	// Image Display
	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		if im.MatchString(e.Message()) {
			flocation := dltemp(e.Message())
			con.Privmsg(room, "Displaying image -- use `!wipe` to close image.")
			powerscreen()
			cycledisplay()
			exec.Command("pqiv", "-itf", flocation).Start()
			go closeafter(flocation, "1h")
		}
	})

	// Kill pqiv
	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		if e.Message() == "!wipe" {
			cmd := exec.Command("killall", "pqiv")
			cmdOut, _ := cmd.Output()
			fmt.Println(string(cmdOut))
		}
	})

	// Set system volume
	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		if vl.MatchString(e.Message()) {
			con.Privmsg(room, "Setting volume at "+e.Message()[5:]+"%")
			exec.Command("amixer", "sset", "'Master'", e.Message()[5:]+"%").Start()
		}
	})

	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		if tx.MatchString(e.Message()) {
			powerscreen()
			cycledisplay()
			exec.Command("sm", "-b black", "-f white", e.Message()[7:]).Start()
		}
	})

	// Print help info
	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		if e.Message() == "!help" {
			helpmsg := "I can display png, gif, jpg, webm, gifv, or youtube urls. I will ignore urls with a leading space. I can also adjust the volume with `!vol <percent>`. Use `!write` to print text to the screen."
			con.Privmsg(room, helpmsg)
		}
	})

	con.Loop()
}
