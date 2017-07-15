# What is pocmon?

pocmon is a a Slack Bot that runs in the background in any channel you invite it to and periodically picks someone to be a "Point Of Contact" (or, POC for short) by adding that information to the channel's Topic and letting that user know that they are now a POC.

The main motivations behind pocmon were:
- facilitate inter-team communication by making it easy for external teams to know who to ask for help in a channel
- not have any one person be overwhelmed by messages

# Configuration

`config.json` has some basic configuration you can customize:
- `rotate_frequency` crontab-style configuration for the frequency at which you want pocmon to update a POC. pocmon uses [cron](https://github.com/robfig/cron), whose syntax varies slightly from the standard [crontab expression](https://en.wikipedia.org/wiki/Cron#CRON_expression):

```
* * * * * *
↑ ↑ ↑ ↑ ↑ ↑
| | | | | |
| | | | | +-- Day of week       (range: 1-7, Monday-Sunday)
| | | | +---- Month             (range: 1-12)
| | | +------ Day of the Month  (range: 1-12)
| | +-------- Hour              (range: 1-23)
| +---------- Minute            (range: 0-59)
+------------ Second            (range: 0-59)
```
- `poc_message_pattern` This is what the POC message will look like in the channel topic. If a POC message with the same pattern as that defined int he config is set in the channel topic, pocmon will simply update the topic. If the channel doesn't have a POC set using the message pattern defined in config, pocmon will append the POC message pattern to the existing topic using a `|`.
- `message_poc_change` When pocmon designates a new POC, it will send a message to that user in the channel.
- `excluded_rotators` A map of usernames you want to exclude from being rotated as a POC. Example:
```json
"excluded_rotators": {
    "johndoe": true
}
```


# Installation

```
go get github.com/premshree/pocmon
go run main.go
```
