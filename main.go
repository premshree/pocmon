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

const (
  PRESENCE_ACTIVE = "active"
  POCMON_ENV_PREFIX = "pocmon"
)

type Config struct {
  TimeZone string `mapstructure:"timezone"`
  Channels []ChannelConfig `mapstructure:"channels"`
}

type SlackChannel slack.Channel

type ChannelConfig struct {
  Name string `mapstructure:"name"`
  RotateFrequency string `mapstructure:"rotate_frequency"`
  PocMessagePattern string `mapstructure:"poc_message_pattern"`
  MessagePocChange string `mapstructure:"message_poc_change"`
  ExcludedRotators map[string]bool `mapstructure:"excluded_rotators"`
  IncludedRotators map[string]bool `mapstructure:"included_rotators"`
}

var (
  token string // set this in env
  api *slack.Client
  channels map[string]slack.Channel
  config Config
  channelConfigMap map[string]ChannelConfig
  rotators map[string][]string // rotators by channel
  rotated map[string]map[string]bool
)

func init() {
  viper := viper.New()
  viper.SetConfigFile("./config.json")
  viper.SetEnvPrefix(POCMON_ENV_PREFIX)
  viper.AutomaticEnv()
  viper.ReadInConfig()

  log.Printf("Using config: %s\n", viper.ConfigFileUsed())

  token = viper.GetString("TOKEN")
  err := viper.Unmarshal(&config)
  if err != nil {
    log.Fatalf("unable to decode config into struct: %v", err)
  }

  channelConfigMap = getChannelConfigMap()
}

func main() {
  api = slack.New(token)
  rtm := api.NewRTM()
  go rtm.ManageConnection()

  rand.Seed(time.Now().Unix())

  rotators = make(map[string][]string)
  rotated = make(map[string]map[string]bool)
  channels = getAllChannels()

  location, err := time.LoadLocation(config.TimeZone)
  if err != nil {
    log.Fatalf("Uh oh, error getting location for timezone %s", config.TimeZone)
  }
  cron := cron.NewWithLocation(location)
  for _, channel := range config.Channels {
    cron.AddFunc(channel.RotateFrequency, rotate(channel.Name))
  }
  cron.Start()

  select { }
}

func rotate(channelName string) func() {
  return func() {
    if _, ok := channels[channelName]; !ok {
      log.Fatalf("Uh oh, #%v not found", channelName)
    }

    channel := channels[channelName]
    if !channel.IsMember {
      log.Fatalf("Uh oh, @pocmon is not in #%v", channelName)
    }

    log.Printf("⟳ Rotating POC for #%v ...", channelName)

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

    if len(channelConfigMap[channel.Name].IncludedRotators) != 0 {
      if !channelConfigMap[channel.Name].IncludedRotators[user.Name] {
        continue
      }
    }

    if channelConfigMap[channel.Name].ExcludedRotators[user.Name] {
      continue
    }

    if !replenish && rotated[channel.Name][user.Name] {
      continue
    }

    rotators = append(rotators, user.Name)
  }

  if rotators == nil {
    log.Printf("... #%v has no rotators left!", channel.Name)
    rotated[channel.Name] = nil
    return getAvailableRotators(channel, true)
  }

  rotators = shuffleSlice(rotators)
  log.Printf("Rotators for #%v: %v [refresh:%v]\n", channel.Name, rotators, replenish)

  return rotators
}

func sendMessageToRotator(channel slack.Channel, rotator string) {
  _, _, errPostMessage := api.PostMessage(
    channel.Name,
    fmt.Sprintf(channelConfigMap[channel.Name].MessagePocChange, rotator, channel.Name),
    slack.PostMessageParameters{})
  if errPostMessage != nil {
    log.Fatal(errPostMessage)
  }
}

func updateChannelTopic(channel slack.Channel, rotator string) {
  var topic string

  r := regexp.MustCompile("%s")
  pocPattern := r.ReplaceAllString(channelConfigMap[channel.Name].PocMessagePattern, "[a-z-]+")
  r = regexp.MustCompile(pocPattern)
  if r.MatchString(channel.Topic.Value) {
    topic = r.ReplaceAllString(channel.Topic.Value, fmt.Sprintf(channelConfigMap[channel.Name].PocMessagePattern, rotator))
  } else if channel.Topic.Value != "" {
    topic = fmt.Sprintf(channel.Topic.Value + " | " + channelConfigMap[channel.Name].PocMessagePattern, rotator)
  } else {
    topic = fmt.Sprintf(channelConfigMap[channel.Name].PocMessagePattern, rotator)
  }

  _, err := api.SetChannelTopic(channel.ID, topic)
  if err != nil {
    log.Fatal(err)
  }
}

func getAllChannels() map[string]slack.Channel {
  allChannels, err := api.GetChannels(false)
  if err != nil {
    log.Fatalf("Uh oh, error fetching channels: %v", err)
  }
  channelsMap := make(map[string]slack.Channel)
  for _, channel := range allChannels {
    channelsMap[channel.Name] = channel
  }

  return channelsMap
}

func getChannelConfigMap() map[string]ChannelConfig {
  channelConfigMap := make(map[string]ChannelConfig)
  for _, channel := range config.Channels {
    channelConfigMap[channel.Name] = channel
  }

  return channelConfigMap
}

func shuffleSlice(slice []string) []string {
  for i := range slice {
    j := rand.Intn(i + 1)
    slice[i], slice[j] = slice[j], slice[i]
  }
  return slice
}
