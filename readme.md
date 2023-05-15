# uberbot

The bot that watches over the Ottomated discord server.

## Description

uberbot is a custom bot written in discordgo. It has a custom data handler, command handler, and argument parser. 
Some main features of uberbot are moderation and meme like functions

## Dependencies
* Go 1.18
* OrderedMap (being deprecated)
* godotenv
* [tinylog](https://github.com/ubergeek77/tinylog)

## Development

> While the guild storage mechanism is abstracted for other storage types (DBs). I have no plans to include database support in the V2 of uberbot.

> Windows support does not exist on this version of uberbot. This is a deliberate choice, as the core maintainers do not use windows to develop on. Please use WSL or some POSIX (or semi posix) based OS in order to develop and run the bot.

### Setup
Make sure you have an ssh key attached to github and have configured it with git. Please also have a gpg key that works with git and can sign commits.

1. Fork the repository
2. Clone your fork of the repository
```sh
git clone git@github.com:<username>/uberbot.git
cd uberbot
```
3. Resolve dependencies
```sh
go mod tidy
```
4. Build uberbot
```shell
cmd/uberbot/build.sh
```
5. Create a .env file in the root directory and add these lines to it
```shell
UBERBOT_TOKEN=<discordtoken>
ADMIN_IDS=<yourdiscordid>
```
6. Run uberbot
```shell
cmd/uberbot/uberbot
```

## Contributing
Any contributions you make are greatly appreciated.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request
