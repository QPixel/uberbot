package core

import (
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/QPixel/orderedmap"
	"github.com/ubergeek77/uberbot/internal"

	"github.com/bwmarrin/discordgo"
)

// commands.go
// This file contains everything required to add core commands to the bot, and parse commands from a message

// GroupTypes.
const (
	Moderation = "moderation"
	Utility    = "utility"
)

// CommandInfo
// The definition of a command's info. This is everything about the command, besides the function it will run.
type CommandInfo struct {
	Aliases     []string               // Aliases for the normal trigger
	Arguments   *orderedmap.OrderedMap // Arguments for the command
	Description string                 // A short description of what the command does
	Group       string                 // The group this command belongs to
	ParentID    string                 // The ID of the parent command
	Public      bool                   // Whether non-admins and non-mods can use this command
	IsTyping    bool                   // Whether the command will show a typing thing when ran.
	IsParent    bool                   // If the command is the parent of a subcommand tree
	IsChild     bool                   // If the command is the child
	Trigger     string                 // The string that will trigger the command
}

// CmdContext
// This is a context of a single command invocation
// This gives the command function access to all the information it might need.
type CmdContext struct {
	Guild       *Guild // NOTE: Guild is a pointer, since we want to use the SAME instance of the guild across the program!
	Cmd         CommandInfo
	Args        Arguments
	Message     *discordgo.Message // Technically deprecated, but still useful for message commands
	Interaction *discordgo.Interaction
}

// BotFunction
// This type defines the functions that are called when commands are triggered
// Contexts are also passed as pointers, so they are not re-allocated when passed through.
type BotFunction func(ctx *CmdContext)

// Command
// The definition of a command, which is that command's information, along with the function it will run.
type Command struct {
	Info     CommandInfo
	Function BotFunction
}

// ChildCommand
// Defines how child commands are stored.
type ChildCommand map[string]map[string]Command

// CustomCommand
// A type that defines a custom command.
type CustomCommand struct {
	Content     string // The content of the custom command. Custom commands are just special strings after all
	InvokeCount int64  // How many times the command has been invoked; int64 for easier use with json
	Public      bool   // Whether non-admins and non-mods can use this command
}

// commands
// All the registered core commands (not custom commands)
// This is private so that other commands cannot modify it.
var commands = make(map[string]Command)

// childCommands
// All the registered ChildCommands (SubCmdGrps)
// This is private so other commands cannot modify it.
var childCommands = make(ChildCommand)

// Command Aliases
// A map of aliases to command triggers.
var commandAliases = make(map[string]string)

// slashCommands
// All the registered core commands that are also slash commands
// This is also private so other commands cannot modify it.
var slashCommands = make(map[string]discordgo.ApplicationCommand)

// commandsGC.
var commandsGC = 0

// AddCommand
// Add a command to the bot.
func AddCommand(info *CommandInfo, function BotFunction) {
	// Add Trigger to the alias
	info.Aliases = append(info.Aliases, info.Trigger)
	// Build a Command object for this command
	command := Command{
		Info:     *info,
		Function: function,
	}
	// adds a alias to a map; command aliases are case-sensitive
	for _, alias := range info.Aliases {
		if _, ok := commandAliases[alias]; ok {
			Log.Errorf("Alias was already registered %s for command %s", alias, info.Trigger)
			continue
		}
		alias = strings.ToLower(alias)
		commandAliases[alias] = info.Trigger
	}
	// Add the command to the map; command triggers are case-insensitive
	commands[strings.ToLower(info.Trigger)] = command
}

// AddChildCommand
// Adds a child command to the bot.
func AddChildCommand(info *CommandInfo, function BotFunction) {
	// Build a Command object for this command
	command := Command{
		Info:     *info,
		Function: function,
	}
	parentID := strings.ToLower(info.ParentID)
	if childCommands[parentID] == nil {
		childCommands[parentID] = make(map[string]Command)
	}
	// Add the command to the map; command triggers are case-insensitive
	childCommands[parentID][command.Info.Trigger] = command
}

