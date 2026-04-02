package twitter

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// extractInstructionsFromUnknownPath walks the JSON tree to find "instructions" at any depth.
func extractInstructionsFromUnknownPath(raw map[string]json.RawMessage) ([]json.RawMessage, error) {
	return findInstructions(raw, 0)
}

func findInstructions(m map[string]json.RawMessage, depth int) ([]json.RawMessage, error) {
	if depth > 8 {
		return nil, fmt.Errorf("instructions not found in response")
	}

	if instrRaw, ok := m["instructions"]; ok {
		var instructions []json.RawMessage
		if err := json.Unmarshal(instrRaw, &instructions); err == nil && len(instructions) > 0 {
			return instructions, nil
		}
	}

	for _, v := range m {
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(v, &nested); err == nil {
			if result, err := findInstructions(nested, depth+1); err == nil {
				return result, nil
			}
		}
	}

	return nil, fmt.Errorf("instructions not found in response")
}

// parseInstructions extracts tweets from X's timeline instruction list.
func parseInstructions(instructions []json.RawMessage) ([]Tweet, error) {
	tweets := make([]Tweet, 0, 50)

	for _, rawInstr := range instructions {
		var instr instruction
		if err := json.Unmarshal(rawInstr, &instr); err != nil {
			continue
		}
		if instr.Type != "TimelineAddEntries" {
			continue
		}

		var addEntries addEntriesInstruction
		if err := json.Unmarshal(rawInstr, &addEntries); err != nil {
			continue
		}

		for _, entry := range addEntries.Entries {
			if strings.HasPrefix(entry.EntryID, "cursor-") {
				continue
			}

			if strings.HasPrefix(entry.EntryID, "tweet-") {
				if t, ok := extractTweetFromItem(entry.Content); ok {
					tweets = append(tweets, t)
				}
				continue
			}

			if strings.HasPrefix(entry.EntryID, "conversationthread-") || strings.HasPrefix(entry.EntryID, "profile-conversation-") {
				var mod timelineModuleContent
				if err := json.Unmarshal(entry.Content, &mod); err == nil && mod.Items != nil {
					for _, item := range mod.Items {
						if t, ok := extractFromTweetContent(item.Item.ItemContent); ok {
							tweets = append(tweets, t)
						}
					}
				}
				continue
			}

			if t, ok := extractTweetFromItem(entry.Content); ok {
				tweets = append(tweets, t)
			}
		}
	}

	return tweets, nil
}

func extractTweetFromItem(raw json.RawMessage) (Tweet, bool) {
	var item timelineItemContent
	if err := json.Unmarshal(raw, &item); err != nil {
		return Tweet{}, false
	}
	return extractFromTweetContent(item.ItemContent)
}

func extractFromTweetContent(tc tweetContent) (Tweet, bool) {
	if tc.Typename != "TimelineTweet" && tc.Typename != "" {
		return Tweet{}, false
	}
	if tc.TweetResults.Result == nil {
		return Tweet{}, false
	}

	var result tweetResult
	if err := json.Unmarshal(tc.TweetResults.Result, &result); err != nil {
		return Tweet{}, false
	}

	// TweetWithVisibilityResults wraps the real tweet one level deeper.
	if result.Typename == "TweetWithVisibilityResults" && result.Tweet != nil {
		if err := json.Unmarshal(result.Tweet, &result); err != nil {
			return Tweet{}, false
		}
	}

	if result.Typename != "Tweet" && result.Typename != "" {
		return Tweet{}, false
	}

	legacy := result.Legacy
	if legacy.IDStr == "" {
		return Tweet{}, false
	}

	views, _ := strconv.Atoi(result.Views.Count)
	ts, _ := time.Parse("Mon Jan 02 15:04:05 -0700 2006", legacy.CreatedAt)

	mediaType := ""
	hasMedia := false
	mediaList := legacy.ExtendedEntities.Media
	if len(mediaList) == 0 {
		mediaList = legacy.Entities.Media
	}
	if len(mediaList) > 0 {
		hasMedia = true
		mediaType = mediaList[0].Type
	}

	// User info lives in .legacy on home timeline, .core on user timeline.
	authorName := result.Core.UserResults.Result.Legacy.Name
	authorHandle := result.Core.UserResults.Result.Legacy.ScreenName
	if authorName == "" {
		authorName = result.Core.UserResults.Result.UserCore.Name
	}
	if authorHandle == "" {
		authorHandle = result.Core.UserResults.Result.UserCore.ScreenName
	}

	return Tweet{
		ID:           legacy.IDStr,
		Text:         legacy.FullText,
		AuthorName:   authorName,
		AuthorHandle: authorHandle,
		Likes:        legacy.FavoriteCount,
		Retweets:     legacy.RetweetCount,
		Replies:      legacy.ReplyCount,
		Quotes:       legacy.QuoteCount,
		Views:        views,
		Timestamp:    ts,
		HasMedia:     hasMedia,
		MediaType:    mediaType,
		IsVerified:   result.Core.UserResults.Result.IsBlueVerified,
		Engagement:   legacy.FavoriteCount + legacy.RetweetCount,
	}, true
}
