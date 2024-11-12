package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"discordtasks.app/internal/discordwebhook"
	"github.com/apognu/gocal"
	"github.com/go-co-op/gocron/v2"
	"github.com/studio-b12/gowebdav"
)

func printErr(err error) {
	fmt.Println(err.Error())
	debug.PrintStack()
}

type Calendar struct {
	Name    string
	Webhook string
}

type CalendarEvent struct {
	Events   []gocal.Event
	Calendar *Calendar
}

func waitOnShutdown(cb func(context.Context)) {
	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal)
	// kill (no param) default send syscanll.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall. SIGKILL but can"t be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cb(ctx)

	// catching ctx.Done(). timeout of 5 seconds.
	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
}

func onEvent(e *gocal.Event, cal *Calendar) {
	fmt.Println(e.Summary, cal.Name)
	username := fmt.Sprintf("Calendar bot %s", cal.Name)
	content := fmt.Sprintf("Event: %s", e.Summary)
	discordwebhook.SendMessage(cal.Webhook, discordwebhook.Message{
		Username: &username,
		Content:  &content,
	})
}

func readEventsFromNextCloud(user string, password string, root string, calendars []Calendar) ([]CalendarEvent, error) {
	cc := gowebdav.NewClient(root, user, password)
	err := cc.Connect()
	if err != nil {
		return nil, err
	}

	eventsMap := []CalendarEvent{}
	for i, c := range calendars {
		eventsMap = append(eventsMap, CalendarEvent{
			Events:   []gocal.Event{},
			Calendar: &c,
		})
		path := "/" + c.Name
		files, err := cc.ReadDir(path)
		if err != nil {
			printErr(err)
			continue
		}
		for _, f := range files {
			fpath := filepath.Join(path, f.Name())
			b, err := cc.Read(fpath)
			if err == nil {
				buffer := bytes.NewReader(b)

				start, end := time.Now(), time.Now().Add(12*30*24*time.Hour)
				cal := gocal.NewParser(buffer)

				cal.Start, cal.End = &start, &end
				cal.Parse()

				// filter out old events
				events := []gocal.Event{}
				for _, e := range cal.Events {
					if e.Start == nil || time.Now().After(*e.Start) {
						continue
					}
					events = append(events, e)
				}

				eventsMap[i] = CalendarEvent{
					Events:   append(eventsMap[i].Events, events...),
					Calendar: &c,
				}

			} else {
				if err != nil {
					printErr(err)
					continue
				}
			}
		}
	}
	return eventsMap, nil
}

func parseCalendars(calendarsEnv string, webhooksEnv string) []Calendar {
	calendarsS := strings.Split(calendarsEnv, ",")
	webhooksS := strings.Split(webhooksEnv, ",")
	calendars := []Calendar{}
	for _, c := range calendarsS {

		calS := strings.Split(c, "|")
		if len(calS) != 2 {
			continue
		}
		webhook := ""

		for _, w := range webhooksS {
			wS := strings.Split(w, "|")
			if len(wS) != 2 {
				continue
			}

			if wS[0] == calS[1] {
				webhook = wS[1]
			}
		}

		if webhook == "" {
			continue
		}
		calendars = append(calendars, Calendar{
			Name:    calS[0],
			Webhook: webhook,
		})
	}
	return calendars
}

func main() {
	running := true
	user := os.Getenv("NEXTCLOUD_USER")
	password := os.Getenv("NEXTCLOUD_PASSWORD")
	root := fmt.Sprintf(os.Getenv("NEXTCLOUD_SERVER")+"/remote.php/dav/calendars/%s/", user)
	calendarsEnv := os.Getenv("NEXTCLOUD_CALENDARS")
	webhooks := os.Getenv("WEBHOOKS")

	calendars := parseCalendars(calendarsEnv, webhooks)

	go func() {
		for running {
			events, err := readEventsFromNextCloud(user, password, root, calendars)
			if err != nil {
				fmt.Println(err)
			}
			s, err := gocron.NewScheduler()
			if err != nil {
				panic(err.Error())
			}

			for _, e := range events {
				for _, ce := range e.Events {
					_, err := s.NewJob(
						gocron.OneTimeJob(gocron.OneTimeJobStartDateTime(*ce.Start)),
						gocron.NewTask(
							func(ce *gocal.Event, cal *Calendar) {
								onEvent(ce, cal)
							},
							&ce,
							e.Calendar,
						),
					)
					if err != nil {
						printErr(err)
						continue
					}

				}

			}
			s.Start()

			fmt.Println("Refreshed events")
			time.Sleep(10 * time.Minute)
			for _, j := range s.Jobs() {
				s.RemoveJob(j.ID())
			}
		}
	}()

	waitOnShutdown(func(ctx context.Context) {
		running = false
	})

}