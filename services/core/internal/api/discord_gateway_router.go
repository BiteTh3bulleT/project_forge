package api

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

func routeDiscordTextIntent(envelope discordEventEnvelope, prefix string, botUserID string) (discordIntent, bool) {
	raw := strings.TrimSpace(envelope.RawContent)
	if raw == "" {
		return discordIntent{}, false
	}
	normalizedPrefix := strings.TrimSpace(prefix)
	if normalizedPrefix == "" {
		normalizedPrefix = "!forge"
	}
	lowerRaw := strings.ToLower(raw)
	lowerPrefix := strings.ToLower(normalizedPrefix)
	if strings.HasPrefix(lowerRaw, lowerPrefix) {
		rest := strings.TrimSpace(raw[len(normalizedPrefix):])
		return parseDiscordCommand(envelope, rest), true
	}

	if strings.TrimSpace(botUserID) != "" {
		mentionA := "<@" + strings.TrimSpace(botUserID) + ">"
		mentionB := "<@!" + strings.TrimSpace(botUserID) + ">"
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(raw, mentionA), mentionB))
		if trimmed != raw && trimmed != "" {
			intent := newDiscordIntent(envelope, discordIntentConversational, "conversation")
			intent.Content = trimmed
			intent.ArgumentText = trimmed
			return intent, true
		}
	}
	return discordIntent{}, false
}

func routeDiscordInteractionIntent(envelope discordEventEnvelope, ic *discordgo.InteractionCreate) (discordIntent, error) {
	if ic == nil || ic.Interaction == nil {
		return discordIntent{}, errDiscordMalformedPayload
	}
	if ic.Type != discordgo.InteractionApplicationCommand {
		intent := newDiscordIntent(envelope, discordIntentAutomationEvent, "interaction_event")
		intent.Content = envelope.RawContent
		return intent, nil
	}

	data := ic.ApplicationCommandData()
	commandName := strings.ToLower(strings.TrimSpace(data.Name))
	if commandName != "forge" {
		intent := newDiscordIntent(envelope, discordIntentDirectCommand, commandName)
		intent.Content = envelope.RawContent
		return intent, nil
	}

	if len(data.Options) == 0 {
		return newDiscordIntent(envelope, discordIntentSystemQuery, "help"), nil
	}

	option := data.Options[0]
	sub := strings.ToLower(strings.TrimSpace(option.Name))
	intent := parseDiscordCommand(envelope, sub)
	if option.Type == discordgo.ApplicationCommandOptionSubCommand {
		if sub == "memory" {
			intent = newDiscordIntent(envelope, discordIntentMemoryQuery, "memory_query")
		} else {
			intent = parseDiscordCommand(envelope, sub)
		}
		for _, subOption := range option.Options {
			if strings.EqualFold(subOption.Name, "query") {
				if v, ok := subOption.Value.(string); ok {
					text, err := normalizeDiscordIngressText(v)
					if err != nil {
						return discordIntent{}, err
					}
					intent.ArgumentText = text
					intent.Content = text
				}
			}
		}
	}
	intent.Metadata["slashSubcommand"] = sub
	if intent.Command == "memory_query" && intent.ArgumentText == "" {
		intent.ArgumentText = strings.TrimSpace(subOptionString(option.Options, "query"))
		intent.Content = intent.ArgumentText
	}
	return intent, nil
}

func parseDiscordCommand(envelope discordEventEnvelope, raw string) discordIntent {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return newDiscordIntent(envelope, discordIntentSystemQuery, "help")
	}
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return newDiscordIntent(envelope, discordIntentSystemQuery, "help")
	}

	head := strings.ToLower(parts[0])
	switch head {
	case "ping":
		return newDiscordIntent(envelope, discordIntentSystemQuery, "ping")
	case "status":
		return newDiscordIntent(envelope, discordIntentSystemQuery, "status")
	case "help":
		return newDiscordIntent(envelope, discordIntentSystemQuery, "help")
	case "agents":
		return newDiscordIntent(envelope, discordIntentAgentControl, "agents")
	case "memory":
		if len(parts) >= 3 && strings.EqualFold(parts[1], "query") {
			query := strings.TrimSpace(strings.Join(parts[2:], " "))
			intent := newDiscordIntent(envelope, discordIntentMemoryQuery, "memory_query")
			intent.Args = []string{"query"}
			intent.ArgumentText = query
			intent.Content = query
			return intent
		}
		intent := newDiscordIntent(envelope, discordIntentSystemQuery, "help")
		intent.Metadata["invalidCommand"] = raw
		return intent
	case "ask":
		text := strings.TrimSpace(strings.Join(parts[1:], " "))
		intent := newDiscordIntent(envelope, discordIntentConversational, "conversation")
		intent.ArgumentText = text
		intent.Content = text
		return intent
	default:
		intent := newDiscordIntent(envelope, discordIntentDirectCommand, "unknown")
		intent.ArgumentText = raw
		intent.Content = raw
		intent.Metadata["unknownCommand"] = head
		return intent
	}
}

func newDiscordIntent(envelope discordEventEnvelope, class discordIntentClass, command string) discordIntent {
	return discordIntent{
		ID:           envelope.CorrelationID + ":" + command,
		Class:        class,
		Command:      strings.TrimSpace(strings.ToLower(command)),
		Source:       envelope.Source,
		Metadata:     map[string]any{"eventType": envelope.EventType},
		Args:         nil,
		ArgumentText: "",
		Content:      "",
	}
}

func subOptionString(options []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, option := range options {
		if strings.EqualFold(option.Name, name) {
			if value, ok := option.Value.(string); ok {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}
