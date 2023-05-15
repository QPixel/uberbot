package core

import (
	"github.com/bwmarrin/discordgo"
	"github.com/ubergeek77/uberbot/internal"
	"runtime"
	"strings"
)

// -- Types and Structs --

// applicationCommandTypes
// A map of *short hand* slash commands types to their discordgo counterparts
var applicationCommandTypes = map[ArgTypeGuards]discordgo.ApplicationCommandOptionType{
	Int:       discordgo.ApplicationCommandOptionInteger,
	String:    discordgo.ApplicationCommandOptionString,
	Channel:   discordgo.ApplicationCommandOptionChannel,
	User:      discordgo.ApplicationCommandOptionUser,
	Role:      discordgo.ApplicationCommandOptionRole,
	Boolean:   discordgo.ApplicationCommandOptionBoolean,
	SubCmd:    discordgo.ApplicationCommandOptionSubCommand,
	SubCmdGrp: discordgo.ApplicationCommandOptionSubCommandGroup,
}

// todo add documentation

type InteractionInfo struct {
	Id string
}

type InteractionCtx struct {
	*discordgo.InteractionCreate
	Session *discordgo.Session
	Info    InteractionInfo
}

type InteractionFunc func(ctx *InteractionCtx)

type InteractionHandler struct {
	Info     InteractionInfo
	Function InteractionFunc
}

var interactionHandlers = make(map[string]InteractionHandler)

// AddInteractHandler
// Add a interaction handler to the bot
func AddInteractHandler(info *InteractionInfo, function InteractionFunc) {
	interact := InteractionHandler{
		Info:     *info,
		Function: function,
	}
	interactionHandlers[strings.ToLower(info.Id)] = interact
}

// createApplicationCommandStruct
// Creates a slash command struct
// todo work on sub command stuff.
func createApplicationCommandStruct(info *CommandInfo) (st *discordgo.ApplicationCommand) {
	if info.Arguments == nil || len(info.Arguments.Keys()) < 1 {
		st = &discordgo.ApplicationCommand{
			Name:        info.Trigger,
			Description: info.Description,
		}
		return
	}
	st = &discordgo.ApplicationCommand{
		Name:        info.Trigger,
		Description: info.Description,
		Options:     make([]*discordgo.ApplicationCommandOption, len(info.Arguments.Keys())),
	}
	for i, k := range info.Arguments.Keys() {
		v, _ := info.Arguments.Get(k)
		vv := v.(*ArgInfo)
		var sType discordgo.ApplicationCommandOptionType
		if val, ok := applicationCommandTypes[vv.TypeGuard]; ok {
			sType = val
		} else {
			sType = applicationCommandTypes["String"]
		}
		optionStruct := discordgo.ApplicationCommandOption{
			Type:        sType,
			Name:        k,
			Description: vv.Description,
			Required:    vv.Required,
		}
		if vv.Choices != nil {
			optionStruct.Choices = make([]*discordgo.ApplicationCommandOptionChoice, len(vv.Choices))
			for i, k := range vv.Choices {
				optionStruct.Choices[i] = &discordgo.ApplicationCommandOptionChoice{
					Name:  k,
					Value: k,
				}
			}
		}
		st.Options[i] = &optionStruct
	}
	return st
}

// Creates a chatinput subcmd struct.
func createChatInputSubCmdStruct(info *CommandInfo, childCmds map[string]Command) (st *discordgo.ApplicationCommand) {
	st = &discordgo.ApplicationCommand{
		Name:        info.Trigger,
		Description: info.Description,
		Options:     make([]*discordgo.ApplicationCommandOption, len(childCmds)),
	}
	currentPos := 0
	for _, v := range childCmds {
		// Stupid inline thing
		if ar, _ := v.Info.Arguments.Get(v.Info.Arguments.Keys()[0]); ar.(*ArgInfo).TypeGuard == SubCmdGrp {

		} else {
			//Pixel:
			//Yes I know this is O(N^2). Most likely I could get something better
			//todo: refactor so this isn't as bad for performance
			st.Options[currentPos] = v.Info.CreateAppOptSt()
			currentPos++
		}
	}
	return st
}

// -- Interaction Handlers --

// handleInteraction
// Handles a slash command interaction.
func handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleInteractionCommand(s, i)
		break
	case discordgo.InteractionMessageComponent:
		handleMessageComponents(s, i)
	}
	return
}

