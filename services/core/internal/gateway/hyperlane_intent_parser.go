package gateway

import "forge/projectforge/services/core/internal/aios/hyperlane"

type HyperlaneIntent = hyperlane.Intent

func ParseHyperlaneIntent(user string) HyperlaneIntent {
	return hyperlane.ParseIntent(user)
}

func SupportsNoModelHyperlaneRoute(intent HyperlaneIntent) bool {
	return hyperlane.SupportsNoModelRoute(intent)
}
