package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aditya-K2/gomp/utils"
	"github.com/spf13/viper"
)

var (
	HOME_DIR, _       = os.UserHomeDir()
	USER_CACHE_DIR, _ = os.UserCacheDir()
	defaults          = map[string]interface{}{
		"ADDITIONAL_PADDING_X": 12,
		"ADDITIONAL_PADDING_Y": 16,
		"IMAGE_WIDTH_EXTRA_X":  -1.5,
		"IMAGE_WIDTH_EXTRA_Y":  -3.75,
		"NETWORK_TYPE":         "tcp",
		"NETWORK_ADDRESS":      "localhost",
		"MUSIC_DIRECTORY":      utils.CheckDirectoryFmt(getMusicDirectory()),
		"PORT":                 "6600",
		"DEFAULT_IMAGE_PATH":   "default.jpg",
		"CACHE_DIR":            utils.CheckDirectoryFmt(USER_CACHE_DIR),
		"SEEK_OFFSET":          1,
	}
)

func ReadConfig() {

	for k, v := range defaults {
		viper.SetDefault(k, v)
	}

	viper.SetConfigName("config")
	viper.AddConfigPath(HOME_DIR + "/.config/gomp")

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Could Not Read Config file.")
	}

	// Expanding ~ to the User's Home Directory
	viper.Set("MUSIC_DIRECTORY", utils.ExpandHomeDir(viper.GetString("MUSIC_DIRECTORY")))
	viper.Set("DEFAULT_IMAGE_PATH", utils.ExpandHomeDir(viper.GetString("DEFAULT_IMAGE_PATH")))
	viper.Set("CACHE_DIR", utils.ExpandHomeDir(viper.GetString("CACHE_DIR")))
	viper.Set("NETWORK_ADDRESS", utils.ExpandHomeDir(viper.GetString("NETWORK_ADDRESS")))
}

func GenerateKeyMap(funcMap map[string]func()) {
	for k := range funcMap {
		mappingsForFunctionK := viper.GetStringSlice(k)
		if len(mappingsForFunctionK) != 0 {
			for _, i := range mappingsForFunctionK {
				aV, err := GetAsciiValue(i)
				if err == nil {
					KEY_MAP[aV] = k
				}
			}
		}
	}
}

func getMusicDirectory() string {
	content, err := ioutil.ReadFile(HOME_DIR + "/.config/mpd/mpd.conf")
	if err != nil {
		fmt.Println("No Config File for mpd Found")
		panic(err)
	}
	ab := string(content)
	maps := strings.Split(ab, "\n")
	for _, j := range maps {
		if strings.Contains(j, "music_directory") {
			s := strings.SplitAfter(strings.ReplaceAll(j, " ", ""), "y")[1]
			s = strings.ReplaceAll(s, "\t", "")
			d := ""
			for z, m := range s {
				if (z != 0) && (z != (len(s) - 1)) {
					d += string(m)
				}
			}
			return d
		}
	}
	return ""
}
