package main

import (
  "fmt"
  "math/rand"
  "regexp"
  "time"

  "github.com/nlopes/slack"
  "github.com/robfig/cron"
  "github.com/spf13/viper"

  "log"
)

const PRESENCE_ACTIVE = "active"

type Config struct {
  RotateFrequency string `mapstructure:"rotate_frequency"`
  PocMessagePattern string `mapstructure:"poc_message_pattern"`
  MessagePocChange string `mapstructure:"message_poc_change"`
  ExcludedRotators map[string]bool `mapstructure:"excluded_rotators"`
  IncludedRotators map[string]bool `mapstructure:"included_rotators"`
}

var (
  token string // set this in env
  api *slack.Client
  config Config
  rotators map[string][]string // rotators by channel
  rotated map[string]map[string]bool
)

func init() {
  viper := viper.New()
  viper.SetConfigFile("./config.json")
  viper.AutomaticEnv()
  viper.ReadInConfig()

  log.Printf("Using config: %s\n", viper.ConfigFileUsed())

  token = viper.GetString("POCMON_TOKEN")
  c := viper.Sub("config")
  err := c.Unmarshal(&config)
  if err != nil {
    log.Fatalf("unable to decode config into struct: %v", err)
  }
}

func main() {
  api = slack.New(token)
  rtm := api.NewRTM()
  go rtm.ManageConnection()

  rand.Seed(time.Now().Unix())
  rotators = make(map[string][]string)
  rotated = make(map[string]map[string]bool)

  cron := cron.New()
  cron.AddFunc(config.RotateFrequency, rotate)
  cron.Start()

  select { }
}

func rotate() {
  log.Print("⟳ Rotating...")
  channels, _ := api.GetChannels(false)
  for _, channel := range channels {
    if !channel.IsMember {
      continue
    }

    rotators[channel.Name] = getAvailableRotators(channel, false)

    rotator := rotators[channel.Name][0]
    if rotated[channel.Name] == nil {
      rotated[channel.Name] = make(map[string]bool)
    }
    rotated[channel.Name][rotator] = true

    updateChannelTopic(channel, rotator)
    sendMessageToRotator(channel, rotator)
  }
}

func getAvailableRotators(channel slack.Channel, replenish bool) []string {
  var rotators []string
  for _, member := range channel.Members {
    user, err := api.GetUserInfo(member)
    if err != nil {
      log.Fatalf("Error getting channel members: %v", err)
    }

    presence, err := api.GetUserPresence(member)
    if err != nil {
      log.Fatal("Cannot get User Presence")
    }

    if presence.Presence != PRESENCE_ACTIVE {
      continue
    }

    if len(config.IncludedRotators) != 0 {
      if !config.IncludedRotators[user.Name] {
        continue
      }
    }

    if config.ExcludedRotators[user.Name] {
      continue
    }

    if !replenish && rotated[channel.Name][user.Name] {
      continue
    }

    rotators = append(rotators, user.Name)
  }

  if rotators == nil {
    log.Print("... no rotators left!")
    rotated = make(map[string]map[string]bool) // reset rotated
    return getAvailableRotators(channel, true)
  }

  rotators = shuffleSlice(rotators)
  log.Printf("Rotators: %v\n", rotators)

  return rotators
}

func sendMessageToRotator(channel slack.Channel, rotator string) {
  _, _, errPostMessage := api.PostMessage(
    channel.Name,
    fmt.Sprintf(config.MessagePocChange, rotator, channel.Name),
    slack.PostMessageParameters{})
  if errPostMessage != nil {
    log.Fatal(errPostMessage)
  }
}

func updateChannelTopic(channel slack.Channel, rotator string) {
  var topic string

  r := regexp.MustCompile("%s")
  pocPattern := r.ReplaceAllString(config.PocMessagePattern, "[a-z]+")
  r = regexp.MustCompile(pocPattern)
  if r.MatchString(channel.Topic.Value) {
    topic = r.ReplaceAllString(channel.Topic.Value, fmt.Sprintf(config.PocMessagePattern, rotator))
  } else if channel.Topic.Value != "" {
    topic = fmt.Sprintf(channel.Topic.Value + " | " + config.PocMessagePattern, rotator)
  } else {
    topic = fmt.Sprintf(config.PocMessagePattern, rotator)
  }

  _, err := api.SetChannelTopic(channel.ID, topic)
  if err != nil {
    log.Fatal(err)
  }
}

func shuffleSlice(slice []string) []string {
  for i := range slice {
    j := rand.Intn(i + 1)
    slice[i], slice[j] = slice[j], slice[i]
  }
  return slice
}
