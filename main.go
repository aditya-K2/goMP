package main

import (
	"strconv"
	"time"

	"github.com/aditya-K2/gomp/client"
	"github.com/aditya-K2/gomp/utils"

	"github.com/aditya-K2/fuzzy"
	"github.com/aditya-K2/gomp/cache"
	"github.com/aditya-K2/gomp/config"
	"github.com/fhs/gompd/mpd"
	"github.com/gdamore/tcell/v2"
	"github.com/spf13/viper"
)

var (
	CONN                *mpd.Client
	UI                  *Application
	NOTIFICATION_SERVER *NotificationServer
	RENDERER            *Renderer
	Volume              int64
	Random              bool
	Repeat              bool
	InsidePlaylist      bool = true
	InsideSearchView    bool = false
	ARTIST_TREE         map[string]map[string]map[string]string
)

func main() {
	config.ReadConfig()
	// Connect to MPD server
	var mpdConnectionError error
	CONN, mpdConnectionError = mpd.Dial("tcp", "localhost:"+viper.GetString("MPD_PORT"))
	if mpdConnectionError != nil {
		panic(mpdConnectionError)
	}
	defer CONN.Close()
	cache.SetCacheDir(viper.GetString("CACHE_DIR"))
	RENDERER = newRenderer()
	c, _ := CONN.CurrentSong()
	if len(c) != 0 {
		RENDERER.Start(c["file"])
	} else {
		RENDERER.Start("stop")
	}

	UI = newApplication()

	fileMap, err := CONN.GetFiles()
	dirTree := client.GenerateDirectoryTree(fileMap)

	client.SetConnection(CONN)
	client.UpdatePlaylist(UI.ExpandedView)

	_v, _ := CONN.Status()
	Volume, _ = strconv.ParseInt(_v["volume"], 10, 64)
	Random, _ = strconv.ParseBool(_v["random"])
	Repeat, _ = strconv.ParseBool(_v["repeat"])

	ARTIST_TREE, err = client.GenerateArtistTree()
	ARTIST_TREE_CONTENT := utils.ConvertToArray(ARTIST_TREE)
	NOTIFICATION_SERVER = NewNotificationServer()
	NOTIFICATION_SERVER.Start()
	client.SetNotificationServer(NOTIFICATION_SERVER)

	var SEARCH_CONTENT_SLICE []interface{}

	UI.ExpandedView.SetDrawFunc(func(s tcell.Screen, x, y, width, height int) (int, int, int, int) {
		if InsidePlaylist {
			client.UpdatePlaylist(UI.ExpandedView)
		} else if InsideSearchView {
			client.UpdateSearchView(UI.ExpandedView, SEARCH_CONTENT_SLICE)
		} else {
			client.Update(dirTree.Children, UI.ExpandedView)
		}
		return UI.ExpandedView.GetInnerRect()
	})

	var FUNC_MAP = map[string]func(){
		"showChildrenContent": func() {
			r, _ := UI.ExpandedView.GetSelection()
			if !InsidePlaylist && !InsideSearchView {
				if len(dirTree.Children[r].Children) == 0 {
					id, _ := CONN.AddId(dirTree.Children[r].AbsolutePath, -1)
					CONN.PlayId(id)
				} else {
					client.Update(dirTree.Children[r].Children, UI.ExpandedView)
					dirTree = &dirTree.Children[r]
				}
			} else if InsidePlaylist {
				CONN.Play(r)
			} else if InsideSearchView {
				r, _ := UI.ExpandedView.GetSelection()
				client.AddToPlaylist(SEARCH_CONTENT_SLICE[r], true)
			}
		},
		"togglePlayBack": func() {
			client.TogglePlayBack()
		},
		"showParentContent": func() {
			if !InsidePlaylist && !InsideSearchView {
				if dirTree.Parent != nil {
					client.Update(dirTree.Parent.Children, UI.ExpandedView)
					dirTree = dirTree.Parent
				}
			}
		},
		"nextSong": func() {
			CONN.Next()
		},
		"clearPlaylist": func() {
			CONN.Clear()
			NOTIFICATION_SERVER.Send("PlayList Cleared")
		},
		"previousSong": func() {
			CONN.Previous()
		},
		"addToPlaylist": func() {
			if !InsidePlaylist && !InsideSearchView {
				r, _ := UI.ExpandedView.GetSelection()
				CONN.Add(dirTree.Children[r].AbsolutePath)
			} else if InsideSearchView {
				r, _ := UI.ExpandedView.GetSelection()
				client.AddToPlaylist(SEARCH_CONTENT_SLICE[r], false)
			}
		},
		"toggleRandom": func() {
			err := CONN.Random(!Random)
			if err == nil {
				Random = !Random
			}
		},
		"toggleRepeat": func() {
			err := CONN.Repeat(!Repeat)
			if err == nil {
				Repeat = !Repeat
			}
		},
		"decreaseVolume": func() {
			if Volume <= 0 {
				Volume = 0
			} else {
				Volume -= 10
			}
			CONN.SetVolume(int(Volume))
		},
		"increaseVolume": func() {
			if Volume >= 100 {
				Volume = 100
			} else {
				Volume += 10
			}
			CONN.SetVolume(int(Volume))
		},
		"navigateToFiles": func() {
			InsidePlaylist = false
			InsideSearchView = false
			UI.Navbar.Select(1, 0)
			client.Update(dirTree.Children, UI.ExpandedView)
		},
		"navigateToPlaylist": func() {
			InsidePlaylist = true
			InsideSearchView = false
			UI.Navbar.Select(0, 0)
			client.UpdatePlaylist(UI.ExpandedView)
		},
		"navigateToMostPlayed": func() {
			InsideSearchView = false
			InsidePlaylist = false
			UI.Navbar.Select(2, 0)
		},
		"navigateToSearch": func() {
			InsideSearchView = true
			InsidePlaylist = false
			UI.Navbar.Select(3, 0)
		},
		"quit": func() {
			UI.App.Stop()
		},
		"stop": func() {
			CONN.Stop()
			NOTIFICATION_SERVER.Send("Playback Stopped")
		},
		"updateDB": func() {
			_, err = CONN.Update("")
			if err != nil {
				panic(err)
			}
			NOTIFICATION_SERVER.Send("Database Updated")
		},
		"deleteSongFromPlaylist": func() {
			if InsidePlaylist {
				r, _ := UI.ExpandedView.GetSelection()
				CONN.Delete(r, -1)
			}
		},
		"FocusSearch": func() {
			UI.App.SetFocus(UI.SearchBar)
		},
	}

	config.GenerateKeyMap(FUNC_MAP)

	UI.SearchBar.SetAutocompleteFunc(func(c string) []string {
		if c != "" && c != " " && c != "  " {
			_, _, w, _ := UI.SearchBar.GetRect()
			matches := fuzzy.Find(c, ARTIST_TREE_CONTENT)
			var suggestions []string
			for i, match := range matches {
				if i == 10 {
					break
				}
				suggestions = append(suggestions, utils.GetFormattedString(match.Str, w-2))
			}
			return suggestions
		} else {
			return make([]string, 0)
		}
	})

	UI.ExpandedView.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if val, ok := config.KEY_MAP[int(e.Rune())]; ok {
			FUNC_MAP[val]()
			return nil
		} else {
			return e
		}
	})

	UI.SearchBar.SetDoneFunc(func(e tcell.Key) {
		if e == tcell.KeyEnter {
			UI.ExpandedView.Select(0, 0)
			InsideSearchView = true
			InsidePlaylist = false
			SEARCH_CONTENT_SLICE = nil
			SEARCH_CONTENT_SLICE, err = client.GenerateContentSlice(UI.SearchBar.GetText())
			if err != nil {
				NOTIFICATION_SERVER.Send("Could Not Retrieve the Results")
			} else {
				UI.SearchBar.SetText("")
				UI.App.SetFocus(UI.ExpandedView)
				UI.Navbar.Select(3, 0)
			}
		}
		if e == tcell.KeyEscape {
			InsideSearchView = false
			UI.App.SetFocus(UI.ExpandedView)
		}
	})

	go func() {
		for {
			UI.App.Draw()
			time.Sleep(time.Second)
		}
	}()

	if err := UI.App.Run(); err != nil {
		panic(err)
	}
}
