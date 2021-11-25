package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/MarcoLucidi01/ytcast/dial"
	"github.com/MarcoLucidi01/ytcast/youtube"
)

const (
	progName      = "ytcast"
	cacheFileName = progName + ".json"
	searchTimeout = 3
)

type cacheEntry struct {
	Device *dial.Device
	Remote *youtube.Remote
}

var (
	errNoDevFound = errors.New("no device found")
	errCanceled   = errors.New("canceled")

	flagVerbose = flag.Bool("verbose", false, "enable verbose logging")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [-v] [videoId...]\n", progName)
		flag.PrintDefaults()
	}
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	if !*flagVerbose {
		log.SetOutput(ioutil.Discard)
	}

	if err := run(); err != nil {
		log.Println(err)
		fmt.Fprintf(os.Stderr, "%s: %s\n", progName, err)
		os.Exit(1)
	}
}

func run() error {
	cacheFilePath := path.Join(mkCacheDir(), cacheFileName)
	cache := loadCache(cacheFilePath)
	defer saveCache(cacheFilePath, cache)

	dev, err := searchAndSelectDevice(cache)
	if err != nil {
		return err
	}
	if err := dev.WakeupFunc(); err != nil {
		return err
	}
	app, err := launchYouTubeApp(dev)
	if err != nil {
		return err
	}
	if flag.NArg() > 0 {
		return playVideos(cache[dev.UniqueServiceName], app, flag.Args())
	}
	return nil
}

func mkCacheDir() string {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Println(err)
			return "." // current directory
		}
		cacheDir = path.Join(homeDir, ".cache")
	}
	cacheDir = path.Join(cacheDir, progName)
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Println(err)
		return "."
	}
	return cacheDir
}

func loadCache(fpath string) map[string]*cacheEntry {
	log.Printf("loading cache %s", fpath)
	data, err := ioutil.ReadFile(fpath)
	if err != nil {
		log.Println(err)
		return make(map[string]*cacheEntry)
	}
	var cache map[string]*cacheEntry
	if err = json.Unmarshal(data, &cache); err != nil {
		log.Println(err)
		return make(map[string]*cacheEntry)
	}
	return cache
}

func saveCache(fpath string, cache map[string]*cacheEntry) {
	log.Printf("saving cache %s", fpath)
	data, err := json.Marshal(cache)
	if err != nil {
		log.Printf("marshal cache: %s", err)
		return
	}
	if err := ioutil.WriteFile(fpath, data, 0600); err != nil {
		log.Println(err)
	}
}

func searchAndSelectDevice(cache map[string]*cacheEntry) (*dial.Device, error) {
	var devices []*dial.Device
	var err error

	// show cached devices first (if any)
	for _, entry := range cache {
		devices = append(devices, entry.Device)
	}
	if len(devices) == 0 {
		if devices, err = searchDevices(); err != nil {
			return nil, err
		}
	}
	for {
		if len(devices) == 0 {
			return nil, errNoDevFound
		}
		updateDevicesCache(cache, devices)
		showDevices(devices)
		idx, err := askDeviceIndex(len(devices) - 1)
		if err != nil {
			return nil, err
		}
		if idx < 0 { // search again
			if devices, err = searchDevices(); err != nil {
				return nil, err
			}
			continue
		}
		return devices[idx], nil
	}
}

func searchDevices() ([]*dial.Device, error) {
	devCh, err := dial.Discover(searchTimeout)
	if err != nil {
		return nil, err
	}
	var devices []*dial.Device
	for dev := range devCh {
		devices = append(devices, dev)
	}
	return devices, nil
}

func updateDevicesCache(cache map[string]*cacheEntry, devices []*dial.Device) {
	for _, dev := range devices {
		if entry, ok := cache[dev.UniqueServiceName]; ok {
			entry.Device = dev
		} else {
			cache[dev.UniqueServiceName] = &cacheEntry{Device: dev}
		}
	}
}

func showDevices(devices []*dial.Device) {
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].FriendlyName < devices[j].FriendlyName
	})
	for i, dev := range devices {
		host := dev.ApplicationUrl
		if u, err := url.Parse(dev.ApplicationUrl); err == nil {
			host = u.Hostname()
		}
		fmt.Printf("[%d] %-30s %s\n", i, dev.FriendlyName, host)
	}
}

func askYesNo(msg string, a ...interface{}) (bool, error) {
	for {
		ans, err := ask(msg, a...)
		if err != nil {
			return false, err
		}
		switch ans {
		case "", "y", "yes", "Y", "YES":
			return true, nil
		case "n", "no", "N", "NO":
			return false, nil
		}
	}
}

func ask(msg string, a ...interface{}) (string, error) {
	fmt.Printf(msg, a...)
	s := bufio.NewScanner(os.Stdin)
	s.Scan()
	return strings.TrimSpace(s.Text()), s.Err()
}

func askDeviceIndex(max int) (int, error) {
	for {
		input, err := ask("select device (default 0, R refresh, C cancel): ")
		if err != nil {
			return 0, err
		}
		switch input {
		case "":
			return 0, nil
		case "R", "r":
			return -1, nil
		case "C", "c":
			return 0, errCanceled
		default:
			if i, err := strconv.Atoi(input); err == nil && i >= 0 && i <= max {
				return i, nil
			}
		}
	}
}

func launchYouTubeApp(dev *dial.Device) (*dial.AppInfo, error) {
	for i := 0; i < 5; i++ {
		app, err := dev.GetAppInfo("YouTube", youtube.Origin)
		if err != nil {
			return nil, err
		}
		switch app.State {
		case "running":
			if screenId, _ := extractScreenId(app.Additional.Data); screenId != "" {
				return app, nil
			}
		case "stopped", "hidden":
			if _, err := dev.Launch("YouTube", youtube.Origin, ""); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown application state: %s", app.State)
		}
		time.Sleep(2 * time.Second)
	}
	return nil, errors.New("unable to start YouTube application")
}

func playVideos(entry *cacheEntry, app *dial.AppInfo, videoIds []string) error {
	screenId, err := extractScreenId(app.Additional.Data)
	if err != nil {
		return err
	}

	remote := entry.Remote
	if remote == nil || remote.ScreenId != screenId || remote.Expired() {
		if remote, err = youtube.Connect(screenId, progName); err != nil {
			return err
		}
		entry.Remote = remote
	}

	for i, v := range videoIds {
		videoIds[i] = extractVideoId(v)
	}
	return remote.Play(videoIds...)
}

func extractScreenId(data string) (string, error) {
	// TODO dial.AppInfo.Additional.Data it's not wrapped in a root element,
	// I add a dummy root here but I think data should already be wrapped in
	// a root element.
	data = fmt.Sprintf("<dummy>%s</dummy>", data)
	var v struct {
		ScreenId string `xml:"screenId"`
	}
	if err := xml.Unmarshal([]byte(data), &v); err != nil {
		return "", err
	}
	v.ScreenId = strings.TrimSpace(v.ScreenId)
	if len(v.ScreenId) == 0 {
		return "", errors.New("screenId empty")
	}
	return v.ScreenId, nil
}

func extractVideoId(v string) string {
	v = strings.TrimSpace(v)
	u, err := url.Parse(v)
	if err != nil {
		return v
	}
	vid := u.Query().Get("v")
	if len(vid) > 0 {
		return vid
	}
	if vid = path.Base(u.Path); len(vid) > 0 && vid != "." && vid != "/" {
		return vid
	}
	return v
}
