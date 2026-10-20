package discord

import (
	"strings"
	"sync"
	"time"

	"github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/config"

	"github.com/axrona/go-discordrpc/client"
)

// rpc is touched from UI and game-monitor goroutines; rpcMu serialises
// every access (login, activity updates, logout) so the client can never
// be logged in twice or used while Close is tearing it down.
var (
	rpc   *client.Client
	rpcMu sync.Mutex
)

// ensureLocked returns the shared client, logging it in on first use.
// Callers must hold rpcMu.
func ensureLocked() *client.Client {
	if config.GetDiscordRPCDisabled() {
		return nil
	}
	if rpc != nil {
		return rpc
	}
	c := client.NewClient("1430869336712544346")
	if err := c.Login(); err != nil {
		return nil
	}
	rpc = c
	return rpc
}

func Close() {
	rpcMu.Lock()
	defer rpcMu.Unlock()
	if rpc != nil {
		_ = rpc.Logout()
		rpc = nil
	}
}

func Init() {
	SetLauncherIdle()
}

func SetLauncherIdle() {
	rpcMu.Lock()
	defer rpcMu.Unlock()
	c := ensureLocked()
	if c == nil {
		return
	}
	now := time.Now()
	_ = c.SetActivity(client.Activity{
		Type:       0,
		State:      "Idling in Launcher",
		Details:    "A Minecraft Bedrock Launcher",
		LargeImage: "appicon",
		LargeText:  "Bedrock Nexus",
		SmallImage: "appicon",
		SmallText:  "Bedrock Nexus",
		Party:      nil,
		Timestamps: &client.Timestamps{Start: &now},
		Buttons: []*client.Button{
			{Label: "GitHub", Url: "https://github.com/BedrockNexusLauncher/BedrockNexusLauncher"},
			{Label: "Discord", Url: "https://discord.gg/v5R5P4vRZk"},
		},
	})
}

func SetPlayingVersion(version string) {
	rpcMu.Lock()
	defer rpcMu.Unlock()
	c := ensureLocked()
	if c == nil {
		return
	}
	now := time.Now()
	v := strings.TrimSpace(version)
	s := "Playing Minecraft"
	if v != "" {
		s = "Playing Minecraft " + v
	}
	_ = c.SetActivity(client.Activity{
		Type:       0,
		State:      s,
		Details:    "In-game",
		LargeImage: "",
		LargeText:  "Bedrock Nexus",
		SmallImage: "",
		SmallText:  "Bedrock Nexus",
		Party:      nil,
		Timestamps: &client.Timestamps{Start: &now},
		Buttons: []*client.Button{
			{Label: "GitHub", Url: "https://github.com/BedrockNexusLauncher/BedrockNexusLauncher"},
			{Label: "Discord", Url: "https://discord.gg/v5R5P4vRZk"},
		},
	})
}