// handleInteractionCommand
// Handles a slash command.
func handleInteractionCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	g := GetGuild(i.GuildID)

	trigger := i.ApplicationCommandData().Name
	//	// Ignore the command if it is globally disabled
	//	if g.IsGloballyDisabled(trigger) {
	//		ErrorResponse(i.Interaction, "Command is globally disabled", trigger)
	//		return
	//	}
	//
	//	// Ignore the command if this channel has blocked the trigger
	//	if g.TriggerIsDisabledInChannel(trigger, i.ChannelID) {
	//		ErrorResponse(i.Interaction, "Command is disabled in this channel!", trigger)
	//		return
	//	}
	//
	//	// Ignore any message if the user is banned from using the bot
	//	if !g.MemberOrRoleIsWhitelisted(i.Member.User.ID) || g.MemberOrRoleIsIgnored(i.Member.User.ID) {
	//		return
	//	}
	//
	//	// Ignore the message if this channel is not whitelisted, or if it is ignored
	//	if !g.ChannelIsWhitelisted(i.ChannelID) || g.ChannelIsIgnored(i.ChannelID) {
	//		return
	//	}

	command := commands[trigger]
	if IsAdmin(i.Member.User.ID) || command.Info.Public || g.IsMod(i.Member.User.ID) {
		// Check if the command is public, or if the current user is a bot moderator
		// Bot admins supercede both checks

		defer handleInteractionError(*i.Interaction)
		command.Function(&CmdContext{
			Guild:       g,
			Cmd:         command.Info,
			Args:        *ParseInteractionArgs(i.ApplicationCommandData().Options),
			Interaction: i.Interaction,
			Message: &discordgo.Message{
				Member:    i.Member,
				Author:    i.Member.User,
				ChannelID: i.ChannelID,
				GuildID:   i.GuildID,
				Content:   "",
			},
		})
		return
	}
}

func handleMessageComponents(s *discordgo.Session, i *discordgo.InteractionCreate) {
	handlerName := i.MessageComponentData().CustomID
	handler, ok := interactionHandlers[handlerName]
	if !ok {
		handleInteractionError(*i.Interaction)
	}

	defer handleInteractionError(*i.Interaction)
	handler.Function(&InteractionCtx{
		Info:              handler.Info,
		InteractionCreate: i,
		Session:           s,
	})
}

// -- Slash Argument Parsing Helpers --

// ParseInteractionArgs
// Parses Interaction args.
func ParseInteractionArgs(options []*discordgo.ApplicationCommandInteractionDataOption) *map[string]CommandArg {
	var args = make(map[string]CommandArg)
	for _, v := range options {
		args[v.Name] = CommandArg{
			info:  ArgInfo{},
			Value: v.Value,
		}
		if v.Options != nil {
			ParseInteractionArgsR(v.Options, &args)
		}
	}
	return &args
}

// ParseInteractionArgsR
// Parses interaction args recursively.
func ParseInteractionArgsR(options []*discordgo.ApplicationCommandInteractionDataOption, args *map[string]CommandArg) {
	for _, v := range options {
		(*args)[v.Name] = CommandArg{
			info:  ArgInfo{},
			Value: v.Value,
		}
		if v.Options != nil {
			ParseInteractionArgsR(v.Options, *&args)
		}
	}
}

// -- :shrug: --

// DeleteGuildApplicationCommands
// Removes all guild slash commands.
func DeleteGuildApplicationCommands(guildID string) {
	commands, err := Session.ApplicationCommands(Session.State.User.ID, guildID)
	if err != nil {
		Log.Errorf("Error getting all slash commands %s", err)
		return
	}
	for _, k := range commands {
		err = Session.ApplicationCommandDelete(Session.State.User.ID, guildID, k.ID)
		if err != nil {
			Log.Errorf("error deleting slash command %s %s %s", k.Name, k.ID, err)
			continue
		}
	}
}

func handleInteractionError(i discordgo.Interaction) {
	if r := recover(); r != nil {
		Log.Warningf("Recovering from panic: %s", r)
		Log.Warningf("Sending Error report to admins")
		SendErrorReport(i.GuildID, i.ChannelID, i.Member.User.ID, "Error!", r.(runtime.Error))
		message, err := Session.InteractionResponseEdit(&i, &discordgo.WebhookEdit{
			Content: internal.ToPtr("error executing command"),
		})
		if err != nil {
			Log.Errorf("err sending message %s", err)
			err = Session.InteractionRespond(&i, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Flags:   1 << 6,
					Content: "error executing command",
				},
			})
			Log.Errorf("err responding to interaction %s", err.Error())
		}
		err = Session.ChannelMessageDelete(i.ChannelID, message.ID)
		if err != nil {
			Log.Errorf("unable to delete message %s", err.Error())
		}
		return
	}
	return
}
