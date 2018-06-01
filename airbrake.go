package airbrake // import "github.com/assembla/logrus-airbrake-hook"

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	"github.com/airbrake/gobrake"
	"github.com/sirupsen/logrus"
)

// AirbrakeHook to send exceptions to an exception-tracking service compatible
// with the Airbrake API.
type airbrakeHook struct {
	Airbrake *gobrake.Notifier
}

func NewHook(projectID int64, apiKey, env string) *airbrakeHook {
	return NewCustomHook(projectID, apiKey, env, "")
}

func NewCustomHook(projectID int64, apiKey, env, host string) *airbrakeHook {
	airbrake := gobrake.NewNotifier(projectID, apiKey)
	if host != "" {
		airbrake.SetHost(host)
	}

	airbrake.AddFilter(func(notice *gobrake.Notice) *gobrake.Notice {
		if env == "development" {
			return nil
		}
		notice.Context["environment"] = env
		return notice
	})
	hook := &airbrakeHook{
		Airbrake: airbrake,
	}
	return hook
}

func (hook *airbrakeHook) Fire(entry *logrus.Entry) error {
	var notifyErr error
	err, ok := entry.Data["error"].(error)
	if ok {
		notifyErr = err
	} else {
		notifyErr = errors.New(entry.Message)
	}
	var req *http.Request
	for k, v := range entry.Data {
		if r, ok := v.(*http.Request); ok {
			req = r
			delete(entry.Data, k)
			break
		}
	}
	notice := hook.Airbrake.Notice(notifyErr, req, 3)
	for k, v := range entry.Data {
		notice.Context[k] = fmt.Sprintf("%s", v)
	}

	hook.sendNotice(notice)
	return nil
}

func (hook *airbrakeHook) sendNotice(notice *gobrake.Notice) {
	if _, err := hook.Airbrake.SendNotice(notice); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to send error to Airbrake: %v\n", err)
	}
}

func (hook *airbrakeHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.FatalLevel,
		logrus.PanicLevel,
	}
}