// AddSlashCommand
// Adds a slash command to the bot
// Allows for separation between normal commands and slash commands.
func AddSlashCommand(info *CommandInfo) {
	if !info.IsParent || !info.IsChild {
		s := createApplicationCommandStruct(info)
		slashCommands[strings.ToLower(info.Trigger)] = *s
		return
	}
	if info.IsParent {
		s := createChatInputSubCmdStruct(info, childCommands[info.Trigger])
		slashCommands[strings.ToLower(info.Trigger)] = *s
		return
	}
}

// RegisterSlashCommands
// Registers the slash commands. Called on the ready event
// defaults to registering commands globally, but it is dependent on the environment.
func RegisterSlashCommands() {
	// Grab our currently registered application commands
	currentCommands, err := Session.ApplicationCommands(Session.State.User.ID, "")
	if err != nil {
		Log.Errorf("unable to get current application commands")
		Log.Error(err.Error())
	}
	// If we get a response at all or if the environment is dev
	// register commands
	if len(currentCommands) >= 0 || IsDevEnv() {
		// Filter through our commands for UX based commands
		// TODO ADD new REGISTRATION LOGIC FOR UX COMMANDS
		commands := internal.Filter(currentCommands, func(item *discordgo.ApplicationCommand) bool {
			return item.Type != discordgo.ChatApplicationCommand
		})
		// add all slash commands to the existing commands slice
		for _, cmd := range slashCommands {
			setCmd := cmd
			commands = append(commands, &setCmd)
		}
		// if the environment is dev, this is running on the dev bot, which is only in a select few guilds
		// so lets just register commands in all guilds in the state
		if IsDevEnv() {
			Log.Infof("Setting slash commands in %d guilds", len(Session.State.Guilds))
			for _, guild := range Session.State.Guilds {
				updateCommands, err := Session.ApplicationCommandBulkOverwrite(Session.State.User.ID, guild.ID, commands)
				if err != nil {
					Log.Errorf("unable to bulk overwrite commands in guild %s (%s)", guild.Name, guild.ID)
					Log.Error(err.Error())
					return
				}
				if updateCommands != nil && len(updateCommands) >= 0 {
					Log.Infof("successfully bulk overwrote %d slash commands in %s (%s)", len(updateCommands), guild.Name, guild.ID)
				}
			}
		} else {
			// bulk register all application commands
			_, err = Session.ApplicationCommandBulkOverwrite(Session.State.User.ID, "", commands)
			if err != nil {
				Log.Error("Unable to register slash commands")
				Log.Error(err.Error())
			}
		}
	}
	return
}

// GetCommands
// Provide a way to read commands without making it possible to modify their functions.
func GetCommands() map[string]CommandInfo {
	list := make(map[string]CommandInfo)
	for x, y := range commands {
		list[x] = y.Info
	}
	return list
}

// customCommandHandler
// Given a custom command, interpret and run it.
func customCommandHandler(command CustomCommand, args []string, message *discordgo.Message) {
	//TODO
}

