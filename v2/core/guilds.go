package core

import (
	"github.com/bwmarrin/discordgo"
	"sync"
	"time"
)

// todo finish lmao

// guilds.go
// This file contains the structure of a guild, and all the functions used to store and retrieve guild information

// GuildInfo
// This is all the settings and data that needs to be stored about a single guild.
type GuildInfo struct {
	AddedDate         int64    // The date the bot was added to the server
	AllowedUsageIDs   []string `json:"whitelistIds"` // List of user/role Ids that a user MUST have one of in order to run any commands, including public ones
	Prefix            string   // The bot prefix
	ModeratorIDs      []string // The list of user/role IDs allowed to run mod-only commands
	ResponseChannelID string
}

// NewGuildInfo
// Returns a default guild info.
func NewGuildInfo() GuildInfo {
	return GuildInfo{
		AddedDate:         time.Now().Unix(),
		Prefix:            "!",
		ResponseChannelID: "",
	}
}

// Guild
// The definition of a bot guild, which includes a pointer to the discordgo.Guild,
// it's id, and guild storage (info).
type Guild struct {
	*discordgo.Guild
	Info         GuildInfo
	RegisteredAt time.Time
}

// Guilds
// A map that stores the data for all known guilds
// We store pointers to the guilds, so that only one guild object is maintained across all contexts
// Otherwise, there will be information desync.
var Guilds map[string]*Guild

// muteLock
// A map to store mutexes for handling mutes for a server synchronously.
var muteLock = make(map[string]*sync.Mutex)

// initProvider
// Stores and allows for the calling of the chosen GuildProvider.
var initProvider func() GuildProvider

// currentProvider
// A reference to a struct of functions that provides the guild info system with a database
// Or similar system to save guild data.
var currentProvider GuildProvider

// AddGuild
// Adds a guild to the storage/initializes already stored guilds.
func AddGuild(g *discordgo.Guild) *Guild {
	// If the storage provider has already loaded the guild,
	// Then lets just add the guild pointer
	if guild, ok := Guilds[g.ID]; ok {
		guild.Guild = g
		return guild
	}
	// Create a new guild with default values
	newGuild := Guild{
		Guild: g,
		Info:  NewGuildInfo(),
	}
	// Add the new guild to the map of guilds
	Guilds[g.ID] = &newGuild
	// Save the guild to .json
	// A failed save is fatal, so we can count on this being successful
	newGuild.save()
	// Log that a new guild was registered
	Log.Infof("new guild registered: %s", g.ID)
	return &newGuild

}

func GetGuild(guildID string) *Guild {
	// The command is being run as a dm, send back an empty guild object with default fields
	if guildID == "" {
		return &Guild{
			Guild: &discordgo.Guild{
				ID: "",
			},
			Info: NewGuildInfo(),
		}
	}
	if guild, ok := Guilds[guildID]; ok {
		return guild
	}
	// the guild does not exist, there must be an awful
	// f**k up if this happens
	Log.Errorf("unable to fetch guild %s", guildID)
	Log.Errorf("something bad must've really happened")
	// lets just return a bare bones guild struct
	return &Guild{
		Guild: &discordgo.Guild{
			ID: "",
		},
		Info: NewGuildInfo(),
	}

}

func GuildExists(guildID string) bool {
	if guildID == "" {
		return false
	}
	if g, ok := Guilds[guildID]; ok {
		// check to see if the guild exists via the channels thing
		if g.Guild.Channels == nil {
			return false
		}
		return true
	}
	return false
}

// loadGuilds loads the guilds from the provider.
func loadGuilds() {
	Guilds = currentProvider.Load()
}

// SetInitProvider sets the initProvider.
func SetInitProvider(provider func() GuildProvider) {
	initProvider = provider
}

//func (st *Storage[K, V]) Get(key K) (value V, ok bool) {
//	item, ok := st.storage.Get(key)
//	if !ok {
//		return
//	}
//	return item.Value, true
//}

// save the guild data to the provider.
func (g *Guild) save() {
	currentProvider.Save(g)
}

// Guild Helpers
