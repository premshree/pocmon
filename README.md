[![Build Status](https://travis-ci.org/premshree/pocmon.svg?branch=master)](https://travis-ci.org/premshree/pocmon)

# What is pocmon?

pocmon is a a Slack Bot that runs in the background in any channel you invite it to and periodically picks someone in the channel to be a "Point Of Contact" (or, POC for short) by adding that information to the channel's Topic and letting that user know that they are now a POC.

The main motivations behind pocmon were:
- facilitate inter-team communication by making it easy for external teams to know who to ask for help in a channel
- not have any one person be overwhelmed by messages

# How It Works

pocmon is very simple. Under the hood it does a few things:
- It keeps track of all the channels it is in and a list of rotators for each channel during each "run"
- It keeps track of people who have already been a POC so they are not picked again
- Amongst the "available rotators", it shuffles the list and then picks the first person from that list
- When there are no more rotators available, it will "replenish" the list by clearing out the already-rotated list.

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

# Adding pocmon to your Slack channel

pocmon works only as a custom [bot user](https://api.slack.com/bot-users) (Slack's API won't allow the `channels:write` OAuth scope for app bots.)
- Go ahead and [create a new bot user](https://my.slack.com/services/new/bot) for your team. Let's assume you called it `@pocmon`
- Note the API Token for your custom bot.

# Installation

Before you are ready to run pocmon, you'll need a Slack API token for your bot user. See above.

```
go get github.com/premshree/pocmon
export POCMON_TOKEN=YOUR-SLACK-TOKEN
go run main.go
```

# Deploying pocmon to Heroku

Deploying your bot to Heroku is seamless.

### Using Heroku Git
```
$ git clone git@github.com:premshree/pocmon.git
$ heroku login
$ heroku create
$ heroku config:set POCMON_TOKEN=YOUR-SLACK-TOKEN
$ git push heroku master
$ heroku ps:scale worker=1
```

### Using Github

- Fork [this repo](https://github.com/premshree/pocmon)
- [Create a new app](https://dashboard.heroku.com/new-app) on Heroku
- In your app dashboard, use "Github" as your Deployment method. Github will want to authorize access to your repo to Heroku.
- Look for the "Deploy a Github Branch" in your Heroku app dashboard and click "Deploy Branch". Et voila!

(For those magic-averse folks like me, how the hell does this work? pocmon has a Heroku `Procfile` defined and uses [godep](https://github.com/tools/godep) to manage dependencies. Heroku's [Go buildpack](https://github.com/heroku/heroku-buildpack-go) detects all of this and knows how to run your app.)

# Running pocmon in your Slack Channel

Invite pocmon to your channel on Slack:
```
/invite @pocmon
```

# TBD
- Allow configuring "business hours" when pocmon sleeps.
- Set config by talking to @pocmon directly?