// commandHandler
// This handler will be added to a *discordgo.Session, and will scan an incoming messages for commands to run.
func commandHandler(session *discordgo.Session, message *discordgo.MessageCreate) {
	// Try getting an object for the current channel, with a fallback in case session.state is not ready or is nil
	channel, err := session.State.Channel(message.ChannelID)
	if err != nil {
		if channel, err = session.Channel(message.ChannelID); err != nil {
			return
		}
	}

	// Ignore messages sent by the bot
	if message.Author.ID == session.State.User.ID {
		return
	}

	g := GetGuild(message.GuildID)

	trigger, argString := ExtractCommand(&g.Info, message.Content)
	if trigger == nil {
		return
	}
	//isCustom := false
	//if _, ok := commands[commandAliases[*trigger]]; !ok {
	//	if !g.IsCustomCommand(*trigger) {
	//		return
	//	} else {
	//		isCustom = true
	//	}
	//}
	//// Only do further checks if the user is not a bot admin
	//if !IsAdmin(message.Author.ID) {
	//	// Ignore the command if it is globally disabled
	//	if g.IsGloballyDisabled(*trigger) {
	//		return
	//	}
	//
	//	// Ignore the command if this channel has blocked the trigger
	//	if g.TriggerIsDisabledInChannel(*trigger, message.ChannelID) {
	//		return
	//	}
	//
	//	// Ignore any message if the user is banned from using the bot
	//	if !g.MemberOrRoleIsWhitelisted(message.Author.ID) || g.MemberOrRoleIsIgnored(message.Author.ID) {
	//		return
	//	}
	//
	//	// Ignore the message if this channel is not whitelisted, or if it is ignored
	//	if !g.ChannelIsWhitelisted(message.ChannelID) || g.ChannelIsIgnored(message.ChannelID) {
	//		return
	//	}
	//}

	//if !isCustom {
	//Get the command to run
	// Error Checking
	command, ok := commands[commandAliases[*trigger]]
	if !ok {
		Log.Errorf("Command was not found")
		if IsAdmin(message.Author.ID) {
			Session.MessageReactionAdd(message.ChannelID, message.ID, "<:redtick:861413502991073281>")
			Session.ChannelMessageSendReply(message.ChannelID, "<:redtick:861413502991073281> Error! Command not found!", message.MessageReference)
		}
		return
	}
	// Check if the command is public, or if the current user is a bot moderator
	// Bot admins supercede both checks
	//if IsAdmin(message.Author.ID) || command.Info.Public || g.IsMod(message.Author.ID) {
	// Run the command with the necessary context
	if command.Info.IsTyping && g.Info.ResponseChannelID == "" {
		_ = Session.ChannelTyping(message.ChannelID)
	}
	// The command is valid, so now we need to delete the invoking message if that is configured
	//if g.Info.DeletePolicy {
	//	err := Session.ChannelMessageDelete(message.ChannelID, message.ID)
	//	if err != nil {
	//		SendErrorReport(message.GuildID, message.ChannelID, message.Author.ID, "Failed to delete message: "+message.ID, err)
	//	}
	//}

	defer handleCommandError(g.ID, channel.ID, message.Author.ID)
	if command.Info.IsParent {
		handleChildCommand(*argString, command, message.Message, g)
		return
	}
	command.Function(&CmdContext{
		Guild:   g,
		Cmd:     command.Info,
		Args:    *ParseArguments(*argString, command.Info.Arguments),
		Message: message.Message,
	})
	// Makes sure that variables ran in ParseArguments are gone.
	if commandsGC == 25 && commandsGC > 25 {
		debug.FreeOSMemory()
		commandsGC = 0
	} else {
		commandsGC++
	}
	return
	//}
	//}
}

// -- Helper Methods.
func handleChildCommand(argString string, command Command, message *discordgo.Message, guild *Guild) {
	split := strings.SplitN(argString, " ", 2)

	childCmd, ok := childCommands[command.Info.Trigger][split[0]]
	if !ok {
		command.Function(&CmdContext{
			Guild:   guild,
			Cmd:     command.Info,
			Args:    nil,
			Message: message,
		})
		return
	}
	if len(split) < 2 {
		childCmd.Function(&CmdContext{
			Guild:   guild,
			Cmd:     childCmd.Info,
			Args:    *ParseArguments("", childCmd.Info.Arguments),
			Message: message,
		})
		return
	}
	childCmd.Function(&CmdContext{
		Guild:   guild,
		Cmd:     childCmd.Info,
		Args:    *ParseArguments(split[1], childCmd.Info.Arguments),
		Message: message,
	})
	return
}

func handleCommandError(gID string, cId string, uId string) {
	if r := recover(); r != nil {
		Log.Warningf("Recovering from panic: %s", r)
		Log.Warningf("Sending Error report to admins")
		SendErrorReport(gID, cId, uId, "Error!", r.(runtime.Error))
		message, err := Session.ChannelMessageSend(cId, "Error!")
		if err != nil {
			Log.Errorf("err sending message %s", err)
		}
		time.Sleep(5 * time.Second)
		_ = Session.ChannelMessageDelete(cId, message.ID)
		return
	}
	return
}
