package main

import (
	"github.com/aditya-K2/goMP/cache"
	"github.com/aditya-K2/goMP/utils"
	"github.com/spf13/viper"
	"gitlab.com/diamondburned/ueberzug-go"
)

/*
	Renderer is just a channel on which we will send the Path to the song whose
	Image is to be Rendered. This channel is passed to the openImage which in turn is called
	by the Start() function as a go routine.
*/
type Renderer struct {
	c chan string
}

/*
	Returns a new Renderer with a string channel
*/
func newRenderer() *Renderer {
	c := make(chan string)
	return &Renderer{
		c: c,
	}
}

/*
	Send Image Path to Renderer
*/
func (self *Renderer) Send(path string) {
	self.c <- path
}

/*

   Go Routine that will Be Called and will listen on the channel c
   for changes and on getting a string over the channel will open the Image and
   keep listening again. This will keep the image blocked ( i.e no need to use time.Sleep() etc. )
   and saves resources too.

*/
func openImage(path string, c chan string) {
	fw, fh := utils.GetFontWidth()
	var im *ueberzug.Image
	if path != "stop" {
		extractedImage := getImagePath(path)
		img2, _ := getImg(extractedImage)
		im, _ = ueberzug.NewImage(img2, int(float32(IMG_X)*fw)+viper.GetInt("ADDITIONAL_PADDING_X"), int(float32(IMG_Y)*fh)+viper.GetInt("ADDITIONAL_PADDING_Y"))
	}
	d := <-c
	if im != nil {
		im.Clear()
	}
	if d != "stop" {
		openImage(d, c)
	} else {
		openImage("stop", c)
	}
}

/*
	Initialises the Renderer and calls the go routine openImage and passes the channel
	as argument.
*/
func (self *Renderer) Start(path string) {
	go openImage(path, self.c)
}

/*
	This Function returns the path to the image that is to be rendered it checks first for the image in the cache
	else it adds the image to the cache and then extracts it and renders it.
*/
func getImagePath(path string) string {
	a, err := CONN.ListInfo(path)
	var extractedImage string
	if err == nil && len(a) != 0 {
		if cache.Exists(a[0]["artist"], a[0]["album"]) {
			extractedImage = cache.GenerateName(a[0]["artist"], a[0]["album"])
		} else {
			imagePath := cache.GenerateName(a[0]["artist"], a[0]["album"])
			absPath := utils.CheckDirectoryFmt(viper.GetString("MUSIC_DIRECTORY")) + path
			extractedImage = extractImageFromFile(absPath, imagePath)
			if extractedImage == viper.GetString("DEFAULT_IMAGE_PATH") && viper.GetString("GET_COVER_ART_FROM_LAST_FM") == "TRUE" {
				downloadedImage, err := getImageFromLastFM(a[0]["artist"], a[0]["album"], imagePath)
				if err == nil {
					NOTIFICATION_SERVER.Send("Image From LastFM")
					extractedImage = downloadedImage
				} else {
					NOTIFICATION_SERVER.Send("Falling Back to Default Image.")
				}
			} else {
				NOTIFICATION_SERVER.Send("Extracted Image Successfully")
			}
		}
	}
	return extractedImage
}
