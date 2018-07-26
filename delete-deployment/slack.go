package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mllu/k8s-executor/config"
	"github.com/nlopes/slack"
)

var slackColors = map[string]string{
	"Normal":  "good",
	"Warning": "warning",
	"Danger":  "danger",
}

var slackErrMsg = `
%s

You need to set both slack token and channel for slack notify,
using "--token/-t" and "--channel/-c", or using environment variables:

export KW_SLACK_TOKEN=slack_token
export KW_SLACK_CHANNEL=slack_channel

Command line flags will override environment variables

`

// Slack handler implements handler.Handler interface,
// Notify event to slack channel
type Slack struct {
	Token   string
	Channel string
}

// InitSlack prepares slack configuration
func InitSlack(c *config.Config) (s *Slack, err error) {
	s = &Slack{}
	token := c.Token
	channel := c.Channel

	log.Println("initializing slack...")

	if token == "" {
		token = os.Getenv("SLACK_TOKEN")
	}

	if channel == "" {
		channel = os.Getenv("SLACK_CHANNEL")
	}

	s.Token = token
	s.Channel = channel

	return s, s.checkMissingSlackVars()
}

func (s *Slack) checkMissingSlackVars() error {
	if s.Token == "" || s.Channel == "" {
		return fmt.Errorf(slackErrMsg, "Missing slack token or channel")
	}

	return nil
}

func (s *Slack) notifySlack(namespace, name, reason, action string) {
	e := NewKubeEvent(namespace, name, reason, action)
	api := slack.New(s.Token)
	params := slack.PostMessageParameters{}
	attachment := prepareSlackAttachment(e)

	params.Attachments = []slack.Attachment{attachment}
	params.AsUser = true
	channelID, timestamp, err := api.PostMessage(s.Channel, "", params)
	if err != nil {
		log.Printf("%s\n", err)
		return
	}

	log.Printf("Message successfully sent to channel %s at %s", channelID, timestamp)
}

func prepareSlackAttachment(e KubeEvent) slack.Attachment {

	attachment := slack.Attachment{
		Fields: []slack.AttachmentField{
			slack.AttachmentField{
				Title: "kubewatch",
				Value: e.Message(),
			},
		},
	}

	if color, ok := slackColors[e.Status]; ok {
		attachment.Color = color
	}

	attachment.MarkdownIn = []string{"fields"}

	return attachment
}

// KubeEvent represent an event got from k8s api server
// Events from different endpoints need to be casted to KubeEvent
// before being able to be handled by handler
type KubeEvent struct {
	Namespace string
	Name      string
	Reason    string
	Action    string
	Status    string
}

var m = map[string]string{
	"created": "Normal",
	"deleted": "Danger",
	"updated": "Warning",
}

// NewKubeEvent create new KubeEvent
func NewKubeEvent(namespace, name, reason, action string) KubeEvent {
	status := m[action]

	kubeEvent := KubeEvent{
		Namespace: namespace,
		Name:      name,
		Reason:    reason,
		Action:    action,
		Status:    status,
	}
	return kubeEvent
}

// Message returns event message in standard format.
// included as a part of event packege to enhance code resuablity across handlers.
func (e *KubeEvent) Message() (msg string) {
	msg = fmt.Sprintf(
		"`%s` in namespace `%s` has been `%s` due to `%s`\n",
		e.Name,
		e.Namespace,
		e.Action,
		e.Reason,
	)
	return msg
}
